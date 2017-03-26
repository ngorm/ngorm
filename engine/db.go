// Package engine  defines the core structure that drives the ngorm API.
package engine

import (
	"context"
	"time"

	"github.com/ngorm/ngorm/dialects"
	"github.com/ngorm/ngorm/model"
)

//Engine is the driving force for ngorm. It contains, Scope, Search and other
//utility properties for easily building complex SQL queries.
//
// This acts as context, allowing passing values around. For fuc
type Engine struct {
	RowsAffected int64

	//When this field is set to true. The table names will not be pluralized.
	//The default behavior is to pluralize table names e.g Order struct will
	//give orders table name.
	SingularTable bool
	Ctx           context.Context
	Dialect       dialects.Dialect

	Search    *model.Search
	Scope     *model.Scope
	StructMap *model.SafeStructsMap
	SQLDB     model.SQLCommon

	Now func() time.Time
}

//AddError adds err to Engine.Error.
//
// THis is here until I refactor all the APIs to return errors instead of
// patching the Engine with arbitrary errors
func (e *Engine) AddError(err error) error {
	return nil
}

// Clone returns a new copy of engine
func (e *Engine) Clone() *Engine {
	return &Engine{
		Scope:         model.NewScope(),
		Search:        &model.Search{},
		SingularTable: e.SingularTable,
		Ctx:           e.Ctx,
		Dialect:       e.Dialect,
		StructMap:     e.StructMap,
		SQLDB:         e.SQLDB,
	}
}

//DBTabler is an interface for getting database table name from the *Engine
type DBTabler interface {
	TableName(*Engine) string
}

//Tabler interface for defining table name
type Tabler interface {
	TableName() string
}
