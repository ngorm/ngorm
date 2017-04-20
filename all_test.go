package ngorm

import (
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	_ "github.com/ngorm/postgres"
	_ "github.com/ngorm/ql"
)

type testDB interface {
	Open() (*DB, error)
	Clear(dbs ...interface{}) error
	Close() error
}

type wrapQL struct {
	*DB
	isClosed bool
}

func (q *wrapQL) Clear(tables ...interface{}) error {
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

type pgWrap struct {
	*DB
	isClosed bool
	conn     string
}

func (q *pgWrap) Open() (*DB, error) {
	if q.isClosed {
		d, err := Open("postgres", q.conn)
		if err != nil {
			return nil, err
		}
		q.DB = d
		q.isClosed = false
		return d, nil
	}
	return q.DB, nil
}

func (q *pgWrap) Close() error {
	q.isClosed = true
	return q.db.Close()
}

func (q *pgWrap) Clear(tables ...interface{}) error {
	_, err := q.DB.DropTable(tables...)
	if err != nil {
		return err
	}
	return q.Close()
}

var tsdb []testDB

func initialize() error {
	tsdb = append(tsdb, &wrapQL{isClosed: true})
	if ps := os.Getenv("NGORM_PG_CONN"); ps != "" {
		tsdb = append(tsdb, &pgWrap{isClosed: true, conn: ps})
	}
	return nil
}

func allTestDB() []testDB {
	return tsdb
}

func runWrapDB(t *testing.T, d testDB, f func(*testing.T, *DB), tables ...interface{}) {
	db, err := d.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Run(db.Dialect().GetName(), func(ts *testing.T) {
		f(ts, db)
	})
	err = d.Clear(tables...)
	if err != nil {
		t.Fatal(err)
	}
}

func runWrapBenchDB(t *testing.B, d testDB, f func(*testing.B, *DB), tables ...interface{}) {
	db, err := d.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Automigrate(tables...)
	if err != nil {
		t.Fatal(err)
	}
	t.Run(db.Dialect().GetName(), func(ts *testing.B) {
		f(ts, db)
	})
	err = d.Clear(tables...)
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
