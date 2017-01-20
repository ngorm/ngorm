// Package engine  defines the core structure that drives the ngorm API.
package engine

import (
	"context"
	"time"

	"github.com/gernest/ngorm/dialects"
	"github.com/gernest/ngorm/logger"
	"github.com/gernest/ngorm/model"
)

//Engine is the driving force for ngorm. It contains, Scope, Search and other
//utility properties for easily building complex SQL queries.
type Engine struct {
	RowsAffected int64

	//When this field is set to true. The table names will not be pluarized.
	//The default behaviour is to plurarize table names e.g Order struct will
	//give orders table name.
	SingularTable bool
	Ctx           context.Context
	Dialect       dialects.Dialect

	Search    *model.Search
	Scope     *model.Scope
	StructMap *model.SafeStructsMap
	SQLDB     model.SQLCommon
	Log       *logger.Zapper

	Now func() time.Time
}

//AddError adds err to Engine.Error.
//
// THis is here until I refactor all the APIs to return errors instead of
// patching the Engine with arbitrary erros
func (e *Engine) AddError(err error) error {
	return nil
}

//DBTabler is an interface for getting database table name from the *Engine
type DBTabler interface {
	TableName(*Engine) string
}

//Tabler interface for defining table name
type Tabler interface {
	TableName() string
}
