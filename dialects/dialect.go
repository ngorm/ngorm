package dialects

import (
	"database/sql"
	"reflect"
	"strconv"
	"strings"

	"github.com/gernest/ngorm/model"
)

// Dialect interface contains behaviors that differ across SQL database
type Dialect interface {
	// GetName get dialect's name
	GetName() string

	// SetDB set db for dialect
	SetDB(db *sql.DB)

	// BindVar return the placeholder for actual values in SQL statements, in many dbs it is "?", Postgres using $1
	BindVar(i int) string
	// Quote quotes field name to avoid SQL parsing exceptions by using a reserved word as a field name
	Quote(key string) string
	// DataTypeOf return data's sql type
	DataTypeOf(field *model.StructField) string

	// HasIndex check has index or not
	HasIndex(tableName string, indexName string) bool
	// HasForeignKey check has foreign key or not
	HasForeignKey(tableName string, foreignKeyName string) bool
	// RemoveIndex remove index
	RemoveIndex(tableName string, indexName string) error
	// HasTable check has table or not
	HasTable(tableName string) bool
	// HasColumn check has column or not
	HasColumn(tableName string, columnName string) bool

	// LimitAndOffsetSQL return generated SQL with Limit and Offset, as mssql has special case
	LimitAndOffsetSQL(limit, offset interface{}) string
	// SelectFromDummyTable return select values, for most dbs, `SELECT values` just works, mysql needs `SELECT value FROM DUAL`
	SelectFromDummyTable() string
	// LastInsertIdReturningSuffix most dbs support LastInsertId, but postgres needs to use `RETURNING`
	LastInsertIDReturningSuffix(tableName, columnName string) string

	// BuildForeignKeyName returns a foreign key name for the given table, field and reference
	BuildForeignKeyName(tableName, field, dest string) string

	// CurrentDatabase return current database name
	CurrentDatabase() string
}

// ParseFieldStructForDialect parse field struct for dialect
func ParseFieldStructForDialect(field *model.StructField) (fieldValue reflect.Value, sqlType string, size int, additionalType string) {
	// Get redirected field type
	var reflectType = field.Struct.Type
	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	// Get redirected field value
	fieldValue = reflect.Indirect(reflect.New(reflectType))

	// Get scanner's real value
	var getScannerValue func(reflect.Value)
	getScannerValue = func(value reflect.Value) {
		fieldValue = value
		if _, isScanner := reflect.New(fieldValue.Type()).Interface().(sql.Scanner); isScanner && fieldValue.Kind() == reflect.Struct {
			getScannerValue(fieldValue.Field(0))
		}
	}
	getScannerValue(fieldValue)

	// Default Size
	if num, ok := field.TagSettings["SIZE"]; ok {
		size, _ = strconv.Atoi(num)
	} else {
		size = 255
	}

	// Default type from tag setting
	additionalType = field.TagSettings["NOT NULL"] + " " + field.TagSettings["UNIQUE"]
	if value, ok := field.TagSettings["DEFAULT"]; ok {
		additionalType = additionalType + " DEFAULT " + value
	}

	return fieldValue, field.TagSettings["TYPE"], size, strings.TrimSpace(additionalType)
}
