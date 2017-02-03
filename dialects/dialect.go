//Package dialects defines a uniform interface for creating custom support for
//different SQL databases.
package dialects

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/ngorm/ngorm/model"
)

// Dialect interface contains behaviors that differ across SQL database
type Dialect interface {
	// GetName get dialect's name
	GetName() string

	// SetDB set db for dialect
	SetDB(db model.SQLCommon)

	// BindVar return the placeholder for actual values in SQL statements, in many dbs it is "?", Postgres using $1
	BindVar(i int) string
	// Quote quotes field name to avoid SQL parsing exceptions by using a reserved word as a field name
	Quote(key string) string

	// DataTypeOf return data's sql type
	DataTypeOf(field *model.StructField) (string, error)

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

	PrimaryKey([]string) string

	QueryFieldName(string) string
}

var baseOpener *DefaultOpener

func init() {
	baseOpener = &DefaultOpener{dialects: make(map[string]Dialect)}
}

func Register(d Dialect) {
	baseOpener.RegisterDialect(d)
}

//DefaultOpener implements Opener interface.
type DefaultOpener struct {
	dialects map[string]Dialect
	mu       sync.RWMutex
}

func (d *DefaultOpener) RegisterDialect(dia Dialect) {
	d.mu.Lock()
	d.dialects[dia.GetName()] = dia
	d.mu.Unlock()
}

func (d *DefaultOpener) FindDialect(dia string) Dialect {
	d.mu.RLock()
	o := d.dialects[dia]
	d.mu.RUnlock()
	return o
}

//Open opens up database connection using the database/sql package.
func (d *DefaultOpener) Open(dialect string, args ...interface{}) (model.SQLCommon, Dialect, error) {
	var source string
	var dia Dialect
	var common model.SQLCommon
	var err error

	switch value := args[0].(type) {
	case string:
		var driver = dialect
		if len(args) == 1 {
			source = value
		} else if len(args) >= 2 {
			driver = value
			source = args[1].(string)
		}
		common, err = sql.Open(driver, source)
		if err != nil {
			return nil, nil, err
		}
	case model.SQLCommon:
		common = value
	default:
		return nil, nil, fmt.Errorf("unknown argument %v", value)
	}
	dia = d.FindDialect(dialect)
	if dia == nil {
		return nil, nil, fmt.Errorf("unsupported dialect %s", dialect)
	}
	return common, dia, nil
}

func Opener() *DefaultOpener {
	return baseOpener
}
