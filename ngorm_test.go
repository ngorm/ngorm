package ngorm

import (
	"fmt"
	"strings"
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
	expect := `
BEGIN TRANSACTION; 
	CREATE TABLE foos (id int,stuff string ) ;
COMMIT;`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
	fmt.Println(sql.Q)
	_, err = db.CreateTable(&Foo{})
	if err != nil {
		t.Error(err)
	}

}
