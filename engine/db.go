package engine

import (
	"context"

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

	Search    *model.Search
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

type DbTabler interface {
	TableName(*Engine) string
}

type Tabler interface {
	TableName() string
}
