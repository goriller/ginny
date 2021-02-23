package ginny

import (
	"errors"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/gorillazer/ginny/logg"
	"github.com/gorillazer/ginny/middleware"
)

// Application
type Application struct {
	*gin.Engine
}

// New
func New(userMiddlewares ...gin.HandlerFunc) *Application {
	engine := gin.New()
	engine.Use(middleware.BenchmarkLog(), middleware.Recovery(logg.DefaultLogger, true),
		middleware.Trace())
	engine.Use(userMiddlewares...)
	// NoRoute
	engine.NoRoute(func(ctx *gin.Context) {
		ResponseNotFound(ctx, errors.New("not found"))
	})
	// NoMethod
	engine.NoMethod(func(ctx *gin.Context) {
		ResponseNotFound(ctx, errors.New("not found"))
	})

	return &Application{engine}
}

// Listen 支持优雅重启
func Listen(host, port, mode string, router *Application) error {
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	err := endless.ListenAndServe(host+":"+port, router)
	if err != nil {
		return err
	}
	return nil
}
