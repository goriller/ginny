package mysql

import (
	"context"
	"database/sql"
)

// IQuery
type IQuery interface {
	QueryContext(c context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(c context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(c context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(c context.Context, query string) (*sql.Stmt, error)
	Stmt(c context.Context, stmt *sql.Stmt) *sql.Stmt
}

// Query
type Query struct {
	Manager *Manager
}

//QueryRowContext executes a query that is expected to return at most one row.
//Use it when transaction is necessary
func (q *Query) QueryRowContext(c context.Context, query string, args ...interface{}) *sql.Row {
	ctx, tx := GetTrans(c)
	if tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return q.Manager.RDB().QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query that doesn't return rows.
// For example: an INSERT and UPDATE.
// Use it when transaction is necessary
func (q *Query) ExecContext(c context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, tx := GetTrans(c)
	if tx != nil {
		return tx.ExecContext(ctx, query, args...)
	}
	return q.Manager.WDB().ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows, typically a SELECT.
// Use it when transaction is necessary
func (q *Query) QueryContext(c context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, tx := GetTrans(c)
	if tx != nil {
		return tx.QueryContext(ctx, query, args)
	}
	return q.Manager.RDB().QueryContext(ctx, query, args...)
}

// PrepareContext creates a prepared statement for  later queries or executions..
// Use it when transaction is necessary
func (q *Query) PrepareContext(c context.Context, query string) (*sql.Stmt, error) {
	ctx, tx := GetTrans(c)
	if tx != nil {
		return tx.PrepareContext(ctx, query)
	}
	return q.Manager.WDB().PrepareContext(ctx, query)
}

// Stmt returns a transaction-specific prepared statement from an existing statement.
//Use it when transaction is necessary
func (q *Query) Stmt(c context.Context, stmt *sql.Stmt) *sql.Stmt {
	_, tx := GetTrans(c)
	return tx.Stmt(stmt)
}
