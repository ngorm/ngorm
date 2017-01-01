// Package engine  defines the core structure that drives the ngorm API.
package engine

import (
	"context"

	"github.com/gernest/ngorm/dialects"
	"github.com/gernest/ngorm/model"
)

//Engine is the driving force for ngorm. It contains, Scope, Search and other
//utility properties for easily building complex SQL queries.
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
}

//AddError adds err to Engine.Error.
//
// THis is here until I refactor all the APIs to return errors instead of
// patching the Engine with arbitrary erros
func (e *Engine) AddError(err error) error {
	return nil
}

//DbTabler is an interface for getting database table name from the *Engine
type DbTabler interface {
	TableName(*Engine) string
}

//Tabler interface for defining table name
type Tabler interface {
	TableName() string
}
