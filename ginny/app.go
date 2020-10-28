/**
 * Author: richen
 * Date: 2020-07-13 15:22:54
 * LastEditTime: 2020-07-28 19:53:18
 * Description:
 * Copyright (c) - <richenlin(at)gmail.com>
 */
package ginny

import (
	"errors"

	"git.code.oa.com/Ginny/ginny/logiy"
	"git.code.oa.com/Ginny/ginny/middleware"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
)

// Application
type Application struct {
	*gin.Engine
}

// New
func New(userMiddlewares ...gin.HandlerFunc) *Application {
	engine := gin.New()
	engine.Use(middleware.BenchmarkLog(), gin.Logger(), middleware.Recovery(logiy.DefaultLogger, true),
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
