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
	CREATE TABLE user_languages (user_id int64,language_id int64 ) ;
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
	CREATE TABLE user_languages (user_id int64,language_id int64 ) ;
	CREATE TABLE emails (id int16,user_id int,email string,created_at time,updated_at time ) ;
	CREATE TABLE languages (id int64,created_at time,updated_at time,deleted_at time,name string ) ;
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
	UPDATE foos SET stuff = $1  WHERE id = $2;
COMMIT;`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_UpdateSQL(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	foo := Foo{ID: 10, Stuff: "twenty"}
	sql, err := db.Model(&foo).UpdateSQL("stuff", "hello")
	if err != nil {
		t.Fatal(err)
	}
	expect := `
BEGIN TRANSACTION;
	UPDATE foos SET stuff = $1  WHERE id = $2;
COMMIT;`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_SingularTable(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	db.SingularTable(true)
	sql, err := db.CreateSQL(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	expect := `
BEGIN TRANSACTION;
	INSERT INTO foo (stuff) VALUES ($1);
COMMIT;`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_HasTable(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if db.HasTable("foos") {
		t.Error("expected false")
	}
	if db.HasTable(&Foo{}) {
		t.Error("expected false")
	}
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	if !db.HasTable("foos") {
		t.Error("expected true")
	}
	if !db.HasTable(&Foo{}) {
		t.Error("expected true")
	}

}

func TestDB_FirstSQL(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// First record order by primary key
	sql, err := db.FirstSQL(&fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	expect := `SELECT * FROM users   ORDER BY id ASC`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}

	// First record with primary key
	sql, err = db.clone().FirstSQL(&fixture.User{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	expect = `SELECT * FROM users  WHERE (id = $1) ORDER BY id ASC`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_First(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.First(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if fu.Stuff != "a" {
		t.Errorf("expected a got  %s", fu.Stuff)
	}
}

func TestDB_LastSQL(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// First record order by primary key
	sql, err := db.LastSQL(&fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	expect := `SELECT * FROM users   ORDER BY id DESC`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}

	// First record with primary key
	sql, err = db.clone().LastSQL(&fixture.User{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	expect = `SELECT * FROM users  WHERE (id = $1) ORDER BY id DESC`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}
func TestDB_Last(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.Last(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if fu.Stuff != "d" {
		t.Errorf("expected d got  %s", fu.Stuff)
	}
}
