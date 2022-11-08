package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/goriller/ginny/middleware"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.uber.org/zap"
	"golang.org/x/net/context/ctxhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	// DefaultTimeout
	DefaultTimeout = time.Second * 10
	// DefaultRetryTimes 如果请求失败，最多重试3次
	DefaultRetryTimes = 3
	// DefaultRetryDelay 在重试前，延迟等待100毫秒
	DefaultRetryDelay = time.Millisecond * 100
	RequestIDHeader   = "X-Request-Id"
)

// ClientOptions
type ClientOptions struct {
	target                string // ip+port/path
	logger                *zap.Logger
	tracer                opentracing.Tracer
	resolver              Resolver
	timeout               time.Duration
	retryTimes            int
	protoJSONMarshaller   *protojson.MarshalOptions
	protoJSONUnmarshaller *protojson.UnmarshalOptions
}

// ClientOptional
type ClientOptional func(o *ClientOptions)

// WithReqestTimeout
func WithReqestTimeout(t time.Duration) ClientOptional {
	return func(opt *ClientOptions) {
		if t > 0 {
			opt.timeout = t
		}
	}
}

// WithRetryTimes 设置失败重试
func WithRetryTimes(retryTimes int) ClientOptional {
	return func(opt *ClientOptions) {
		if retryTimes > 0 {
			opt.retryTimes = retryTimes
		}
	}
}

// WithLogger
func WithLogger(logger *zap.Logger) ClientOptional {
	return func(o *ClientOptions) {
		o.logger = logger
	}
}

// WithTracer
func WithTracer(tracer opentracing.Tracer) ClientOptional {
	return func(o *ClientOptions) {
		o.tracer = tracer
	}
}

// WithResolver
func WithResolver(resolver Resolver) ClientOptional {
	return func(o *ClientOptions) {
		o.resolver = resolver
	}
}

// Client
type Client struct {
	client  *http.Client
	options *ClientOptions
}

// NewClient
func NewClient(ctx context.Context, uri string, opts ...ClientOptional) (*Client, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse uri %s error for %w", uri, err)
	}
	opt, err := parseOptions(ctx, u, opts...)
	if err != nil {
		return nil, err
	}

	cli, err := newClientConn(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &Client{
		client:  cli,
		options: opt,
	}, nil

}

// newClientConn
func newClientConn(ctx context.Context, o *ClientOptions) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 100, //默认是2
	}

	return &http.Client{
		Transport: transport,
		Timeout:   0, //默认是0，无超时
	}, nil
}

// Request 自动序列化和反序列化地请求
// 请求 和 响应 支持 struct 和 string 和 []byte 三种方式
func (c *Client) Request(ctx context.Context, method,
	path string, header http.Header, reqData interface{},
	respDataPtr interface{},
) (err error) {
	start := time.Now()
	if header == nil {
		header = http.Header{}
	}
	var httpRequestURL string
	if strings.HasPrefix(path, "/") {
		httpRequestURL = c.options.target + path
	}
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		httpRequestURL = path
	}

	if c.options.tracer != nil {
		clientSpan := parseTrace(ctx, method, "httpClient-"+method, c.options.tracer)
		carrier := opentracing.HTTPHeadersCarrier(header)
		_ = clientSpan.Tracer().Inject(clientSpan.Context(), opentracing.HTTPHeaders, carrier)
		header = http.Header(carrier)
		defer clientSpan.Finish()
	}

	var (
		tryTimes int
		reqBody  []byte
		resp     *http.Response
	)

	reqBody, err = c.buildRequestBody(ctx, header, reqData)
	for i := 0; i <= c.options.retryTimes; i++ {
		resp, err = c.request(ctx, method,
			httpRequestURL, header, reqBody)
		tryTimes++
		if err == nil {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	if err != nil {
		return err
	}

	defer func() {
		resp.Body.Close()
		c.onRequestClose(ctx, method, httpRequestURL, tryTimes, start, header, resp.StatusCode, err)
	}()

	err = c.parseResponseBody(ctx, resp.Body, respDataPtr)
	if err != nil {
		return err
	}

	if resp.StatusCode > http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return status.Error(codes.Code(resp.StatusCode), "http status code is not 200")
	}
	return nil
}

