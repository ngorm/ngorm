package builder

import (
	"fmt"
	"testing"

	"github.com/gernest/ngorm/dialects/ql"
	"github.com/gernest/ngorm/fixture"
	"github.com/gernest/ngorm/search"
)

func TestGroup(t *testing.T) {
	e := fixture.TestEngine()
	s := GroupSQL(e)
	if s != "" {
		t.Errorf("expected an empty string got %s", s)
	}
	by := "location"
	search.Group(e, by)
	s = GroupSQL(e)
	expect := " GROUP BY " + by
	if s != expect {
		t.Errorf("expected %s got %s", expect, s)
	}

}

func TestLimitAndOffsetSQL(t *testing.T) {
	e := fixture.TestEngine()
	e.Dialect = ql.Memory()
	limit := 2
	offset := 4
	search.Limit(e, limit)
	search.Offset(e, offset)
	expect := fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	s := LimitAndOffsetSQL(e)
	if s != expect {
		t.Errorf("expected %s got %s", expect, s)
	}

}
