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
}
