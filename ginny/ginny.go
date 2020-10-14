/**
 * Author: richen
 * Date: 2020-07-13 15:22:54
 * LastEditTime: 2020-07-28 19:53:18
 * Description:
 * Copyright (c) - <richenlin(at)gmail.com>
 */
package ginny

import "github.com/gin-gonic/gin"

type Application struct {
	*gin.Engine
}

func New() *Application {
	g := gin.New()

	//g.Use(Logger(), gin.Recovery())

	return &Application{g}
}

// // handelMap
// var handelMap = make(map[string]interface{}, 100)

// /**
//  * description:
//  * param {type}
//  * return:
//  */
// func (app *Application) Config(key string, val ...interface{}) interface{} {
// 	if val == nil {
// 		return handelMap[key]
// 	}
// 	handelMap[key] = val
// 	return nil
// }

// func (app *Application) Listen() {

// }
