package search

import (
	"testing"

	"github.com/ngorm/ngorm/fixture"
)

func TestSearch(t *testing.T) {
	e := fixture.TestEngine()
	Where(e, "name = ?", "gernest")
	Order(e, "name")
	Attr(e, "name", "gernest")
	Select(e, "name, age")
}
