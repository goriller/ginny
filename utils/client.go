package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	_httpPool *http.Client
	_poolOnce sync.Once
)

//主要自定义了 MaxIdleConnsPerHost 和 Timeout 两个参数
func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 100, //默认是2
		},
		Timeout: 5 * time.Second, //默认是0，无超时
	}
}

func httpClient() *http.Client {
	_poolOnce.Do(func() {
		_httpPool = NewHTTPClient()
	})
	return _httpPool
}

//HTTPRequest http调用
func HTTPRequest(method, path string, bodyData interface{}, header map[string]string, rsp interface{}) ([]byte, error) {
	var body io.Reader
	if bodyData != nil {
		bodyRaw, err := json.Marshal(bodyData)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(bodyRaw)
	}

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	for k, v := range header {
		req.Header.Set(k, v)
	}

	httpRsp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer httpRsp.Body.Close()

	bin, err := ioutil.ReadAll(httpRsp.Body)
	if err != nil {
		return nil, err
	}
	if httpRsp.StatusCode != 200 {
		return bin, fmt.Errorf("Status Code:%v", httpRsp.StatusCode)
	}
	if rsp != nil {
		err = json.Unmarshal(bin, rsp)
		if err != nil {
			return bin, err
		}
	}
	return bin, nil
}

//HTTPPost Post请求
func HTTPPost(path string, bodyData interface{}, header map[string]string, rsp interface{}) ([]byte, error) {
	return HTTPRequest(http.MethodPost, path, bodyData, header, rsp)
}

//HTTPGet Get请求
func HTTPGet(path string, header map[string]string, rsp interface{}) ([]byte, error) {
	return HTTPRequest(http.MethodGet, path, nil, header, rsp)
}
