/**
 * Author: richen
 * Date: 2020-07-13 11:19:57
 * LastEditTime: 2020-07-29 15:15:24
 * Description:
 * Copyright (c) - <richenlin(at)gmail.com>
 */
package controller

import (
	"fmt"
)

type IHelloController interface {
	IController
	Hello()
}

type HelloController struct {
	*Controller
}

func (c *HelloController) Hello() {
	fmt.Println("hello !!")
}
