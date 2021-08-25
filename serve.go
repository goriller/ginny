package ginny

import (
	"github.com/gorillazer/ginny-serve/grpc"
	"github.com/gorillazer/ginny-serve/http"
)

// Serve
type Serve func(app *Application) error

// HttpServe
func HttpServe(svr *http.Server) Serve {
	return func(app *Application) error {
		svr.Application(app.name)
		app.httpServer = svr

		return nil
	}
}

// GrpcServe
func GrpcServe(svr *grpc.Server) Serve {
	return func(app *Application) error {
		svr.Application(app.name)
		app.grpcServer = svr
		return nil
	}
}
