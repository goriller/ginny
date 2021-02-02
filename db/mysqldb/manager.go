package mysqldb

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"git.code.oa.com/Ginny/ginny/logg"
	_ "github.com/go-sql-driver/mysql"
)

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

	readDBs := make([]*sql.DB, 0, len(config.RDBs))
	for i := 0; i < len(config.RDBs); i++ {
		readDB, err := newDB(&config.RDBs[i], config)
		if err != nil {
			return nil, err
		}
		readDBs = append(readDBs, readDB)
	}

	return NewManagerFromSQLDB(writeDB, readDBs, time.Duration(config.Keepalive)*time.Second), nil
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
		logg.Error(fmt.Sprintf("close db write pool error: %s", err.Error()))
	}
	for i := 0; i < len(m.readDBs); i++ {
		if err := m.readDBs[i].Close(); err != nil {
			logg.Error(fmt.Sprintf("close db read pool error: %s", err.Error()))
		}
	}
}

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

// 默认的keepalive间隔 3h
const defaultKeepalive = 3 * 60 * 60 * time.Second

// 定时ping db 保持连接激活
func keepalive(ctx context.Context, db *sql.DB, interval time.Duration) {
	if interval.Nanoseconds() == 0 {
		interval = defaultKeepalive
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			logg.Info("keepalive db end")
			return
		case <-ticker.C:
			if err := db.Ping(); err != nil {
				logg.Error(fmt.Sprintf("keepalive db ping error: %s", err.Error()))
			}
		}
	}
}

//QueryRowContext executes a query that is expected to return at most one row.
//Use it when transaction is necessary
func (m *Manager) QueryRowContext(c context.Context, query string, args ...interface{}) *sql.Row {
	ctx, tx := GetTrans(c)
	if tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return m.RDB().QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query that doesn't return rows.
// For example: an INSERT and UPDATE.
// Use it when transaction is necessary
func (m *Manager) ExecContext(c context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, tx := GetTrans(c)
	if tx != nil {
		return tx.ExecContext(ctx, query, args...)
	}
	return m.WDB().ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows, typically a SELECT.
// Use it when transaction is necessary
func (m *Manager) QueryContext(c context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, tx := GetTrans(c)
	if tx != nil {
		return tx.QueryContext(ctx, query, args)
	}
	return m.RDB().QueryContext(ctx, query, args...)
}

// PrepareContext creates a prepared statement for  later queries or executions..
// Use it when transaction is necessary
func (m *Manager) PrepareContext(c context.Context, query string) (*sql.Stmt, error) {
	ctx, tx := GetTrans(c)
	if tx != nil {
		return tx.PrepareContext(ctx, query)
	}
	return m.WDB().PrepareContext(ctx, query)
}

// Stmt returns a transaction-specific prepared statement from an existing statement.
//Use it when transaction is necessary
func (m *Manager) Stmt(c context.Context, stmt *sql.Stmt) *sql.Stmt {
	_, tx := GetTrans(c)
	return tx.Stmt(stmt)
}
