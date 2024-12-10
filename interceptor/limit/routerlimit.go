// Package routerlimit provide router limit tools
package limit

import (
	"fmt"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

// RouterLimit 路由频次限制
type RouterLimit struct {
	// 默认限制：路由下匹配 header key 做限制
	Limit []Limit
	// 设置黑名单：当 header key value 匹配某规则时，则禁止访问
	Block []KV
	// 默认频次控制
	Default Default
	// 关闭频次限制
	Disabled bool
}

const (
	// NoLimit 无限制
	NoLimit int = -1
	// Block 禁止
	Block int = 0
)

// Match 匹配频次限制规则
func (r *RouterLimit) Match(path string, header Getter) *LimitValue {
	if r.Disabled {
		return &LimitValue{
			Quota: NoLimit,
		}
	}
	// 黑名单检查
	for _, kv := range r.Block {
		headerValue := header.Get(kv.Key)
		if headerValue == kv.Value {
			return &LimitValue{
				Quota:   Block,
				Message: fmt.Sprintf("%s [ %s ] is in the blacklist", getKeyName(kv.Key), headerValue),
			}
		}
	}

	for _, v := range r.Limit {
		if !strings.HasPrefix(path, v.Prefix) {
			continue
		}
		return getQuota(v, path, header)
	}
	return getQuota(Limit{
		Headers:  r.Default.Headers,
		Quota:    r.Default.Quota,
		Duration: r.Default.Duration,
	}, path, header)
}

func getQuota(quotaLimit Limit, path string, header Getter) *LimitValue {
	if quotaLimit.Quota < 0 {
		return &LimitValue{
			Quota: NoLimit,
		}
	}
	limitValue := &LimitValue{
		Quota:    quotaLimit.Quota,
		Duration: quotaLimit.Duration,
		Key:      "",
	}
	var targetHeaderKey string
	for _, headerKey := range quotaLimit.Headers {
		headerValue := header.Get(headerKey)
		if headerValue != "" {
			limitValue.Key += "." + headerValue
			targetHeaderKey += headerKey
		}
	}
	if limitValue.Key == "" {
		return &LimitValue{
			Quota: NoLimit,
		}
	}
	limitValue.Message = fmt.Sprintf("trace key %s, limit key %s", targetHeaderKey, limitValue.Key)
	return limitValue
}

// Limit data
type Limit struct {
	Prefix   string
	Headers  []string
	Quota    int
	Duration time.Duration
}

// Default the default limit
type Default struct {
	Headers  []string
	Quota    int
	Duration time.Duration
}

// LimitValue the limit value
type LimitValue struct {
	// 频次限制key
	Key string
	// 频次限制提示消息
	Message string
	// Duration 周期(向桶中放置 Token 的间隔)
	Duration time.Duration
	// Quota 配额
	Quota int
}

func getKeyName(key string) string {
	if strings.HasPrefix(key, "x-") {
		return key[2:]
	}
	return key
}

// KV kv
type KV struct {
	Key   string
	Value string
}

// MatchMap 匹配Map类型
func (r *RouterLimit) MatchMap(path string, data logging.Fields) *LimitValue {
	return r.Match(path, mapGetter(data))
}

// MatchHeader 匹配http header 类型
func (r *RouterLimit) MatchHeader(path string, data map[string][]string) *LimitValue {
	return r.Match(path, headerGetter(data))
}

// Getter the getter interface for map or http header
type Getter interface {
	Get(key string) string
}

// mapGetter getter from logging.Ite
type mapGetter logging.Fields

// Get implement map value
func (m mapGetter) Get(key string) string {
	i := m.Iterator()
	for i.Next() {
		k, _ := i.At()
		existing[k] = struct{}{}
	}
	return m[key]
}

// headerGetter map implements for header
type headerGetter map[string][]string

// Get implement header value
func (m headerGetter) Get(key string) string {
	if value := m[key]; len(value) > 0 {
		return value[0]
	}
	return ""
}
