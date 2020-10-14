/**
 * Author: richen
 * Date: 2020-07-13 15:22:54
 * LastEditTime: 2020-07-28 19:53:18
 * Description:
 * Copyright (c) - <richenlin(at)gmail.com>
 */
package ginny

import (
	"git.code.oa.com/linyyyang/ginny/logger"
	"git.code.oa.com/linyyyang/ginny/middleware"
	"github.com/gin-gonic/gin"
)

type Application struct {
	*gin.Engine
	*gin.Context
}

func New(userMiddlewares ...gin.HandlerFunc) *Application {
	engine := gin.New()
	engine.Use(middleware.BenchmarkLog(), middleware.Recovery(logger.DefaultLogger, true),
		middleware.Trace())
	engine.Use(userMiddlewares...)

	return &Application{engine, &gin.Context{}}
}
