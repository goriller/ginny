package mysqldb

import (
	"context"
	"errors"
	"fmt"

	"github.com/gorillazer/ginny/logg"
	"github.com/didi/gendry/builder"
	"github.com/didi/gendry/scanner"
)

func init() {
	scanner.SetTagName("json")
}

// Query by native sql
func Query(ctx context.Context, m *Manager, sqlStr string, bindMap map[string]interface{}, entity interface{}) error {
	var err error
	cond, val, err := builder.NamedQuery(sqlStr, bindMap)
	if err != nil {
		return err
	}
	logg.Info(fmt.Sprintf("%v, %v", cond, val))
	return querySql(ctx, m, cond, val, entity)
}

//Find gets one record from table by condition "where"
func Find(ctx context.Context, m *Manager, entity interface{}, table string, where map[string]interface{}, selectFields ...[]string) error {
	if table == "" {
		return errors.New("table name couldn't be empty")
	}
	var field []string
	if len(selectFields) > 0 {
		field = selectFields[0]
	} else {
		field = nil
	}
	// limit
	if where == nil {
		where = map[string]interface{}{}
	}
	where["_limit"] = []uint{0, 1}
	cond, val, err := builder.BuildSelect(table, where, field)
	if nil != err {
		return err
	}
	logg.Info(fmt.Sprintf("%v, %v", cond, val))

	return querySql(ctx, m, cond, val, entity)
}

//FindAll gets multiple records from table by condition "where"
func FindAll(ctx context.Context, m *Manager, entity interface{}, table string, where map[string]interface{}, selectFields ...[]string) error {
	if table == "" {
		return errors.New("table name couldn't be empty")
	}
	var field []string
	if len(selectFields) > 0 {
		field = selectFields[0]
	} else {
		field = nil
	}
	cond, val, err := builder.BuildSelect(table, where, field)
	if nil != err {
		return err
	}
	logg.Info(fmt.Sprintf("%v, %v", cond, val))

	return querySql(ctx, m, cond, val, entity)
}

// Execute by native sql
func Execute(ctx context.Context, m *Manager, sqlStr string, bindMap map[string]interface{}) (int64, error) {
	var err error
	cond, val, err := builder.NamedQuery(sqlStr, bindMap)
	if err != nil {
		return 0, err
	}
	logg.Info(fmt.Sprintf("%v, %v", cond, val))

	return execSql(ctx, m, cond, val)
}

//Insert inserts an array of data into table
func Insert(ctx context.Context, m *Manager, table string, data []map[string]interface{}) (int64, error) {
	if table == "" {
		return 0, errors.New("table name couldn't be empty")
	}
	cond, val, err := builder.BuildInsert(table, data)
	if nil != err {
		return 0, err
	}
	logg.Info(fmt.Sprintf("%v, %v", cond, val))

	return execSql(ctx, m, cond, val)
}

//Update updates the table COLUMNS
func Update(ctx context.Context, m *Manager, table string, where, data map[string]interface{}) (int64, error) {
	if table == "" {
		return 0, errors.New("table name couldn't be empty")
	}
	cond, val, err := builder.BuildUpdate(table, where, data)
	if err != nil {
		return 0, err
	}
	logg.Info(fmt.Sprintf("%v, %v", cond, val))

	return execSql(ctx, m, cond, val)
}

// Delete deletes matched records in COLUMNS
func Delete(ctx context.Context, m *Manager, table string, where map[string]interface{}) (int64, error) {
	if table == "" {
		return 0, errors.New("table name couldn't be empty")
	}
	cond, val, err := builder.BuildDelete(table, where)
	if err != nil {
		return 0, err
	}
	logg.Info(fmt.Sprintf("%v, %v", cond, val))

	return execSql(ctx, m, cond, val)
}

// querySql
func querySql(ctx context.Context, m *Manager, cond string, val []interface{}, entity interface{}) error {
	stmt, err := m.RDB().PrepareContext(ctx, cond)
	if err != nil {
		return err
	}
	rows, err := stmt.QueryContext(ctx, val...)
	if nil != err || nil == rows {
		return err
	}
	defer func() {
		if stmt != nil {
			stmt.Close()
		}
	}()
	return scanner.ScanClose(rows, entity)
}

// execSql
func execSql(ctx context.Context, m *Manager, cond string, val []interface{}) (int64, error) {
	stmt, err := m.WDB().PrepareContext(ctx, cond)
	if err != nil {
		return 0, err
	}
	result, err := stmt.ExecContext(ctx, val...)
	if nil != err || nil == result {
		return 0, err
	}
	defer func() {
		if stmt != nil {
			stmt.Close()
		}
	}()
	return result.RowsAffected()
}
