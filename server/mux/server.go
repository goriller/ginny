package mux

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
)

// MuxServe the custom serve mux that implement grpc MuxServe to simplify the http restful.
type MuxServe struct {
	serveMux *runtime.ServeMux
	opts     *MuxOption
	handler  http.Handler
}

// NewMuxServe allocates and returns a new MuxServe.
func NewMuxServe(logger *zap.Logger, opts ...Optional) *MuxServe {
	o := fullOptions(logger, opts...)

	mux := &MuxServe{
		opts: o,
	}
	mux.serveMux = runtime.NewServeMux(o.runTimeOpts...)

	// middleWares
	var middlewares = []func(http.Handler) http.Handler{
		TraceMiddleWare,
	}
	if len(o.middleWares) > 0 {
		middlewares = append(middlewares, o.middleWares...)
	}
	mux.handler = handlerWithMiddleWares(mux.serveMux, middlewares...)

	return mux
}

// ServeMux return grpc gateway server mux
func (srv *MuxServe) ServeMux() *runtime.ServeMux {
	return srv.serveMux
}

// Handle handle http path
func (srv *MuxServe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.handler.ServeHTTP(w, r)
}

// Handle handle http path
func (srv *MuxServe) Handle(method, path string, h runtime.HandlerFunc) {
	err := srv.serveMux.HandlePath(method, path, h)
	if err != nil {
		panic(err)
	}
}
