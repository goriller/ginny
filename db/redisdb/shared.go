package redisdb

import (
	"github.com/go-redis/redis/v8"
)

// Top-level convenience functions

var globalManager *Manager

// Init 根据基础配置 初始化全局redis管理器
func Init(config *Config) error {
	var err error
	globalManager, err = NewManager(config)
	return err
}

// DB 返回连接池客户端
func DB() redis.UniversalClient {
	return globalManager.DB()
}

// RegisterScript 注册redis lua script：若服务用到某个脚本，需在服务初始化时注册脚本，每增加一个脚本均需注册
// kvs必须偶数个: key1 value1 key2 value2 ...。key为各自服务包内定义，可参考type scriptKeyXXX struct{} 空结构体。
//   value是相应脚本*redisdb.Script=redisdb.NewScript()
func RegisterScript(kvs ...interface{}) error {
	return globalManager.RegisterScript(kvs)
}

// Script 根据脚本key取脚本结构
// 调用方保证：需在服务初始化时 调用RegisterScript()注册相应的键值对，对未注册的key 本函数将返回nil
func Script(key interface{}) *redis.Script {
	return globalManager.Script(key)
}

// Close 关闭全局管理器(释放连接池资源等)。该函数应当很少使用到
func Close() {
	globalManager.Close()
}

// GlobalManager 返回维护的全局Manager对象
func GlobalManager() *Manager {
	return globalManager
}
