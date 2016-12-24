package search

import (
	"fmt"

	"github.com/gernest/gorm/base"
	"github.com/gernest/gorm/engine"
	"github.com/gernest/gorm/regexes"
)

func Where(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.WhereConditions = append(e.WhereConditions, map[string]interface{}{"query": query, "args": values})
	return e
}

func Not(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.NotConditions = append(e.NotConditions, map[string]interface{}{"query": query, "args": values})
	return e
}

func Or(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.OrConditions = append(e.OrConditions, map[string]interface{}{"query": query, "args": values})
	return e
}

func Attr(e *engine.Engine, attrs ...interface{}) *engine.Engine {
	e.InitAttrs = append(e.InitAttrs, toSearchableMap(attrs...))
	return e
}

func toSearchableMap(attrs ...interface{}) (result interface{}) {
	if len(attrs) > 1 {
		if str, ok := attrs[0].(string); ok {
			result = map[string]interface{}{str: attrs[1]}
		}
	} else if len(attrs) == 1 {
		if attr, ok := attrs[0].(map[string]interface{}); ok {
			result = attr
		}

		if attr, ok := attrs[0].(interface{}); ok {
			result = attr
		}
	}
	return
}

func Assign(e *engine.Engine, attrs ...interface{}) *engine.Engine {
	e.AssignAttrs = append(e.AssignAttrs, toSearchableMap(attrs...))
	return e
}

func Select(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	if regexes.DistinctSQL.MatchString(fmt.Sprint(query)) {
		e.IgnoreOrderQuery = true
	}
	e.Selects = map[string]interface{}{"query": query, "args": values}
	return e
}

func Omit(e *engine.Engine, columns ...string) *engine.Engine {
	e.Omits = columns
	return e
}

func Limit(e *engine.Engine, limit interface{}) *engine.Engine {
	e.Limit = limit
	return e
}

func Offset(e *engine.Engine, offset interface{}) *engine.Engine {
	e.Offset = offset
	return e
}

func Group(e *engine.Engine, query interface{}) *engine.Engine {
	s, err := base.GetInterfaceAsSQL(query)
	if err != nil {
		e.Error = err
	} else {
		e.Group = s
	}
	return e
}

func Having(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.HavingConditions = append(e.HavingConditions, map[string]interface{}{"query": query, "args": values})
	return e
}
