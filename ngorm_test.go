package ngorm

import (
	"testing"

	_ "github.com/cznic/ql/driver"
)

type Foo struct {
	Id    int
	Stuff string
}

func TestDB(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}

	// create table tests
	err = db.CreateTable(&Foo{})
	if err != nil {
		t.Error(err)
	}

}
