/**
 * Author: richen
 * Date: 2020-10-14 14:20:24
 * LastEditTime: 2020-10-14 14:31:47
 * Description:
 * Copyright (c) - <richenlin(at)gmail.com>
 */
package config

import (
	"log"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/json"
	"github.com/gookit/config/v2/yaml"
	"go.uber.org/zap"
)

var (
	conf     map[string]interface{}
	confPath string
	jsonConf bool
)

// Init
func Init(fileName ...string) {
	if len(fileName) == 0 {
		confPath = "./config/config.yaml"
		jsonConf = false
	} else {
		confPath = fileName[0]
		if fileName[1] != "" {
			jsonConf = true
		}
	}
	loadConfig(confPath, jsonConf)
}

// GetConfig
func GetConfig() map[string]interface{} {
	return conf
}

// loadConfig
func loadConfig(filePath string, isJson bool) {
	// parse env
	config.WithOptions(config.ParseEnv)
	// add driver
	if isJson {
		config.AddDriver(json.Driver)
	} else {
		config.AddDriver(yaml.Driver)
	}

	// load
	err := config.LoadFiles(filePath)
	if err != nil {
		log.Fatal("read config file failed", zap.String("error", err.Error()))
	}
	conf = config.Data()
}
