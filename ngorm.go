//Package ngorm i a Go Object relation mapper that focus on performance,
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
	"errors"
	"fmt"
	"strings"

	"github.com/gernest/ngorm/dialects"
	"github.com/gernest/ngorm/dialects/ql"
	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/hooks"
	"github.com/gernest/ngorm/logger"
	"github.com/gernest/ngorm/model"
	"github.com/gernest/ngorm/scope"
	"github.com/gernest/ngorm/search"
	"github.com/gernest/ngorm/util"
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
	e             *engine.Engine
	err           error
}

func (db *DB) clone() *DB {
	n := &DB{
		db:            db.db,
		dialect:       db.dialect,
		ctx:           db.ctx,
		cancel:        db.cancel,
		singularTable: db.singularTable,
		structMap:     db.structMap,
		hooks:         db.hooks,
		log:           db.log,
	}
	ne := n.NewEngine()
	n.e = ne
	return n
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
	h := hooks.DefaultBook()
	switch dia.GetName() {
	case "ql", "ql-mem":
		h.Create.Set(
			hooks.HookFunc(model.AfterCreate, hooks.QLAfterCreate),
		)
	}
	return &DB{
		db:        db,
		dialect:   dia,
		structMap: model.NewStructsMap(),
		ctx:       ctx,
		hooks:     h,
		cancel:    cancel,
		log:       logger.New(o),
	}, nil
}

