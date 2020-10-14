package mysqldb

import "database/sql"

// Top-level convenience functions

var globalManager *Manager

// Init 根据基础配置 初始化全局数据库
func Init(config *Config) error {
	var err error
	globalManager, err = NewManager(config)
	return err
}

// MockInit 使用外部sqlDB 初始化全局数据库，供mock db测试时替换使用
func MockInit(writeDB *sql.DB, readDBs []*sql.DB) {
	globalManager = NewManagerFromSQLDB(writeDB, readDBs, defaultKeepalive)
}

// RDB 返回读库
func RDB() *sql.DB {
	return globalManager.RDB()
}

// WDB 返回写库
func WDB() *sql.DB {
	return globalManager.WDB()
}

// Close 关闭全局数据库(连接池及保活协程)。该函数应当很少使用到
func Close() {
	globalManager.Close()
}

// GlobalManager 返回维护的全局Manager对象
func GlobalManager() *Manager {
	return globalManager
}
