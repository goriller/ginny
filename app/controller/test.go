package controller

import "fmt"

type ITestController interface {
	IController
}

type TestController struct {
	*Controller
}

func (c *TestController) New() IController {
	return &TestController{}
}

func (c *TestController) Test() {
	fmt.Println("test")
}
