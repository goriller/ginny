package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql" // init mysql driver
)

// 默认的keepalive间隔 3h
const defaultKeepalive = 3 * 60 * 60 * time.Second

// Manager 数据库管理器 读写分离 仅对同一业务库
type Manager struct {
	writeDB *sql.DB
	readDBs []*sql.DB

	ctx    context.Context //控制keepalive goroutine结束
	cancel context.CancelFunc
}

// NewManager 根据基础配置 初始化数据库
func NewManager(config *Config) (*Manager, error) {
	writeDB, err := newDB(&config.WDB, config)
	if err != nil {
		return nil, err
	}
	// RDB多个
	readDBs := make([]*sql.DB, 0, len(config.RDBs))
	for i := 0; i < len(config.RDBs); i++ {
		source := &Source{
			Host: config.RDBs[i].Host,
			User: config.RDBs[i].User,
			Pass: config.RDBs[i].Pass,
		}
		readDB, err := newDB(source, config)
		if err != nil {
			return nil, err
		}
		readDBs = append(readDBs, readDB)
	}

	return NewManagerFromSQLDB(writeDB, readDBs, time.Duration(config.Keepalive)*time.Second), nil
}

// RDB 随机返回一个读库
func (m *Manager) RDB() *sql.DB {
	return m.readDBs[rand.Intn(len(m.readDBs))]
}

// WDB 返回唯一写库
func (m *Manager) WDB() *sql.DB {
	return m.writeDB
}

// Close 关闭所有读写连接池，停止keepalive保活协程。该函数应当很少使用到
func (m *Manager) Close() {
	m.cancel()
	if err := m.writeDB.Close(); err != nil {
		log.Fatal(fmt.Sprintf("close db write pool error: %s", err.Error()))
	}
	for i := 0; i < len(m.readDBs); i++ {
		if err := m.readDBs[i].Close(); err != nil {
			log.Fatal(fmt.Sprintf("close db read pool error: %s", err.Error()))
		}
	}
}

//
func newDB(source *Source, config *Config) (*sql.DB, error) {
	// user:pass@tcp(ip:port)/dbname
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		source.User, source.Pass, source.Host, config.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(config.MaxOpenConn)
	db.SetMaxIdleConns(config.MaxIdleConn)
	db.SetConnMaxLifetime(time.Duration(config.MaxLifetime) * time.Second)
	return db, db.Ping()
}

// NewManagerFromSQLDB 根据SqlDB对象 初始化数据库
func NewManagerFromSQLDB(writeDB *sql.DB, readDBs []*sql.DB, keepaliveInterval time.Duration) *Manager {
	rand.Seed(time.Now().Unix())

	ctx, cancel := context.WithCancel(context.Background())
	go keepalive(ctx, writeDB, keepaliveInterval)
	for i := 0; i < len(readDBs); i++ {
		go keepalive(ctx, readDBs[i], keepaliveInterval)
	}

	return &Manager{
		writeDB: writeDB,
		readDBs: readDBs,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// 定时ping db 保持连接激活
func keepalive(ctx context.Context, db *sql.DB, interval time.Duration) {
	if interval.Nanoseconds() == 0 {
		interval = defaultKeepalive
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			log.Print("keepalive db end")
			return
		case <-ticker.C:
			if err := db.Ping(); err != nil {
				log.Print(fmt.Errorf("keepalive db ping error: %s", err.Error()))
			}
		}
	}
}
