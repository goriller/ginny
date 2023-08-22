package logging

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc/codes"
)

var (
	defaultOptions = &options{
		shouldLog: DefaultLoggingDeciderMethod,
		levelFunc: DefaultCodeToLevel,
	}
)

// PayloadDecision defines rules for enabling Request or Response logging
type PayloadDecision struct {
	Enable     logging.LoggableEvent
	Request    bool
	Response   bool
	ClearBytes bool
}

type options struct {
	levelFunc                  logging.CodeToLevel
	shouldLog                  Decider
	requestFieldExtractorFunc  RequestFieldExtractorFunc
	responseFieldExtractorFunc ResponseFieldExtractorFunc
}

// Option the Options for this module
type Option func(*options)

// WithDecider customizes the function for deciding if the gRPC interceptor logs should log.
func WithDecider(f Decider) Option {
	return func(o *options) {
		o.shouldLog = f
	}
}

// WithLevels customizes the function for mapping gRPC return codes and interceptor log level statements.
func WithLevels(f logging.CodeToLevel) Option {
	return func(o *options) {
		o.levelFunc = f
	}
}

// WithResponseFieldExtractorFunc customizes the function for extracting log fields from protobuf messages, for
// unary and server-streamed methods only.
func WithResponseFieldExtractorFunc(f ResponseFieldExtractorFunc) Option {
	return func(o *options) {
		o.responseFieldExtractorFunc = f
	}
}

// WithRequestFieldExtractorFunc customizes the function for extracting log fields from protobuf messages, for
// unary and server-streamed methods only.
func WithRequestFieldExtractorFunc(f RequestFieldExtractorFunc) Option {
	return func(o *options) {
		o.requestFieldExtractorFunc = f
	}
}

// RequestFieldExtractorFunc is a user-provided function that extracts field information from a gRPC request.
// It is called from tags middleware on arrival of unary request or a server-stream request.
// Keys and values will be added to the context tags of the request. If there are no fields, you should return a nil.
type RequestFieldExtractorFunc func(fullMethod string, req interface{}) map[string]string

// Decider function defines rules for suppressing any interceptor logs
type Decider func(fullMethod string, err error) PayloadDecision

// DefaultLoggingDeciderMethod is the default implementation of decider to see if you should log the call
// by default this if always true so all calls are logged
func DefaultLoggingDeciderMethod(_ string, _ error) PayloadDecision {
	return PayloadDecision{
		Enable:   logging.FinishCall,
		Request:  false,
		Response: false,
	}
}

func evaluateOpt(opts []Option) *options {
	optCopy := &options{}
	*optCopy = *defaultOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

// ResponseFieldExtractorFunc is a user-provided function that extracts field information from a gRPC response.
// It is called from tags middleware on arrival of unary request or a server-stream request.
// Keys and values will be added to the context tags of the request. If there are no fields, you should return a nil.
type ResponseFieldExtractorFunc func(fullMethod string, resp interface{}) map[string]string

const (
	statusOKPrefix         = 2
	statusBadRequestPrefix = 4
)

// DefaultCodeToLevel the default grpc status code to log level
func DefaultCodeToLevel(code codes.Code) logging.Level {
	if code < 100 {
		switch code {
		case codes.OK, codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.ResourceExhausted,
			codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.PermissionDenied, codes.Unauthenticated:
			return logging.LevelInfo
		case codes.DeadlineExceeded, codes.Unavailable, codes.DataLoss, codes.Unimplemented:
			return logging.LevelWarn
		case codes.Unknown, codes.Internal:
			return logging.LevelError
		default:
			return logging.LevelWarn
		}
	}
	for code >= 10 {
		code /= 10
	}
	switch code {
	case statusOKPrefix:
		return logging.LevelInfo
	case statusBadRequestPrefix:
		return logging.LevelWarn
	default:
		return logging.LevelError
	}
}
