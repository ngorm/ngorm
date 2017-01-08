package ngorm

import (
	"fmt"
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
	sql, err := db.CreateTableSQL(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(sql.Q)
	_, err = db.CreateTable(&Foo{})
	if err != nil {
		t.Error(err)
	}

}
