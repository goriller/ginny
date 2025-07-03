package mux

import (
	"context"
	"net/http"

	"github.com/goriller/ginny/middleware"
	"github.com/goriller/ginny/server/mux/rewriter"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	// "github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/protobuf/proto"
)

func defaultErrorHandler(ctx context.Context,
	mux *runtime.ServeMux, marshaler runtime.Marshaler,
	w http.ResponseWriter, r *http.Request, err error,
) {
	rewriter.WriteHTTPErrorResponse(w, r, err)
}

// handlerWithMiddleWares handler with middle wares.
func handlerWithMiddleWares(h http.Handler, middleWares ...middleware.MuxMiddleware) http.Handler {
	lenMiddleWare := len(middleWares)
	for i := lenMiddleWare - 1; i >= 0; i-- {
		middleWare := middleWares[i]
		h = middleWare(h)
	}
	return h
}

// // HandlerGRPCService
// func HandlerGRPCService(mux *runtime.ServeMux, server interface{}, v grpc.MethodDesc) func(w http.ResponseWriter,
// 	r *http.Request, _ map[string]string) {
// 	return func(w http.ResponseWriter, req *http.Request, _ map[string]string) {
// 		ctx, cancel := context.WithCancel(req.Context())
// 		defer cancel()

// 		var stream runtime.ServerTransportStream
// 		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)

// 		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
// 		resp, md, err := handlerGRPCRequest(ctx, inboundMarshaler, server, req, v.MethodName)
// 		if err != nil {
// 			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
// 			return
// 		}

// 		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
// 		ctx = runtime.NewServerMetadataContext(ctx, md)
// 		if err != nil {
// 			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
// 			return
// 		}
// 		runtime.ForwardResponseMessage(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
// 	}
// }

// // handlerGRPCRequest
// func handlerGRPCRequest(ctx context.Context, marshaler runtime.Marshaler,
// 	server interface{}, req *http.Request, methodName string,
// ) (proto.Message, runtime.ServerMetadata, error) {
// 	var metadata runtime.ServerMetadata
// 	newReader, berr := utilities.IOReaderFactory(req.Body)
// 	if berr != nil {
// 		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", berr)
// 	}
// 	method := reflect.ValueOf(server).MethodByName(methodName)
// 	if !method.IsValid() {
// 		return nil, metadata, status.Errorf(codes.Unimplemented, "method %s is is valid", methodName)
// 	}
// 	methodType := method.Type()
// 	if methodType.NumIn() != 2 || methodType.NumOut() != 2 {
// 		return nil, metadata, status.Errorf(codes.Unimplemented, "method %s may not a unary service", methodName)
// 	}
// 	callParams := make([]reflect.Value, 2)
// 	callParams[0] = reflect.ValueOf(ctx)
// 	in := reflect.New(methodType.In(1).Elem())
// 	protoReq := in.Interface()
// 	if err := marshaler.NewDecoder(newReader()).Decode(protoReq); err != nil && !errors.Is(err, io.EOF) {
// 		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
// 	}
// 	callParams[1] = in
// 	resps := method.Call(callParams)
// 	if err := resps[1].Interface(); err != nil {
// 		return nil, metadata, err.(error)
// 	}
// 	msg := resps[0].Interface()
// 	return msg.(proto.Message), metadata, nil
// }

// forwardResponseOptionFunc
func forwardResponseOptionFunc(ctx context.Context, w http.ResponseWriter, message proto.Message) error {
	if body, ok := message.(*httpbody.HttpBody); ok {
		if body.ContentType == typeLocation {
			location := string(body.Data)
			w.Header().Set(typeLocation, location)
			body.ContentType = "text/html; charset=utf-8"
			w.Header().Set("Content-Type", body.ContentType)
			w.WriteHeader(http.StatusFound)
			body.Data = []byte("<a href=\"" + htmlReplacer.Replace(location) + "\">Found</a>.\n")
		}
	}

	return nil
}
