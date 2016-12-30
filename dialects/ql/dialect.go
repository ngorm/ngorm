package ql

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/gernest/ngorm/model"
)

type QL struct {
	name string
	db   model.SQLCommon
}

func Memory() *QL {
	return &QL{name: "ql-mem"}
}

func File() *QL {
	return &QL{name: "ql"}
}

// GetName get dialect's name
func (q *QL) GetName() string {
	return q.name
}

// SetDB set db for dialect
func (q *QL) SetDB(db model.SQLCommon) {
	q.db = db
}

// BindVar return the placeholder for actual values in SQL statements, in many dbs it is "?", Postgres using $1
func (q QL) BindVar(i int) string {
	return fmt.Sprintf("$%d", i)
}

// Quote quotes field name to avoid SQL parsing exceptions by using a reserved word as a field name
func (a *QL) Quote(key string) string {
	return fmt.Sprintf(`"%s"`, key)
}

// DataTypeOf return data's sql type
func (q *QL) DataTypeOf(field *model.StructField) string {
	var dataValue, sqlType, _, additionalType = model.ParseFieldStructForDialect(field)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "boolean"
		case reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Float32,
			reflect.Float64,
			reflect.String:
			sqlType = dataValue.Kind().String()
		case reflect.Struct:
			switch dataValue.Interface().(type) {
			case time.Time:
				sqlType = "time"
			case big.Int:
				sqlType = "bigint"
			case big.Rat:
				sqlType = "bigrat"
			}
		default:
			if _, ok := dataValue.Interface().([]byte); ok {
				sqlType = "blob"
			}
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for ql", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

// HasIndex check has index or not
func (q *QL) HasIndex(tableName string, indexName string) bool {
	querry := "select count() from __Index where Name=$1  && TableName=$2"
	var count int
	q.db.QueryRow(querry, indexName, tableName).Scan(&count)
	return count > 0
}

// HasForeignKey check has foreign key or not
func (q *QL) HasForeignKey(tableName string, foreignKeyName string) bool {
	return false
}

// RemoveIndex remove index
func (q *QL) RemoveIndex(tableName string, indexName string) error {
	tx, err := q.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(fmt.Sprintf("DROP INDEX %v", indexName))
	if err != nil {
		return err
	}
	return tx.Commit()
}

// HasTable check has table or not
func (q *QL) HasTable(tableName string) bool {
	querry := "select count() from __Table where Name=$1"
	var count int
	q.db.QueryRow(querry, tableName).Scan(&count)
	return count > 0
}

// HasColumn check has column or not
func (q *QL) HasColumn(tableName string, columnName string) bool {
	querry := "select count() from __Column where Name=$1  && TableName=$2"
	var count int
	q.db.QueryRow(querry, columnName, tableName).Scan(&count)
	return count > 0
}

// LimitAndOffsetSQL return generated SQL with Limit and Offset, as mssql has special case
func (q *QL) LimitAndOffsetSQL(limit, offset interface{}) string {
	return ""
}

// SelectFromDummyTable return select values, for most dbs, `SELECT values` just works, mysql needs `SELECT value FROM DUAL`
func (q *QL) SelectFromDummyTable() string {
	return ""
}

// LastInsertIdReturningSuffix most dbs support LastInsertId, but postgres needs to use `RETURNING`
func (q *QL) LastInsertIDReturningSuffix(tableName, columnName string) string {
	return ""
}

// BuildForeignKeyName returns a foreign key name for the given table, field and reference
func (q *QL) BuildForeignKeyName(tableName, field, dest string) string {
	return ""
}

// CurrentDatabase return current database name
func (q *QL) CurrentDatabase() string {
	return ""
}
