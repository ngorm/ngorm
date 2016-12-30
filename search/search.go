package search

import (
	"fmt"

	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/model"
	"github.com/gernest/ngorm/regexes"
	"github.com/gernest/ngorm/util"
)

func Where(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.Search.WhereConditions = append(e.Search.WhereConditions, map[string]interface{}{"query": query, "args": values})
	return e
}

func Not(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.Search.NotConditions = append(e.Search.NotConditions, map[string]interface{}{"query": query, "args": values})
	return e
}

func Or(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.Search.OrConditions = append(e.Search.OrConditions, map[string]interface{}{"query": query, "args": values})
	return e
}

func Attr(e *engine.Engine, attrs ...interface{}) *engine.Engine {
	e.Search.InitAttrs = append(e.Search.InitAttrs, toSearchableMap(attrs...))
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
	e.Search.AssignAttrs = append(e.Search.AssignAttrs, toSearchableMap(attrs...))
	return e
}

func Order(e *engine.Engine, value interface{}, reorder ...bool) *engine.Engine {
	if len(reorder) > 0 && reorder[0] {
		e.Search.Orders = []interface{}{}
	}

	if value != nil {
		e.Search.Orders = append(e.Search.Orders, value)
	}
	return e
}

func Select(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	if regexes.DistinctSQL.MatchString(fmt.Sprint(query)) {
		e.Search.IgnoreOrderQuery = true
	}
	e.Search.Selects = map[string]interface{}{"query": query, "args": values}
	return e
}

func Omit(e *engine.Engine, columns ...string) *engine.Engine {
	e.Search.Omits = columns
	return e
}

func Limit(e *engine.Engine, limit interface{}) *engine.Engine {
	e.Search.Limit = limit
	return e
}

func Offset(e *engine.Engine, offset interface{}) *engine.Engine {
	e.Search.Offset = offset
	return e
}

func Group(e *engine.Engine, query interface{}) *engine.Engine {
	s, err := util.GetInterfaceAsSQL(query)
	if err != nil {
		e.Error = err
	} else {
		e.Search.Group = s
	}
	return e
}

func Having(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.Search.HavingConditions = append(e.Search.HavingConditions, map[string]interface{}{"query": query, "args": values})
	return e
}

func Join(e *engine.Engine, query interface{}, values ...interface{}) *engine.Engine {
	e.Search.JoinConditions = append(e.Search.JoinConditions, map[string]interface{}{"query": query, "args": values})
	return e
}

func Preload(e *engine.Engine, schema string, values ...interface{}) *engine.Engine {
	var preloads []model.SearchPreload
	for _, preload := range e.Search.Preload {
		if preload.Schema != schema {
			preloads = append(preloads, preload)
		}
	}
	preloads = append(preloads, model.SearchPreload{schema, values})
	e.Search.Preload = preloads
	return e
}

func Raw(e *engine.Engine, b bool) *engine.Engine {
	e.Search.Raw = b
	return e
}

func Unscoped(e *engine.Engine, b bool) *engine.Engine {
	e.Search.Unscoped = b
	return e
}

func Table(e *engine.Engine, name string) *engine.Engine {
	e.Search.TableName = name
	return e
}
