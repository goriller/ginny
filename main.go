package main

import (
	"errors"

	"git.code.oa.com/linyyyang/ginny/ginny"
)

func test() error {
	return errors.New("aaa")
}

var err error

func main() {
	app := ginny.New()

	app.Use()
}