// NewEngine returns an initialized engine ready to kick some ass.
func (db *DB) NewEngine() *engine.Engine {
	return &engine.Engine{
		Search:        &model.Search{},
		Scope:         model.NewScope(),
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
	var scopeVars map[string]interface{}
	if db.e != nil {
		scopeVars = db.e.Scope.GetAll()
	}
	var buf bytes.Buffer
	_, _ = buf.WriteString("BEGIN TRANSACTION; \n")
	for _, m := range models {
		e := db.NewEngine()
		for k, v := range scopeVars {
			e.Scope.Set(k, v)
		}
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

//Open opens up database connection using the database/sql package.
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

//DropTableSQL generates sql query for DROP TABLE. The generated query is
//wrapped under TRANSACTION block.
func (db *DB) DropTableSQL(models ...interface{}) (*model.Expr, error) {
	var buf bytes.Buffer
	_, _ = buf.WriteString("BEGIN TRANSACTION; \n")
	for _, m := range models {
		e := db.NewEngine()
		if n, ok := m.(string); ok {
			e.Search.TableName = n
		}
		// Firste we generate the SQL
		err := scope.DropTable(e, m)
		if err != nil {
			return nil, err
		}
		_, _ = buf.WriteString("\t" + e.Scope.SQL + ";\n")
	}
	_, _ = buf.WriteString("COMMIT;")
	return &model.Expr{Q: buf.String()}, nil
}

//DropTable drops tables that are mapped to models. You can also pass the name
//of the table as astring and it will be handled.
func (db *DB) DropTable(models ...interface{}) (sql.Result, error) {
	query, err := db.DropTableSQL(models...)
	if err != nil {
		return nil, err
	}
	return db.ExecTx(query.Q, query.Args...)
}

//Automigrate creates tables that map to models if the tables don't exist yet in
//the database. This also takes care of situation where the models's fields have
//been updated(changed)
func (db *DB) Automigrate(models ...interface{}) (sql.Result, error) {
	query, err := db.AutomigrateSQL(models...)
	if err != nil {
		return nil, err
	}
	return db.ExecTx(query.Q, query.Args...)
}

//AutomigrateSQL generates sql query for running migrations on models.
func (db *DB) AutomigrateSQL(models ...interface{}) (*model.Expr, error) {
	var buf bytes.Buffer
	_, _ = buf.WriteString("BEGIN TRANSACTION;\n")
	keys := make(map[string]bool)
	for _, m := range models {
		e := db.NewEngine()

		// Firste we generate the SQL
		err := scope.Automigrate(e, m)
		if err != nil {
			return nil, err
		}
		if e.Scope.SQL != "" {
			i := strings.Index(e.Scope.SQL, "(")
			k := e.Scope.SQL[:i]
			if _, ok := keys[k]; !ok {
				_, _ = buf.WriteString("\t" + e.Scope.SQL + ";\n")
				keys[k] = true
			}
		}
		if e.Scope.MultiExpr {
			for _, expr := range e.Scope.Exprs {
				i := strings.Index(expr.Q, "(")
				k := expr.Q[:i]
				if _, ok := keys[k]; !ok {
					_, _ = buf.WriteString("\t" + expr.Q + ";\n")
					keys[k] = true
				}
			}
		}
	}
	_, _ = buf.WriteString("COMMIT;")
	return &model.Expr{Q: buf.String()}, nil
}

//Close closes the database connection and sends Done signal across all
//goroutines that subscribed to this instance context.
func (db *DB) Close() error {
	db.cancel()
	return db.db.Close()
}

//Create creates a new record.
//
// You can hijack the execution of the generated SQL by overiding
// model.HookCreateExec hook.
func (db *DB) Create(value interface{}) error {
	sql, err := db.CreateSQL(value)
	if err != nil {
		return err
	}
	c, ok := db.hooks.Create.Get(model.HookCreateExec)
	if !ok {
		return errors.New("missing execution hook")
	}
	e := db.NewEngine()
	e.Scope.Value = value
	e.Scope.SQL = sql.Q
	e.Scope.SQLVars = sql.Args
	err = c.Exec(db.hooks, e)
	if err != nil {
		return err
	}
	if ac, ok := db.hooks.Create.Get(model.AfterCreate); ok {
		return ac.Exec(db.hooks, e)
	}
	return nil
}

//CreateSQL generates SQl query for creating a new record/records for value. This
//uses Hooks to allow more flexibility.
//
// There is no error propagation. Each step/hook execution must pass. Any error
// indicate the end of the execution.
//
// The hooks that are used here are all defined in model package  as constants. You can
// easily overide them by using DB.SetCreateHook method.
//
//	model.BeforeCreate
//If set, this is the first hook to be executed. The default hook that is used
//is defined in hooks.BeforeCreate. If by any chance the hook returns an error
//then execution is halted and the error is returned.
//
//	model.HookSaveBeforeAss
// If the model value has association and  this is set then it will be executed.
// This is useful if you also want to save associations.
//
//	model.HookUpdateTimestamp
// New record needs to have CreatedAt and UpdatedAt properly set. This is
// excuted to update the record timestamps( The default hook for this assumes
// you used model.Model convention for naming the timestamp fields).
//
//	model.Create
// The last hook to be executed.
//
// NOTE: All the hooks must be tailored towards generating SQL not executing
// anything that might change the state of the table.
//
// All the other hooks apart from model.Create should write SQQL gerries in
// e.Scope.Epxrs only model.Create hook should write to e.Scope.SQL.
//
// The end query is wrapped under TRANSACTION block.
func (db *DB) CreateSQL(value interface{}) (*model.Expr, error) {
	var e *engine.Engine
	if db.e != nil {
		e = db.e
	} else {
		e = db.NewEngine()
	}
	e.Scope.Value = value
	if c, ok := db.hooks.Create.Get(model.HookCreateSQL); ok {
		err := c.Exec(db.hooks, e)
		if err != nil {
			return nil, err
		}
		return &model.Expr{Q: e.Scope.SQL, Args: e.Scope.SQLVars}, nil
	}
	return nil, errors.New("missing create sql hook")

}

//Dialect return the dialect that is used by DB
func (db *DB) Dialect() dialects.Dialect {
	return db.dialect
}

//SQLCommon return SQLCommon used by the DB
func (db *DB) SQLCommon() model.SQLCommon {
	return db.db
}

//SaveSQL generates SQL query for saving/updating database record for value.
func (db *DB) SaveSQL(value interface{}) (*model.Expr, error) {
	e := db.NewEngine()
	e.Scope.Value = value
	if u, ok := db.hooks.Update.Get(model.HookUpdateSQL); ok {
		err := u.Exec(db.hooks, e)
		if err != nil {
			return nil, err
		}
		return &model.Expr{Q: e.Scope.SQL, Args: e.Scope.SQLVars}, nil
	}
	return nil, errors.New("missing update sql hook")
}

//Model sets value as the database model. This model will be used for future
//calls on the erturned DB e.g
//
//	db.Model(&user).Update("name","hero")
func (db *DB) Model(value interface{}) *DB {
	c := db.clone()
	c.e.Scope.Value = value
	return c
}

//Update runs UPDATE queries.
func (db *DB) Update(attrs ...interface{}) error {
	return db.Updates(util.ToSearchableMap(attrs), true)
}

//Updates runs UPDATE query
func (db *DB) Updates(values interface{}, ignoreProtectedAttrs ...bool) error {
	if db.e == nil || db.e.Scope.Value == nil {
		return errors.New("missing model, before calling this startwith db.Model")
	}
	var ignore bool
	if len(ignoreProtectedAttrs) > 0 {
		ignore = ignoreProtectedAttrs[0]
	}
	db.e.Scope.Set(model.IgnoreProtectedAttrs, ignore)
	db.e.Scope.Set(model.UpdateInterface, values)
	u, ok := db.hooks.Update.Get(model.Update)
	if !ok {
		return errors.New("missing update hook")
	}
	return u.Exec(db.hooks, db.e)
}

//UpdateSQL generates SQL that will be executed when you use db.Update
func (db *DB) UpdateSQL(attrs ...interface{}) (*model.Expr, error) {
	return db.UpdatesSQL(util.ToSearchableMap(attrs), true)
}

//UpdatesSQL generates sql that will be used when you run db.UpdatesSQL
func (db *DB) UpdatesSQL(values interface{}, ignoreProtectedAttrs ...bool) (*model.Expr, error) {
	if db.e == nil || db.e.Scope.Value == nil {
		return nil, errors.New("missing model, before calling this startwith db.Model")
	}
	var ignore bool
	if len(ignoreProtectedAttrs) > 0 {
		ignore = ignoreProtectedAttrs[0]
	}
	db.e.Scope.Set(model.IgnoreProtectedAttrs, ignore)
	db.e.Scope.Set(model.UpdateInterface, values)
	u, ok := db.hooks.Update.Get(model.HookUpdateSQL)
	if !ok {
		return nil, errors.New("missing update sql  hook")
	}
	err := u.Exec(db.hooks, db.e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: db.e.Scope.SQL, Args: db.e.Scope.SQLVars}, nil
}

//Set sets scope key to value.
func (db *DB) Set(key string, value interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	db.e.Scope.Set(key, value)
	return db
}

//SingularTable enables or diables singular tables name. By default this is
//diables, meaning table names are in plurar.
//
//	Model	| Plural table name
//	----------------------------
//	Session	| sessions
// 	User	| users
//
//	Model	| Singular table name
//	----------------------------
//	Session	| session
// 	User	| user
func (db *DB) SingularTable(enable bool) {
	db.singularTable = enable
	if db.e != nil {
		db.e.SingularTable = enable
	}
}

//HasTable returns true if there is a table for the given value, the value can
//either be a string representing a table name or a ngorm model.
func (db *DB) HasTable(value interface{}) bool {
	var name string
	if n, ok := value.(string); ok {
		name = n
	} else {
		e := db.NewEngine()
		name = scope.TableName(e, value)
	}
	return db.Dialect().HasTable(name)
}

//First  fets the first record and order by primary key.
//
// BUG: For some reason the ql database doesnt order the keys in ascending
// order. So this uses DESC to get the real record instead of ASC , I will need
// to dig more and see.
func (db *DB) First(out interface{}, where ...interface{}) error {
	db.Set(model.OrderByPK, "DESC")
	if len(where) > 0 {
		if len(where) == 1 {
			search.Where(db.e, where[0])
		} else {
			search.Where(db.e, where[0], where[1:]...)
		}
	}
	db.e.Scope.Value = out
	q, ok := db.hooks.Query.Get(model.Query)
	if !ok {
		return errors.New("missing query hook")
	}
	return q.Exec(db.hooks, db.e)
}

//FirstSQL returns SQL query for retrieving the first record ordering by primary
//key.
func (db *DB) FirstSQL(out interface{}, where ...interface{}) (*model.Expr, error) {
	db.Set(model.OrderByPK, "ASC")
	if len(where) > 0 {
		if len(where) == 1 {
			search.Where(db.e, where[0])
		} else {
			search.Where(db.e, where[0], where[1:]...)
		}
	}
	db.e.Scope.Value = out
	sql, ok := db.hooks.Query.Get(model.HookQuerySQL)
	if !ok {
		return nil, errors.New("missing  query sql hook")
	}
	err := sql.Exec(db.hooks, db.e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: db.e.Scope.SQL, Args: db.e.Scope.SQLVars}, nil
}

//Last finds the last record and order by primary key.
//
// BUG: For some reason the ql database doesnt order the keys in descending
// order despite the use of DESC, so this uses ASC but the keys are in
// descending order.
// order. So this uses DESC to get the real record instead of ASC , I will need
func (db *DB) Last(out interface{}, where ...interface{}) error {
	db.Set(model.OrderByPK, "ASC")
	if len(where) > 0 {
		if len(where) == 1 {
			search.Where(db.e, where[0])
		} else {
			search.Where(db.e, where[0], where[1:]...)
		}
	}
	db.e.Scope.Value = out
	q, ok := db.hooks.Query.Get(model.Query)
	if !ok {
		return errors.New("missing query hook")
	}
	return q.Exec(db.hooks, db.e)
}

//LastSQL returns SQL query for retrieving the last record ordering by primary
//key.
func (db *DB) LastSQL(out interface{}, where ...interface{}) (*model.Expr, error) {
	db.Set(model.OrderByPK, "DESC")
	if len(where) > 0 {
		if len(where) == 1 {
			search.Where(db.e, where[0])
		} else {
			search.Where(db.e, where[0], where[1:]...)
		}
	}
	db.e.Scope.Value = out
	sql, ok := db.hooks.Query.Get(model.HookQuerySQL)
	if !ok {
		return nil, errors.New("missing  query sql hook")
	}
	err := sql.Exec(db.hooks, db.e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: db.e.Scope.SQL, Args: db.e.Scope.SQLVars}, nil
}

//Hooks returns the hook book for this db instance.
func (db *DB) Hooks() *hooks.Book {
	return db.hooks
}

// Limit specify the number of records to be retrieved
func (db *DB) Limit(limit interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Limit(db.e, limit)
	return db
}
