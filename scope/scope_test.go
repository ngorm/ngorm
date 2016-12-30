package scope

import (
	"testing"

	"github.com/gernest/ngorm/dialects/ql"
	"github.com/gernest/ngorm/fixture"
)

func TestFieldByName(t *testing.T) {
	e := fixture.TestEngine()
	e.Parent = e
	var field fixture.CalculateField
	if f, ok := FieldByName(e, &field, "Children"); !ok || f.Relationship == nil {
		t.Errorf("Should calculate fields correctly for the first time")
	}
}

func TestQuote(t *testing.T) {
	e := fixture.TestEngine()
	e.Dialect = &ql.QL{}
	e.Parent = e
	sample := []struct {
		src, expetc string
	}{
		{"quote", `"quote"`},
		{"quote.quote.quote", `"quote"."quote"."quote"`},
	}

	for _, v := range sample {
		q := Quote(e, v.src)
		if q != v.expetc {
			t.Errorf("expected %s got %s", v.expetc, q)
		}
	}
}
