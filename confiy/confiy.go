/**
 * Author: richen
 * Date: 2020-10-14 14:20:24
 * LastEditTime: 2020-10-14 14:31:47
 * Description:
 * Copyright (c) - <richenlin(at)gmail.com>
 */
package confiy

import (
	"time"

	"git.code.oa.com/Ginny/ginny/logg"
	"github.com/jinzhu/configor"
)

var (
	confDriver *configor.Configor
)

// Confiy
type Confiy struct {
}

// Init
func Init() *Confiy {
	confDriver = configor.New(&configor.Config{
		AutoReload:         true,
		AutoReloadInterval: time.Minute,
		AutoReloadCallback: func(config interface{}) {
			logg.Warn("Config is changed")
		},
		//ErrorOnUnmatchedKeys: false,
	})
	return &Confiy{}
}

// LoadConf
func (c *Confiy) LoadConf(s interface{}, files ...string) error {
	return confDriver.Load(s, files...)
}
