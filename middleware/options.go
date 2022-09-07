package middleware

import "net/http"

// 常用的header key
var (
	RequestId            = "request_id"
	RequestIDHeader      = "x-request-id"
	DeviceIDHeader       = "x-device-id"
	PathHeader           = "x-request-path"
	MethodHeader         = "x-request-method"
	TraceidHeader        = "x-b3-traceid"
	SpanidHeader         = "x-b3-spanid"
	ParentspanidHeader   = "x-b3-parentspanid"
	SampledHeader        = "x-b3-sampled"
	FlagsHeader          = "x-b3-flags"
	SpanContextHeader    = "x-ot-span-context"
	ResponseStatusHeader = "x-response-status"

	HeaderMap = map[string]string{
		RequestIDHeader:    RequestId,
		DeviceIDHeader:     "device_id",
		TraceidHeader:      "traceid",
		SpanidHeader:       "spanid",
		SpanidHeader:       "spanid",
		ParentspanidHeader: "parentspanid",
		SampledHeader:      "sampled",
		FlagsHeader:        "flags",
		SpanContextHeader:  "span_context",
	}
)

// MuxMiddleware
type MuxMiddleware func(http.Handler) http.HandlerFunc
