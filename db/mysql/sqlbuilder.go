package mysql

import (
	"context"
	"errors"
	"log"

	"github.com/didi/gendry/builder"
	"github.com/didi/gendry/scanner"
	"github.com/google/wire"
)

func init() {
	scanner.SetTagName("json")
}

// SqlBuilderProvider
var SqlBuilderProvider = wire.NewSet(NewSqlBuilder, wire.Bind(new(ISqlBuilder), new(*SqlBuilder)))

// ISqlBuilder
type ISqlBuilder interface{}

// SqlBuilder
type SqlBuilder struct {
	Query *Query
}

// NewSqlBuilder
func NewSqlBuilder(config *Config) *SqlBuilder {
	mgr, err := NewManager(config)
	if err != nil {
		log.Fatalf("mysql manager error: %s", err.Error())
	}
	return &SqlBuilder{
		Query: &Query{
			Manager: mgr,
		},
	}
}

// SqlQuery by native sql
func (s *SqlBuilder) SqlQuery(ctx context.Context, sqlStr string, bindMap map[string]interface{}, entity interface{}) error {
	var err error
	cond, val, err := builder.NamedQuery(sqlStr, bindMap)
	if err != nil {
		return err
	}
	log.Printf("%v, %v", cond, val)
	return s.querySql(ctx, cond, val, entity)
}

//Find gets one record from table by condition "where"
func (s *SqlBuilder) Find(ctx context.Context, entity interface{}, table string, where map[string]interface{}, selectFields ...[]string) error {
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
	log.Printf("%v, %v", cond, val)

	return s.querySql(ctx, cond, val, entity)
}

//FindAll gets multiple records from table by condition "where"
func (s *SqlBuilder) FindAll(ctx context.Context, entity interface{}, table string, where map[string]interface{}, selectFields ...[]string) error {
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
	log.Printf("%v, %v", cond, val)

	return s.querySql(ctx, cond, val, entity)
}

// Execute by native sql
func (s *SqlBuilder) Execute(ctx context.Context, sqlStr string, bindMap map[string]interface{}) (int64, error) {
	var err error
	cond, val, err := builder.NamedQuery(sqlStr, bindMap)
	if err != nil {
		return 0, err
	}
	log.Printf("%v, %v", cond, val)

	return s.execSql(ctx, cond, val)
}

//Insert inserts an array of data into table
func (s *SqlBuilder) Insert(ctx context.Context, table string, data []map[string]interface{}) (int64, error) {
	if table == "" {
		return 0, errors.New("table name couldn't be empty")
	}
	cond, val, err := builder.BuildInsert(table, data)
	if nil != err {
		return 0, err
	}
	log.Printf("%v, %v", cond, val)

	return s.execSql(ctx, cond, val)
}

//Update updates the table COLUMNS
func (s *SqlBuilder) Update(ctx context.Context, table string, where, data map[string]interface{}) (int64, error) {
	if table == "" {
		return 0, errors.New("table name couldn't be empty")
	}
	cond, val, err := builder.BuildUpdate(table, where, data)
	if err != nil {
		return 0, err
	}
	log.Printf("%v, %v", cond, val)

	return s.execSql(ctx, cond, val)
}

// Delete deletes matched records in COLUMNS
func (s *SqlBuilder) Delete(ctx context.Context, table string, where map[string]interface{}) (int64, error) {
	if table == "" {
		return 0, errors.New("table name couldn't be empty")
	}
	cond, val, err := builder.BuildDelete(table, where)
	if err != nil {
		return 0, err
	}
	log.Printf("%v, %v", cond, val)

	return s.execSql(ctx, cond, val)
}

// querySql
func (s *SqlBuilder) querySql(ctx context.Context, cond string, val []interface{}, entity interface{}) error {
	stmt, err := s.Query.Manager.RDB().PrepareContext(ctx, cond)
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
	err = scanner.ScanClose(rows, entity)
	if err != nil && err.Error() != "[scanner]: empty result" {
		return err
	}
	return nil
}

// execSql
func (s *SqlBuilder) execSql(ctx context.Context, cond string, val []interface{}) (int64, error) {
	stmt, err := s.Query.Manager.WDB().PrepareContext(ctx, cond)
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
