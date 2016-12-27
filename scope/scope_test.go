package scope

import (
	"testing"

	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/fixture"
	"github.com/gernest/ngorm/model"
)

func TestFieldByName(t *testing.T) {
	e := &engine.Engine{
		Search:    &engine.Search{},
		Scope:     &engine.Scope{},
		StructMap: model.NewModelStructsMap(),
	}
	e.Parent = e
	var field fixture.CalculateField
	if f, ok := FieldByName(e, &field, "Children"); !ok || f.Relationship == nil {
		t.Errorf("Should calculate fields correctly for the first time")
	}
}
