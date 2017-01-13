package ngorm

import (
	"strings"
	"testing"
	"time"

	_ "github.com/cznic/ql/driver"
	"github.com/gernest/ngorm/fixture"
)

type Foo struct {
	ID    int
	Stuff string
}

func TestDB_CreateTable(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

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

func TestDB_DropTable(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	_, err = db.DropTable(&Foo{})
	if err == nil {
		t.Error("expected error")
	}

	_, err = db.CreateTable(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.DropTable(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	if db.dialect.HasTable("foos") {
		t.Error("expected the table to disappear")
	}

	sql, err := db.DropTableSQL(&Foo{}, &fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	expect := `
BEGIN TRANSACTION; 
	DROP TABLE foos;
	DROP TABLE users;
COMMIT;`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_Automirate(t *testing.T) {
	db, err := Open("ql-mem", "est.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	sql, err := db.AutomigrateSQL(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}
	expect := `
BEGIN TRANSACTION;
	CREATE TABLE users (id int64,age int64,user_num int64,name string,email string,birthday time,created_at time,updated_at time,billing_address_id int64,shipping_address_id int64,latitude float64,company_id int,role string,password_hash blob,sequence uint ) ;
	CREATE TABLE user_languages (user_id uint,language_id uint ) ;
	CREATE TABLE emails (id int16,user_id int,email string,created_at time,updated_at time ) ;
	CREATE TABLE languages (id uint,created_at time,updated_at time,deleted_at time,name string ) ;
	CREATE INDEX idx_languages_deleted_at ON languages(deleted_at);
	CREATE TABLE companies (id int64,name string ) ;
	CREATE TABLE credit_cards (id int8,number string,user_id int64,created_at time NOT NULL,updated_at time,deleted_at time ) ;
	CREATE TABLE addresses (id int,address1 string,address2 string,post string,created_at time,updated_at time,deleted_at time ) ;
COMMIT;`

	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
	_, err = db.Automigrate(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDb_Create(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}
	n := time.Now()
	user := fixture.User{Name: "Jinzhu", Age: 18, Birthday: &n}
	_, err = db.CreateSQL(&user)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&user)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDB_SaveSQL(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	sql, err := db.SaveSQL(&Foo{ID: 10, Stuff: "twenty"})
	if err != nil {
		t.Fatal(err)
	}
	expect := `
BEGIN TRANSACTION;
	UPDATE foos SET stuff = $1  WHERE foos.id = $2;
COMMIT;`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}
