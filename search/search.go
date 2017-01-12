//Package search contains search functions for ngorm.
package search

import (
	"fmt"

	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/model"
	"github.com/gernest/ngorm/regexes"
	"github.com/gernest/ngorm/util"
)

//Where adds WHERE search condition.
func Where(e *engine.Engine, query interface{}, values ...interface{}) {
	e.Search.WhereConditions = append(e.Search.WhereConditions, map[string]interface{}{"query": query, "args": values})
}

//Not adds NOT search condition
func Not(e *engine.Engine, query interface{}, values ...interface{}) {
	e.Search.NotConditions = append(e.Search.NotConditions, map[string]interface{}{"query": query, "args": values})
}

//Or add OR search condition
func Or(e *engine.Engine, query interface{}, values ...interface{}) {
	e.Search.OrConditions = append(e.Search.OrConditions, map[string]interface{}{"query": query, "args": values})
}

//Attr add attributes
func Attr(e *engine.Engine, attrs ...interface{}) {
	e.Search.InitAttrs = append(e.Search.InitAttrs, toSearchableMap(attrs...))
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

//Assign assigns attrs
func Assign(e *engine.Engine, attrs ...interface{}) {
	e.Search.AssignAttrs = append(e.Search.AssignAttrs, toSearchableMap(attrs...))
}

//Order add ORDER BY search condition
func Order(e *engine.Engine, value interface{}, reorder ...bool) {
	if len(reorder) > 0 && reorder[0] {
		e.Search.Orders = []interface{}{}
	}

	if value != nil {
		e.Search.Orders = append(e.Search.Orders, value)
	}
}

//Select add SELECT querry
func Select(e *engine.Engine, query interface{}, values ...interface{}) {
	if regexes.DistinctSQL.MatchString(fmt.Sprint(query)) {
		e.Search.IgnoreOrderQuery = true
	}
	e.Search.Selects = map[string]interface{}{"query": query, "args": values}
}

//Omit ommits seacrh condition
func Omit(e *engine.Engine, columns ...string) {
	e.Search.Omits = columns
}

//Limit add search LIMIT
func Limit(e *engine.Engine, limit interface{}) {
	e.Search.Limit = limit
}

//Offset add search OFFSET
func Offset(e *engine.Engine, offset interface{}) {
	e.Search.Offset = offset
}

//Group  add GROUP BY search condition.
func Group(e *engine.Engine, query interface{}) {
	s, err := util.GetInterfaceAsSQL(query)
	if err != nil {
		e.Error = err
	} else {
		e.Search.Group = s
	}
}

//Having add HAVING condition
func Having(e *engine.Engine, query interface{}, values ...interface{}) {
	e.Search.HavingConditions = append(e.Search.HavingConditions, map[string]interface{}{"query": query, "args": values})
}

//Join add JOIN condition
func Join(e *engine.Engine, query interface{}, values ...interface{}) {
	e.Search.JoinConditions = append(e.Search.JoinConditions, map[string]interface{}{"query": query, "args": values})
}

//Preload add preloading condition
func Preload(e *engine.Engine, schema string, values ...interface{}) {
	var preloads []model.SearchPreload
	for _, preload := range e.Search.Preload {
		if preload.Schema != schema {
			preloads = append(preloads, preload)
		}
	}
	preloads = append(preloads, model.SearchPreload{Schema: schema, Conditions: values})
	e.Search.Preload = preloads
}

//Raw set the seacrh querry to RAw
func Raw(e *engine.Engine, b bool) {
	e.Search.Raw = b
}

//Unscoped set the search scope status
func Unscoped(e *engine.Engine, b bool) {
	e.Search.Unscoped = b
}

//Table set the search table name.
func Table(e *engine.Engine, name string) {
	e.Search.TableName = name
}
