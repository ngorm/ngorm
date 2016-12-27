package scope

import (
	"testing"

	"github.com/gernest/gorm/engine"
	"github.com/gernest/gorm/model"
)

type CalculateField struct {
	model.Model
	Name     string
	Children []CalculateFieldChild
	Category CalculateFieldCategory
	EmbeddedField
}

type EmbeddedField struct {
	EmbeddedName string `sql:"NOT NULL;DEFAULT:'hello'"`
}

type CalculateFieldChild struct {
	model.Model
	CalculateFieldID uint
	Name             string
}

type CalculateFieldCategory struct {
	model.Model
	CalculateFieldID uint
	Name             string
}

func TestFieldByName(t *testing.T) {
	e := &engine.Engine{
		Search:    &engine.Search{},
		Scope:     &engine.Scope{},
		StructMap: model.NewModelStructsMap(),
	}
	e.Parent = e
	var field CalculateField
	if f, ok := FieldByName(e, &field, "Children"); !ok || f.Relationship == nil {
		t.Errorf("Should calculate fields correctly for the first time")
	}
}
