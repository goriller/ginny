package main

import (
	"net/http"
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
)

type info struct {
	Name   string `json:"name"`
	Mobile string
	Cardno string
}

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")
	app.Use(recover.New())
	app.Use(logger.New())

	fileServer := app.StaticHandler("/static", false, true)

	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
		path := r.URL.Path
		if !strings.Contains(path, ".") {
			//如果它不是资源，那么就像正常情况一样继续使用路由器. <-- IMPORTANT
			router(w, r)
			return
		}
		ctx := app.ContextPool.Acquire(w, r)
		fileServer(ctx)
		app.ContextPool.Release(ctx)
	})
	type info struct {
		Aa string
		Bb string
	}
	app.Get("/", func(ctx iris.Context) {
		Info := info{
			Aa: "1",
			Bb: "2",
		}
		ctx.JSON(Info)
	})
	app.Run(iris.Addr(":8000"), iris.WithoutServerError(iris.ErrServerClosed))
}
