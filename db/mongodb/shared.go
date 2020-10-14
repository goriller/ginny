package mongodb

import (
	"go.mongodb.org/mongo-driver/mongo"
)

// Top-level convenience functions

var globalManager *Manager

// Init 根据基础配置 初始化全局mongodb deployment连接管理器
func Init(config *Config) error {
	var err error
	globalManager, err = NewManager(config)
	return err
}

// Client 返回客户端连接池
func Client() *mongo.Client {
	return globalManager.Client()
}

// DB 返回默认业务库
func DB() *mongo.Database {
	return globalManager.DB()
}

// Database 获取给定dbname的业务库
func Database(dbname string) *mongo.Database {
	return globalManager.Database(dbname)
}

// Close 关闭全局管理器(释放连接池资源等)。该函数应当很少使用到
func Close() {
	globalManager.Close()
}

// GlobalManager 返回维护的全局Manager对象
func GlobalManager() *Manager {
	return globalManager
}
