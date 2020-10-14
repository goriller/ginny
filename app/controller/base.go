package controller

type IController interface {
	New() IController
}

type Controller struct {
	app interface{}
}

func (c *Controller) New() IController {
	panic("not implements")
}

func New(controller Controller) interface{} {
	return controller.New()
}