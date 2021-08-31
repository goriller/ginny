package ginny

import (
	consul "github.com/gorillazer/ginny-consul"
	"github.com/gorillazer/ginny-serve/grpc"
	"github.com/gorillazer/ginny-serve/http"
)

// Server
type Server struct {
	HttpServer *http.Server
	GrpcServer *grpc.Server
}

// Serve
type Serve func(app *Application) error

// HttpServe
func HttpServe(svr *http.Server) Serve {
	return func(app *Application) error {
		svr.AppName(app.Name)
		app.Server = &Server{
			HttpServer: svr,
		}
		return nil
	}
}

// HttpServeWithConsul
func HttpServeWithConsul(svr *http.Server, c *consul.Client) Serve {
	return func(app *Application) error {
		svr.AppName(app.Name)
		svr.ConsulClient(c.Client)
		app.Server = &Server{
			HttpServer: svr,
		}
		return nil
	}
}

// GrpcServe
func GrpcServe(svr *grpc.Server) Serve {
	return func(app *Application) error {
		svr.AppName(app.Name)
		app.Server = &Server{
			GrpcServer: svr,
		}
		return nil
	}
}

// GrpcServeWithConsul
func GrpcServeWithConsul(svr *grpc.Server, c *consul.Client) Serve {
	return func(app *Application) error {
		svr.AppName(app.Name)
		svr.ConsulClient(c.Client)
		app.Server = &Server{
			GrpcServer: svr,
		}
		return nil
	}
}
