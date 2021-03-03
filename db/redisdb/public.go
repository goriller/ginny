package redisdb

import (
	"context"
	"time"
)

// RDB
type RDB struct {
	Prefix string
}

// IRDB
type IRDB interface {
	Get(ctx context.Context, k string) (string, error)
	Set(ctx context.Context, k, v string, ex time.Duration) error
	Del(ctx context.Context, k string) error
	TTL(ctx context.Context, k string) (t time.Duration, err error)
	Expire(ctx context.Context, k string, ex time.Duration) error
	Exists(ctx context.Context, k string) (ex int64, err error)
	Incr(ctx context.Context, k string) (ex int64, err error)
	Decr(ctx context.Context, k string) (ex int64, err error)
	IncrBy(ctx context.Context, k string, step int64) (ex int64, err error)
	DecrBy(ctx context.Context, k string, step int64) (ex int64, err error)
	HGet(ctx context.Context, k, f string) (v string, err error)
	HSet(ctx context.Context, k, f, v string, ex time.Duration) error
	HDel(ctx context.Context, k, f string) error
	HExists(ctx context.Context, k, f string) (ex bool, err error)
	HLen(ctx context.Context, k string) (ex int64, err error)
	HIncrBy(ctx context.Context, k, f string, step int64) (ex int64, err error)
	RPush(ctx context.Context, k, v string) error
	LPush(ctx context.Context, k, v string) error
	LPop(ctx context.Context, k string) (ex string, err error)
	RPop(ctx context.Context, k string) (ex string, err error)
	LRange(ctx context.Context, k string, start, stop int64) (ex []string, err error)
	SAdd(ctx context.Context, k string, member interface{}) (ex int64, err error)
	SCard(ctx context.Context, k string) (ex int64, err error)
	SIsMember(ctx context.Context, k string, member interface{}) (ex bool, err error)
	SetNx(ctx context.Context, k string, v interface{}, ex time.Duration) error
	SPop(ctx context.Context, k string) (ex string, err error)
	SRem(ctx context.Context, k string) error
}

// NewRDB
func NewRDB(prefix string) IRDB {
	return &RDB{
		Prefix: prefix,
	}
}

// Get get
func (r *RDB) Get(ctx context.Context, k string) (string, error) {
	k = r.Prefix + k
	v, err := DB().Get(ctx, k).Result()
	if err != nil {
		return "", err
	}
	return v, nil
}

// Set set
func (r *RDB) Set(ctx context.Context, k, v string, ex time.Duration) error {
	k = r.Prefix + k
	if err := DB().Set(ctx, k, v, ex).Err(); err != nil {
		return err
	}
	return nil
}

// Del del
func (r *RDB) Del(ctx context.Context, k string) error {
	k = r.Prefix + k
	if err := DB().Del(ctx, k).Err(); err != nil {
		return err
	}
	return nil
}

// TTL 返回给定 key 的剩余生存时间
func (r *RDB) TTL(ctx context.Context, k string) (t time.Duration, err error) {
	k = r.Prefix + k
	if t, err = DB().TTL(ctx, k).Result(); err != nil {
		return 0, err
	}
	return t, nil
}

// Expire 设置key超时属性
func (r *RDB) Expire(ctx context.Context, k string, ex time.Duration) error {
	k = r.Prefix + k
	if err := DB().Expire(ctx, k, ex).Err(); err != nil {
		return err
	}
	return nil
}

// Exists 判断key是否存在
func (r *RDB) Exists(ctx context.Context, k string) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().Exists(ctx, k).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// Incr 键自增
func (r *RDB) Incr(ctx context.Context, k string) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().Incr(ctx, k).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// Decr 键自减
func (r *RDB) Decr(ctx context.Context, k string) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().Decr(ctx, k).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// IncrBy 键按照步长自增
func (r *RDB) IncrBy(ctx context.Context, k string, step int64) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().IncrBy(ctx, k, step).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// DecrBy 键按照步长自减
func (r *RDB) DecrBy(ctx context.Context, k string, step int64) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().DecrBy(ctx, k, step).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// HGet hget
func (r *RDB) HGet(ctx context.Context, k, f string) (v string, err error) {
	k = r.Prefix + k
	if v, err = DB().HGet(ctx, k, f).Result(); err != nil {
		return "", err
	}
	return v, nil
}

