package mux

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	reflect "reflect"
	"strconv"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	statusOKPrefix         = 2
	statusBadRequestPrefix = 4
	fallback               = `{"code": 13,"message":"failed to marshal error message"}`
)

// codesErrors some errors string for grpc codes
var codesErrors = map[codes.Code]string{
	codes.OK:                 "ok",
	codes.Canceled:           "canceled",
	codes.Unknown:            "unknown",
	codes.InvalidArgument:    "invalid_argument",
	codes.DeadlineExceeded:   "deadline_exceeded",
	codes.NotFound:           "not_found",
	codes.AlreadyExists:      "already_exists",
	codes.PermissionDenied:   "permission_denied",
	codes.ResourceExhausted:  "resource_exhausted",
	codes.FailedPrecondition: "failed_precondition",
	codes.Aborted:            "aborted",
	codes.OutOfRange:         "out_of_range",
	codes.Unimplemented:      "unimplemented",
	codes.Internal:           "internal",
	codes.Unavailable:        "unavailable",
	codes.DataLoss:           "data_loss",
	codes.Unauthenticated:    "unauthenticated",
}

// RegisterErrorCodes set custom error codes for DefaultHTTPError
// for exp: server.RegisterErrorCodes(pb.ErrorCode_name)
// SetCustomErrorCodes set custom error codes for DefaultHTTPError
// the map[int32]string is compact to protobuf's ENMU_name
// 2*** HTTP status 200
// 4*** HTTP status 400
// 5*** AND other HTTP status 500
// For exp:
// in proto
// enum CommonError {
//	captcha_required = 4001;
//	invalid_captcha = 4002;
// }
// in code
// server.RegisterErrorCodes(common.CommonError_name)
func RegisterErrorCodes(codeErrors map[int32]string) {
	for code, errorMsg := range codeErrors {
		codesErrors[codes.Code(code)] = errorMsg
	}
}

// httpStatusCode the 2xxx is 200, the 4xxx is 400, the 5xxx is 500
func httpStatusCode(code codes.Code) (httpStatusCode int) {
	// http status codes can be error codes
	if code >= 200 && code < 599 {
		return int(code)
	}
	for code >= 10 {
		code /= 10
	}
	switch code {
	case statusOKPrefix:
		httpStatusCode = http.StatusOK
	case statusBadRequestPrefix:
		httpStatusCode = http.StatusBadRequest
	default:
		httpStatusCode = http.StatusInternalServerError
	}
	return
}

// CodeToError
func CodeToError(c codes.Code) string {
	errStr, ok := codesErrors[c]
	if ok {
		return errStr
	}
	return strconv.FormatInt(int64(c), 10)
}

// CodeToStatus
func CodeToStatus(code codes.Code) int {
	st := int(code)
	if st > 100 {
		st = httpStatusCode(code)
	} else {
		st = runtime.HTTPStatusFromCode(code)
	}
	return st
}

func defaultErrorHandler(ctx context.Context,
	mux *runtime.ServeMux, marshaler runtime.Marshaler,
	w http.ResponseWriter, r *http.Request, err error,
) {
	w.Header().Del("Trailer")
	w.Header().Del("Transfer-Encoding")
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		grpclog.Infof("Failed to extract ServerMetadata from context")
	}

	// RFC 7230 https://tools.ietf.org/html/rfc7230#section-4.1.2
	// Unless the request includes a TE header field indicating "trailers"
	// is acceptable, as described in Section 4.3, a server SHOULD NOT
	// generate trailer fields that it believes are necessary for the user
	// agent to receive.

	if te := r.Header.Get("TE"); strings.Contains(strings.ToLower(te), "trailers") {
		handleForwardResponseTrailerHeader(w, md)
		w.Header().Set("Transfer-Encoding", "chunked")
		handleForwardResponseTrailer(w, md)
	}

	WriteHTTPErrorResponse(w, r, err)
}

func handleForwardResponseTrailerHeader(w http.ResponseWriter, md runtime.ServerMetadata) {
	for k := range md.TrailerMD {
		tKey := textproto.CanonicalMIMEHeaderKey(fmt.Sprintf("%s%s", runtime.MetadataTrailerPrefix, k))
		w.Header().Add("Trailer", tKey)
	}
}

func handleForwardResponseTrailer(w http.ResponseWriter, md runtime.ServerMetadata) {
	for k, vs := range md.TrailerMD {
		tKey := runtime.MetadataTrailerPrefix + k
		for _, v := range vs {
			w.Header().Add(tKey, v)
		}
	}
}

// WriteHTTPErrorResponse  set HTTP status code and write error description to the body.
func WriteHTTPErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	s, ok := status.FromError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	preTags := tags.Extract(r.Context())
	preTags.Set("code", strconv.Itoa(int(s.Code())))
	preTags.Set("message", s.Message())
	w.WriteHeader(CodeToStatus(s.Code()))

	_, e := defaultMuxOption.bodyWriter(w, r, s)
	// newErrorBytes(requestId, apiModel, s, optsWare.errorMarshaler, optsWare.newErrorBody)
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, fallback); err != nil {
			grpclog.Infof("Failed to write response: %v", err)
		}
		return
	}
}

// handlerWithMiddleWares handler with middle wares.
func handlerWithMiddleWares(h http.Handler, middleWares ...MuxMiddleware) http.Handler {
	lenMiddleWare := len(middleWares)
	for i := lenMiddleWare - 1; i >= 0; i-- {
		middleWare := middleWares[i]
		h = middleWare(h)
	}
	return h
}

// HandlerGRPCService
func HandlerGRPCService(mux *runtime.ServeMux, server interface{}, v grpc.MethodDesc) func(w http.ResponseWriter,
	r *http.Request, _ map[string]string) {
	return func(w http.ResponseWriter, req *http.Request, _ map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()

		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)

		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		resp, md, err := handlerGRPCRequest(ctx, inboundMarshaler, server, req, v.MethodName)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		runtime.ForwardResponseMessage(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	}
}

// handlerGRPCRequest
func handlerGRPCRequest(ctx context.Context, marshaler runtime.Marshaler,
	server interface{}, req *http.Request, methodName string,
) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata
	newReader, berr := utilities.IOReaderFactory(req.Body)
	if berr != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", berr)
	}
	method := reflect.ValueOf(server).MethodByName(methodName)
	if !method.IsValid() {
		return nil, metadata, status.Errorf(codes.Unimplemented, "method %s is is valid", methodName)
	}
	methodType := method.Type()
	if methodType.NumIn() != 2 || methodType.NumOut() != 2 {
		return nil, metadata, status.Errorf(codes.Unimplemented, "method %s may not a unary service", methodName)
	}
	callParams := make([]reflect.Value, 2)
	callParams[0] = reflect.ValueOf(ctx)
	in := reflect.New(methodType.In(1).Elem())
	protoReq := in.Interface()
	if err := marshaler.NewDecoder(newReader()).Decode(protoReq); err != nil && !errors.Is(err, io.EOF) {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	callParams[1] = in
	resps := method.Call(callParams)
	if err := resps[1].Interface(); err != nil {
		return nil, metadata, err.(error)
	}
	msg := resps[0].Interface()
	return msg.(proto.Message), metadata, nil
}
