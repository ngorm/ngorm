package ngorm

import (
	"strings"
	"testing"

	_ "github.com/cznic/ql/driver"
	"github.com/gernest/ngorm/fixture"
)

type Foo struct {
	ID    int
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
	_, err = db.CreateTable(&Foo{})
	if err != nil {
		t.Error(err)
	}

	// multiple tables
	sql, err = db.CreateTableSQL(
		&fixture.User{},
	)
	if err != nil {
		t.Fatal(err)
	}
	expect = `
BEGIN TRANSACTION; 
	CREATE TABLE users (id int64,age int64,user_num int64,name string,email string,birthday time,created_at time,updated_at time,billing_address_id int64,shipping_address_id int64,latitude float64,company_id int,role string,password_hash blob,sequence uint ) ;
	CREATE TABLE user_languages (user_id uint,language_id uint ) ;
COMMIT;`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
	_, err = db.CreateTable(&fixture.User{})
	if err != nil {
		t.Error(err)
	}
}