func (c *Client) request(ctx context.Context, method, uri string,
	header http.Header, reqBody []byte,
) (*http.Response, error) {
	var (
		err     error
		request *http.Request
	)

	if len(reqBody) > 0 {
		request, err = http.NewRequest(method, uri, bytes.NewReader(reqBody))
	} else {
		request, err = http.NewRequest(method, uri, nil)
	}
	if err != nil {
		return nil, err
	}
	request.Header = header
	c.client.Timeout = time.Duration(c.options.timeout) * time.Second
	return ctxhttp.Do(ctx, c.client, request)
}

func (c *Client) onRequestClose(ctx context.Context,
	method, path string, tryTimes int, start time.Time,
	header http.Header, statusCode int, err error,
) {
	if c.options.logger != nil {
		used := time.Since(start)
		log := c.options.logger.With(
			zap.String("action", path),
			zap.String("host", c.options.target),
			zap.String(middleware.RequestId, header.Get(RequestIDHeader)),
			zap.Int("status", statusCode),
			zap.Int("referer", tryTimes),
			zap.String("protocol", "http/client"),
			zap.Float32("time_ms", durationToMilliseconds(used)),
		)
		if err != nil {
			log.Error(err.Error())
		} else {
			log.Info(method)
		}
	}
}

// buildRequestBody
func (c *Client) buildRequestBody(ctx context.Context,
	header http.Header, reqData interface{}) (reqBody []byte, err error) {
	if reqData == nil {
		return
	}
	switch v := reqData.(type) {
	case []byte:
		reqBody = v
	case string:
		reqBody = []byte(v)
	default:
		if protoData, ok := reqData.(proto.Message); ok {
			reqBody, err = c.options.protoJSONMarshaller.Marshal(protoData)
		} else {
			reqBody, err = json.Marshal(reqData)
		}
		if err == nil {
			header.Set("Content-Type", "application/json")
		}
	}
	return
}

// parseResponseBody
func (c *Client) parseResponseBody(ctx context.Context,
	body io.ReadCloser, respDataPtr interface{}) (err error) {
	respBody, err := ioutil.ReadAll(body)
	if respDataPtr != nil {
		switch v := respDataPtr.(type) {
		case *string:
			*v = string(respBody)
		default:
			if _, ok := respDataPtr.(proto.Message); ok {
				err = c.options.protoJSONUnmarshaller.Unmarshal(respBody,
					respDataPtr.(proto.Message))
			} else {
				err = json.Unmarshal(respBody, respDataPtr)
			}
			if err != nil {
				err = fmt.Errorf(" can not unmarshal %s to %s for %w ",
					string(respBody), reflect.TypeOf(respDataPtr), err)
			}
		}
	}
	return
}

// parseTrace
func parseTrace(ctx context.Context, method, tag string, tracer opentracing.Tracer) opentracing.Span {
	var parentCtx opentracing.SpanContext
	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentCtx = parent.Context()
	}

	clientSpan := tracer.StartSpan(
		method,
		opentracing.ChildOf(parentCtx),
		ext.SpanKindRPCClient,
		opentracing.Tag{Key: string(ext.Component), Value: tag},
	)
	return clientSpan
}

// parseOptions
func parseOptions(ctx context.Context, u *url.URL, options ...ClientOptional) (*ClientOptions, error) {
	o := &ClientOptions{
		timeout:    DefaultTimeout,
		retryTimes: DefaultRetryTimes,
		protoJSONMarshaller: &protojson.MarshalOptions{
			UseProtoNames: true,
		},
		protoJSONUnmarshaller: &protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}

	for _, option := range options {
		option(o)
	}
	query := u.Query()
	if query.Get("emitUnpopulated") == "true" {
		o.protoJSONMarshaller.EmitUnpopulated = true
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		o.target = u.String()
	}

	tag := query.Get("tag")
	if o.resolver != nil {
		addr, err := o.resolver(ctx, u.String(), tag)
		if err != nil {
			return nil, err
		}
		o.target = addr
	}

	return o, nil
}

// buildQuery
func buildQuery(uri string, form url.Values) (string, error) {
	if len(form) == 0 {
		return "", errors.New("form required")
	}
	target, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	urlValues := target.Query()
	for k, v := range form {
		for _, i := range v {
			urlValues.Add(k, i)
		}
	}

	target.RawQuery = urlValues.Encode()
	return target.String(), nil
}

// durationToMilliseconds
func durationToMilliseconds(duration time.Duration) float32 {
	milliseconds := float32(duration.Nanoseconds()/1000) / 1000
	return milliseconds
}
