package engine

import "context"

// DB contains information for current db connection
type Engine struct {
	Value             interface{}
	Error             error
	RowsAffected      int64
	parent            *Engine
	logMode           int
	singularTable     bool
	source            string
	values            map[string]interface{}
	blockGlobalUpdate bool
	ctx               context.Context

	// search conditions
	WhereConditions  []map[string]interface{}
	OrConditions     []map[string]interface{}
	NotConditions    []map[string]interface{}
	HavingConditions []map[string]interface{}
	joinConditions   []map[string]interface{}
	InitAttrs        []interface{}
	AssignAttrs      []interface{}
	Selects          map[string]interface{}
	Omits            []string
	orders           []interface{}
	preload          []SearchPreload
	Offset           interface{}
	Limit            interface{}
	Group            string
	tableName        string
	raw              bool
	Unscoped         bool
	IgnoreOrderQuery bool
}

type SearchPreload struct {
	schema     string
	conditions []interface{}
}
