package scope

import (
	"github.com/gernest/gorm/base"
	"github.com/gernest/gorm/engine"
	"github.com/gernest/gorm/search"
)

// Scope contain current operation's information when you perform any operation on the database
type Scope struct {
	Search          *search.Search
	Value           interface{}
	SQL             string
	SQLVars         []interface{}
	db              *engine.Engine
	instanceID      string
	primaryKeyField *base.Field
	skipLeft        bool
	fields          *[]*base.Field
	selectAttrs     *[]string
}

////////////////////////////////////////////////////////////////////////////////
// Scope DB
////////////////////////////////////////////////////////////////////////////////

// DB return scope's DB connection
func (scope *Scope) DB() *engine.Engine {
	return scope.db
}

// HasError check if there are any error
func (scope *Scope) HasError() bool {
	return scope.db.Error != nil
}

// SkipLeft skip remaining callbacks
func (scope *Scope) SkipLeft() {
	scope.skipLeft = true
}

// PrimaryKeyValue get the primary key's value