// HSet hset
func (r *RDB) HSet(ctx context.Context, k, f, v string, ex time.Duration) error {
	k = r.Prefix + k
	if err := DB().HSet(ctx, k, f, v, ex).Err(); err != nil {
		return err
	}
	return nil
}

// HDel hdel
func (r *RDB) HDel(ctx context.Context, k, f string) error {
	k = r.Prefix + k
	if err := DB().HDel(ctx, k, f).Err(); err != nil {
		return err
	}
	return nil
}

// HExists HExists
func (r *RDB) HExists(ctx context.Context, k, f string) (ex bool, err error) {
	k = r.Prefix + k
	if ex, err = DB().HExists(ctx, k, f).Result(); err != nil {
		return false, err
	}
	return ex, nil
}

// HLen HLen
func (r *RDB) HLen(ctx context.Context, k string) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().HLen(ctx, k).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// HIncrBy HIncrBy
func (r *RDB) HIncrBy(ctx context.Context, k, f string, step int64) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().HIncrBy(ctx, k, f, step).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// LLen LLen
func (r *RDB) LLen(ctx context.Context, k string) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().LLen(ctx, k).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// RPush RPush
func (r *RDB) RPush(ctx context.Context, k, v string) error {
	k = r.Prefix + k
	if err := DB().RPush(ctx, k, v).Err(); err != nil {
		return err
	}
	return nil
}

// LPush LPush
func (r *RDB) LPush(ctx context.Context, k, v string) error {
	k = r.Prefix + k
	if err := DB().LPush(ctx, k, v).Err(); err != nil {
		return err
	}
	return nil
}

// LPop LPop
func (r *RDB) LPop(ctx context.Context, k string) (ex string, err error) {
	k = r.Prefix + k
	if ex, err = DB().LPop(ctx, k).Result(); err != nil {
		return "", err
	}
	return ex, nil
}

// RPop RPop
func (r *RDB) RPop(ctx context.Context, k string) (ex string, err error) {
	k = r.Prefix + k
	if ex, err = DB().RPop(ctx, k).Result(); err != nil {
		return "", err
	}
	return ex, nil
}

// LRange LRange
func (r *RDB) LRange(ctx context.Context, k string, start, stop int64) (ex []string, err error) {
	k = r.Prefix + k
	if ex, err = DB().LRange(ctx, k, start, stop).Result(); err != nil {
		return nil, err
	}
	return ex, nil
}

// SAdd SAdd
func (r *RDB) SAdd(ctx context.Context, k string, member interface{}) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().SAdd(ctx, k, member).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// SCard 返回集合的基数(集合中元素的数量)
func (r *RDB) SCard(ctx context.Context, k string) (ex int64, err error) {
	k = r.Prefix + k
	if ex, err = DB().SCard(ctx, k).Result(); err != nil {
		return 0, err
	}
	return ex, nil
}

// SIsMember SIsMember
func (r *RDB) SIsMember(ctx context.Context, k string, member interface{}) (ex bool, err error) {
	k = r.Prefix + k
	if ex, err = DB().SIsMember(ctx, k, member).Result(); err != nil {
		return false, err
	}
	return ex, nil
}

// SetNx setnx
func (r *RDB) SetNx(ctx context.Context, k string, v interface{}, ex time.Duration) error {
	k = r.Prefix + k
	if err := DB().SetNX(ctx, k, v, ex).Err(); err != nil {
		return err
	}
	return nil
}

// SPop 移除并返回集合中的一个随机元素
func (r *RDB) SPop(ctx context.Context, k string) (ex string, err error) {
	k = r.Prefix + k
	if ex, err = DB().SPop(ctx, k).Result(); err != nil {
		return "", err
	}
	return ex, nil
}

// SRem 移除集合 key 中的一个 member 元素
func (r *RDB) SRem(ctx context.Context, k string) error {
	k = r.Prefix + k
	if err := DB().SRem(ctx, k).Err(); err != nil {
		return err
	}
	return nil
}
