package mysql

import (
	"context"
	"database/sql"
)

// key
type dbTrans struct{}

// 从ctx获取事务上下文
func GetTrans(ctx context.Context) (context.Context, *sql.Tx) {
	if value, ok := ctx.Value(dbTrans{}).(*sql.Tx); ok {
		return ctx, value
	}
	return ctx, nil
}

// 创建事务上下文
func NewTrans(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, dbTrans{}, tx)
}

// 执行事务
func Transaction(c context.Context, m *Manager, fn func(ctx context.Context) error) error {
	var err error
	ctx, tx := GetTrans(c)
	if tx == nil {
		//tx为空时初始化一个tx，然后放到context, 事务隔离级别为driver默认
		tx, err = m.WDB().Begin()
		if err != nil {
			return err
		}
		ctx = NewTrans(c, tx)
	}

	defer func() {
		if err == nil {
			_ = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()

	err = fn(ctx)

	if err != nil {
		return err
	}

	return nil
}
