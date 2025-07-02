package errs

import "reflect"

type catcher struct {
	err      error
	hasCatch bool
}

type CatchHandler interface {
	Catch(e error, handler func(err error)) CatchHandler
	CatchAll(handler func(err error)) FinalHandler
	FinalHandler
}

type FinalHandler interface {
	Finally(handlers ...func())
}

func Try(f func()) CatchHandler {
	//返回一个实现CatchHandler接口的对象
	t := &catcher{}
	defer func() {
		defer func() { //<2>
			r := recover()
			if r != nil {
				t.err = r.(error)
			}
		}()
		f() //<1>
	}()
	return t
}

//<1>RequireCatch函数有两个作用：一个是判断是否已捕捉异常，另一个是否发生了异常。如果返回false则代表没有异常，或异常已被捕捉。
func (t *catcher) RequireCatch() bool {
	if t.hasCatch { //<2>如果已经执行了catch块，就直接判断不执行
		return false
	}
	if t.err == nil { //<3>如果异常为空，则判断不执行
		return false
	}
	return true
}
func (t *catcher) Catch(e error, handler func(err error)) CatchHandler {
	if !t.RequireCatch() {
		return t
	}
	//<4>如果传入的error类型和发生异常的类型一致，则执行异常处理器，并将hasCatch修改为true代表已捕捉异常
	if reflect.TypeOf(e) == reflect.TypeOf(t.err) {
		handler(t.err)
		t.hasCatch = true
	}
	return t
}

func (t *catcher) CatchAll(handler func(err error)) FinalHandler {
	//<5>CatchAll()函数和Catch()函数都是返回同一个对象，但返回的接口类型却不一样，也就是CatchAll()之后只能调用Finally()
	if !t.RequireCatch() {
		return t
	}
	handler(t.err)
	t.hasCatch = true
	return t
}

func (t *catcher) Finally(handlers ...func()) {
	//<6>遍历处理器，并在Finally函数执行完毕之后执行
	for _, handler := range handlers {
		defer handler()
	}
	err := t.err
	//<7>如果异常不为空，且未捕捉异常，则抛出异常
	if err != nil && !t.hasCatch {
		panic(err)
	}
}
