package search

import (
	"fmt"
	"regexp"

	"github.com/gernest/gorm/engine"
)

type Search struct {
	db               *engine.Engine
	whereConditions  []map[string]interface{}
	orConditions     []map[string]interface{}
	notConditions    []map[string]interface{}
	havingConditions []map[string]interface{}
	joinConditions   []map[string]interface{}
	initAttrs        []interface{}
	assignAttrs      []interface{}
	selects          map[string]interface{}
	omits            []string
	orders           []interface{}
	preload          []SearchPreload
	offset           interface{}
	limit            interface{}
	group            string
	tableName        string
	raw              bool
	Unscoped         bool
	ignoreOrderQuery bool
}

type SearchPreload struct {
	schema     string
	conditions []interface{}
}

func (s *Search) clone() *Search {
	clone := *s
	return &clone
}

func (s *Search) Where(query interface{}, values ...interface{}) *Search {
	s.whereConditions = append(s.whereConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *Search) Not(query interface{}, values ...interface{}) *Search {
	s.notConditions = append(s.notConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *Search) Or(query interface{}, values ...interface{}) *Search {
	s.orConditions = append(s.orConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *Search) Order(value interface{}, reorder ...bool) *Search {
	if len(reorder) > 0 && reorder[0] {
		s.orders = []interface{}{}
	}

	if value != nil {
		s.orders = append(s.orders, value)
	}
	return s
}

var distinctSQLRegexp = regexp.MustCompile(`(?i)distinct[^a-z]+[a-z]+`)

func (s *Search) Select(query interface{}, args ...interface{}) *Search {
	if distinctSQLRegexp.MatchString(fmt.Sprint(query)) {
		s.ignoreOrderQuery = true
	}

	s.selects = map[string]interface{}{"query": query, "args": args}
	return s
}

func (s *Search) Omit(columns ...string) *Search {
	s.omits = columns
	return s
}

func (s *Search) Limit(limit interface{}) *Search {
	s.limit = limit
	return s
}

func (s *Search) Offset(offset interface{}) *Search {
	s.offset = offset
	return s
}

func (s *Search) Having(query string, values ...interface{}) *Search {
	s.havingConditions = append(s.havingConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *Search) Joins(query string, values ...interface{}) *Search {
	s.joinConditions = append(s.joinConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *Search) Preload(schema string, values ...interface{}) *Search {
	var preloads []SearchPreload
	for _, preload := range s.preload {
		if preload.schema != schema {
			preloads = append(preloads, preload)
		}
	}
	preloads = append(preloads, SearchPreload{schema, values})
	s.preload = preloads
	return s
}

func (s *Search) Raw(b bool) *Search {
	s.raw = b
	return s
}

func (s *Search) unscoped() *Search {
	s.Unscoped = true
	return s
}

func (s *Search) Table(name string) *Search {
	s.tableName = name
	return s
}
