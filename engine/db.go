package engine

import (
	"context"
	"database/sql"

	"github.com/gernest/ngorm/dialects"
	"github.com/gernest/ngorm/model"
)

// DB contains information for current db connection
type Engine struct {
	Value             interface{}
	Error             error
	RowsAffected      int64
	Parent            *Engine
	logMode           int
	SingularTable     bool
	source            string
	values            map[string]interface{}
	blockGlobalUpdate bool
	ctx               context.Context
	Dialect           dialects.Dialect

	Search    *Search
	Scope     *model.Scope
	StructMap *model.SafeModelStructsMap

	SQL *SQL
}

type SQL struct {
	Query  string
	Values []interface{}
}

func (e *Engine) AddError(err error) error {
	return nil
}

type Search struct {
	WhereConditions  []map[string]interface{}
	OrConditions     []map[string]interface{}
	NotConditions    []map[string]interface{}
	HavingConditions []map[string]interface{}
	JoinConditions   []map[string]interface{}
	InitAttrs        []interface{}
	AssignAttrs      []interface{}
	Selects          map[string]interface{}
	Omits            []string
	Orders           []interface{}
	Preload          []SearchPreload
	Offset           interface{}
	Limit            interface{}
	Group            string
	TableName        string
	Raw              bool
	Unscoped         bool
	IgnoreOrderQuery bool
}

type SearchPreload struct {
	Schema     string
	Conditions []interface{}
}

type DbTabler interface {
	TableName(*Engine) string
}

type Tabler interface {
	TableName() string
}

type SQLCommon interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}
