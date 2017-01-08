package ql

import (
	"database/sql"
	"testing"

	_ "github.com/cznic/ql/driver"
)

type Department struct {
	ID   int
	Name string
}

const migration = `
BEGIN TRANSACTION;
	CREATE TABLE Orders (CustomerID int, Date time);
	CREATE INDEX OrdersID ON Orders (id());
	CREATE INDEX OrdersDate ON Orders (Date);
	CREATE TABLE Items (OrderID int, ProductID int, Qty int);
	CREATE INDEX ItemsOrderID ON Items (OrderID);
COMMIT;
`

func TestDialect(t *testing.T) {
	dialect := Memory()
	if dialect.GetName() != "ql-mem" {
		t.Errorf("expected ql-mem got %s", dialect.GetName())
	}
	dialect = File()
	if dialect.GetName() != "ql" {
		t.Errorf("expected ql got %s", dialect.GetName())
	}
	db, err := sql.Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tx.Exec(migration)
	_ = tx.Commit()
	dialect.SetDB(db)
	if dialect.db == nil {
		t.Fatal("expected the database to be set")
	}

	//
	// HasIndex
	//
	if !dialect.HasIndex("Orders", "OrdersID") {
		t.Error("expected to be true")
	}

	//RemoveIndex
	err = dialect.RemoveIndex("Orders", "OrdersID")
	if err != nil {
		t.Error(err)
	}

	if dialect.HasIndex("Orders", "OrdersID") {
		t.Error("expected to be fasle")
	}

	// Has Table
	if !dialect.HasTable("Orders") {
		t.Error("expected to be true")
	}
	if !dialect.HasColumn("Orders", "Date") {
		t.Error("expected to be true")
	}

}

func TestQL_Quote(t *testing.T) {
	q := &QL{}
	src := "quote"
	expect := `quote`
	v := q.Quote(src)
	if v != expect {
		t.Errorf("expected %s got %s", expect, v)
	}
}

func TestQL_BindVar(t *testing.T) {
	q := &QL{}
	src := 1
	expect := "$1"
	v := q.BindVar(src)
	if v != expect {
		t.Errorf("expected %s got %s", expect, v)
	}
}
