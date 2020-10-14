package main

import (
	. "Ginny/ginny"
	"errors"
	"time"
)

func test() error {
	return errors.New("aaa")
}

var err error

func main() {
	e := test()

	DefaultLogger.Trace(time.Now(), e)
	DefaultLogger.Info("aaa")
	DefaultLogger.Warn("aaa")
	DefaultLogger.Error("aaa", err)

	conf := NewConfig(&Options{Feeder()})
}
