package ngorm

import (
	"log"
	"os"
	"testing"
)

type testDB interface {
	Open() (*DB, error)
	Clear(dbs ...string) error
	Close() error
}

type wrapQL struct {
	*DB
	isClosed bool
}

func (q *wrapQL) Clear(databases ...string) error {
	return q.Close()
}
func (q *wrapQL) Close() error {
	q.isClosed = true
	return q.db.Close()
}

func (q *wrapQL) Open() (*DB, error) {
	if q.isClosed {
		d, err := Open("ql-mem", "test.db")
		if err != nil {
			return nil, err
		}
		q.DB = d
		q.isClosed = false
		return d, nil
	}
	return q.DB, nil
}

var tsdb []testDB

func initialize() error {
	qldb, err := Open("ql-mem", "test")
	if err != nil {
		return err
	}
	tsdb = append(tsdb, &wrapQL{DB: qldb})
	return nil
}

func AllTestDB() []testDB {
	return tsdb
}

func runWrapDB(t *testing.T, d testDB, f func(*testing.T, *DB)) {
	db, err := d.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Run(db.Dialect().GetName(), func(ts *testing.T) {
		f(ts, db)
	})
	err = d.Clear()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	err := initialize()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}
