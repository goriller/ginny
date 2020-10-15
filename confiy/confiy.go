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

	"git.code.oa.com/linyyyang/ginny/loggy"
	"github.com/jinzhu/configor"
)

// LoadConf
func LoadConf(s interface{}, files ...string) error {
	return configor.New(&configor.Config{
		AutoReload:         true,
		AutoReloadInterval: time.Minute,
		AutoReloadCallback: func(config interface{}) {
			loggy.Warn("Config is changed")
		},
		//ErrorOnUnmatchedKeys: false,
	}).Load(s, files...)
}
