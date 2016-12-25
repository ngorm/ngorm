package engine

import (
	"context"

	"github.com/gernest/gorm/dialects"
	"github.com/gernest/gorm/model"
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
	Scope     *Scope
	StructMap *model.SafeModelStructsMap
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
	orders           []interface{}
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

type Scope struct {
	Value           interface{}
	SQL             string
	SQLVars         []interface{}
	InstanceID      string
	PrimaryKeyField *model.Field
	SkipLeft        bool
	Fields          *[]*model.Field
	SelectAttrs     *[]string
}

type DbTabler interface {
	TableName(*Engine) string
}

type Tabler interface {
	TableName() string
}
