//Package ngorm is a Go Object relation mapper that focus on performance,
//maintainability, modularity,	battle testing, extensibility , safety and
//developer frindliness.
//
// To achieve all of the goals, the project is divided into many components. The
// components are desined in a functional style API, whereby objects are
// explicitly passed around as arguments to functions that operate on them.
//
// This tries to avoid defining methods on structs. This comes at a cost of
// limiting chaining, this cost is intentional. I intend to work really hard on
// improving performance and thus avoiding spaghetti is not an option.
//
// Installation
//
// You can install  with go get
//   go get -u github.com/gernest/ngorm
//
//
// The package is divided into two phases, Query building and Query execution
// phase.
//
// Query Building
//
// The subpackage engine exposes a structure named Engine. This structure has
// everything necessary to build a query. Most of the functions defined in this
// package subpackages operate on this struct by accepting it as the first
// argument.
//
// Having this as a separate layer helps fine tuning the generated querries and
// also it make easy to test and very that the ORM is doing the right thing. So,
// the generated query can be easily optimised without adding a lot of overhead.
//
// Query execution
//
// This is s the phase where the generated sql query is executed. This phase is as generic as
// possible in a way that you can easily implement adoptes for non SQL database
// and still reap all the benefits of this package.
//
// Table of Ccntents
//
// The following are links to packages under this project.
//
// WARNING: You will never be touching most of these  packages. They are the
// building block of the high level API.
//   [engine] https://godoc.org/github.com/gernest/ngorm/engine
// This is what drives the whole project, helps with query building and provides
// conveinet structure to help with query execution.
//
//   [scope] https://godoc.org/github.com/gernest/ngorm/scope
// Functions to help with model manipulations.
//
//   [search] https://godoc.org/github.com/gernest/ngorm/search
// Functions to help with search  querries building.
//
//   [hooks] https://godoc.org/github.com/gernest/ngorm/hooks
// Callbacks executed by ngorm. You can easily overide and provide custom ones
// to suit your needs.
//
//   [logger] https://godoc.org/github.com/gernest/ngorm/logger
// The logger used by ngorm for logging. It is an interface, and a reference
// implementation is provided.
//
//   [dialects] https://godoc.org/github.com/gernest/ngorm/dialects
// Adopts to different SQL databases supported by ngorm. For now ngorm support
// ql .
package ngorm

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"

	"github.com/gernest/ngorm/dialects"
	"github.com/gernest/ngorm/dialects/ql"
	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/hooks"
	"github.com/gernest/ngorm/logger"
	"github.com/gernest/ngorm/model"
	"github.com/gernest/ngorm/scope"
	"github.com/uber-go/zap"
)

//Opener is an interface that is used to open up connection to SQL databases.
type Opener interface {
	Open(dialect string, args ...interface{}) (model.SQLCommon, dialects.Dialect, error)
}

// DB provide an API for interacting with SQL databases using Go data structures.
type DB struct {
	db            model.SQLCommon
	dialect       dialects.Dialect
	connStr       string
	ctx           context.Context
	cancel        func()
	singularTable bool
	structMap     *model.SafeStructsMap
	hooks         *hooks.Book
	log           *logger.Zapper
}

//Open opens a database connection and returns *DB instance., dialect is the
//name of the driver that you want to use. The underlying connections are
//handled by database/sql package. Arguments that are accepted by database/sql
//Open function are valid here.
//
// Not all databases are supported. There is still an ongoing efforts to add
// more databases but for now the following are the databases  supported by this
// library,
//
//   * ql https://github.com/cznic/ql
//
// The drivers for the libraries must be imported inside your application in the
// same package as you invoke this function.
//
// Example
//
//   import _ "github.com/cznic/ql/driver"  // imports ql driver
func Open(dialect string, args ...interface{}) (*DB, error) {
	return OpenWithOpener(&DefaultOpener{}, dialect, args...)
}

// OpenWithOpener uses the opener to initialize the dialects and establish
// database connection. In fact Open does not do anything by itself, it just
// calls this function with the default Opener.
//
// Please see Open function for details. The only difference is here you need to
// pass an Opener. See the Opener interface for details about what the Opener is
// and what it is used for.
func OpenWithOpener(opener Opener, dialect string, args ...interface{}) (*DB, error) {
	db, dia, err := opener.Open(dialect, args...)
	if err != nil {
		return nil, err
	}
	dia.SetDB(db)
	o := zap.New(
		zap.NewTextEncoder(zap.TextNoTime()), // drop timestamps in tests
	)
	ctx, cancel := context.WithCancel(context.Background())
	return &DB{
		db:        db,
		dialect:   dia,
		structMap: model.NewStructsMap(),
		ctx:       ctx,
		hooks:     hooks.DefaultBook(),
		cancel:    cancel,
		log:       logger.New(o),
	}, nil
}

// NewEngine returns an initialized engine ready to kick some ass.
func (db *DB) NewEngine() *engine.Engine {
	return &engine.Engine{
		Search:        &model.Search{},
		Scope:         &model.Scope{},
		StructMap:     db.structMap,
		SingularTable: db.singularTable,
		Ctx:           db.ctx,
		Dialect:       db.dialect,
		SQLDB:         db.db,
		Log:           db.log,
	}
}

//CreateTable creates new database tables that maps to the models.
func (db *DB) CreateTable(models ...interface{}) (sql.Result, error) {
	query, err := db.CreateTableSQL(models...)
	if err != nil {
		return nil, err
	}
	return db.ExecTx(query.Q, query.Args...)
}

//ExecTx wraps the query execution in a Transaction. This ensure all operations
//are Rolled back in case the execution fials.
func (db *DB) ExecTx(query string, args ...interface{}) (sql.Result, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return nil, err
	}
	r, err := tx.Exec(query, args...)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return r, nil
}

//CreateTableSQL return the sql query for creating tables for all the given
//models. The queries are wrapped in a TRANSACTION block.
func (db *DB) CreateTableSQL(models ...interface{}) (*model.Expr, error) {
	var buf bytes.Buffer
	_, _ = buf.WriteString("BEGIN TRANSACTION; \n")
	for _, m := range models {
		e := db.NewEngine()

		// Firste we generate the SQL
		err := scope.CreateTable(e, m)
		if err != nil {
			return nil, err
		}
		_, _ = buf.WriteString("\t" + e.Scope.SQL + ";\n")
		if e.Scope.MultiExpr {
			for _, expr := range e.Scope.Exprs {
				_, _ = buf.WriteString("\t" + expr.Q + ";\n")
			}
		}
	}
	_, _ = buf.WriteString("COMMIT;")
	return &model.Expr{Q: buf.String()}, nil
}

//DefaultOpener implements Opener interface.
type DefaultOpener struct {
}

//Open opens up database connection using the database/sql packge.
func (d *DefaultOpener) Open(dialect string, args ...interface{}) (model.SQLCommon, dialects.Dialect, error) {
	var source string
	var dia dialects.Dialect
	var common model.SQLCommon
	var err error

	switch value := args[0].(type) {
	case string:
		var driver = dialect
		if len(args) == 1 {
			source = value
		} else if len(args) >= 2 {
			driver = value
			source = args[1].(string)
		}
		common, err = sql.Open(driver, source)
		if err != nil {
			return nil, nil, err
		}
	case model.SQLCommon:
		common = value
	default:
		return nil, nil, fmt.Errorf("unknown argument %v", value)
	}
	switch dialect {
	case "ql":
		dia = ql.File()
	case "ql-mem":
		dia = ql.Memory()
	default:
		return nil, nil, fmt.Errorf("unsupported dialect %s", dialect)
	}
	return common, dia, nil
}
