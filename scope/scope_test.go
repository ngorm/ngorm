package scope

import (
	"testing"

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
