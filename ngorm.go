//Package ngorm is a Go Object relation mapper that focus on performance,
//maintainability, modularity,	battle testing, extensibility , safety and
//developer friendliness.
//
//
// Installation
//
// You can install  with go get
//   go get -u github.com/ngorm/ngorm
//
// Supported databases
//
//At the moment the following databases are supported
//   - ql
//   - postgresql
//
//
package ngorm

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ngorm/ngorm/builder"
	"github.com/ngorm/ngorm/dialects"
	"github.com/ngorm/ngorm/engine"
	"github.com/ngorm/ngorm/errmsg"
	"github.com/ngorm/ngorm/hooks"
	"github.com/ngorm/ngorm/model"
	"github.com/ngorm/ngorm/regexes"
	"github.com/ngorm/ngorm/scope"
	"github.com/ngorm/ngorm/search"
	"github.com/ngorm/ngorm/util"
)

//Opener is an interface that is used to open up connection to SQL databases.
type Opener interface {
	Open(dialect string, args ...interface{}) (model.SQLCommon, dialects.Dialect, error)
}

// DB provide an API for interacting with SQL databases using Go data structures.
type DB struct {
	db            *model.SQLCommonWrapper
	dialect       dialects.Dialect
	connStr       string
	ctx           context.Context
	cancel        func()
	singularTable bool
	structMap     *model.SafeStructsMap
	e             *engine.Engine
	err           error
	now           func() time.Time
}

func (db *DB) clone() *DB {
	return &DB{
		db:            db.db,
		dialect:       db.dialect,
		ctx:           db.ctx,
		cancel:        db.cancel,
		singularTable: db.singularTable,
		structMap:     db.structMap,
		now:           time.Now,
		e:             db.NewEngine(),
	}
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
	return OpenWithOpener(dialects.Opener(), dialect, args...)
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
	ctx, cancel := context.WithCancel(context.Background())
	return &DB{
		db:        &model.SQLCommonWrapper{SQLCommon: db},
		dialect:   dia,
		structMap: model.NewStructsMap(),
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// NewEngine returns an initialized engine ready to kick some ass.
func (db *DB) NewEngine() *engine.Engine {
	e := engine.Get()
	e.StructMap = db.structMap
	e.SingularTable = db.singularTable
	e.Ctx = db.ctx
	e.Dialect = db.dialect
	e.SQLDB = db.db
	e.Now = db.now
	return e
}

//CreateTable creates new database tables that maps to the models.
func (db *DB) CreateTable(models ...interface{}) (sql.Result, error) {
	query, err := db.CreateTableSQL(models...)
	if err != nil {
		return nil, err
	}
	if isQL(db) {
		return db.ExecTx(query.Q, query.Args...)
	}
	return db.SQLCommon().Exec(query.Q, query.Args...)

}

// Verbose prints what is executed on stdout.
//
//DOn't set this to true when in production. It is dog slow, and a security
//risk. Use this only in development
func (db *DB) Verbose(b bool) {
	db.db.Verbose(b)
}

//ExecTx wraps the query execution in a Transaction. This ensure all operations
//are Rolled back in case the execution fails.
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
	if isQL(db) {
		_, _ = buf.WriteString("BEGIN TRANSACTION; \n")
	}
	for _, m := range models {
		e := db.NewEngine()
		defer engine.Put(e)
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
	if isQL(db) {
		_, _ = buf.WriteString("COMMIT;")
	}
	return &model.Expr{Q: buf.String()}, nil
}

func isQL(db *DB) bool {
	return dialects.IsQL(db.Dialect())
}

//DropTableSQL generates sql query for DROP TABLE. The generated query is
//wrapped under TRANSACTION block.
func (db *DB) DropTableSQL(models ...interface{}) (*model.Expr, error) {
	var buf bytes.Buffer
	if isQL(db) {
		_, _ = buf.WriteString("BEGIN TRANSACTION; \n")
	}
	for _, m := range models {
		e := db.NewEngine()
		defer engine.Put(e)
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
	if isQL(db) {
		_, _ = buf.WriteString("COMMIT;")
	}
	return &model.Expr{Q: buf.String()}, nil
}

//DropTable drops tables that are mapped to models. You can also pass the name
//of the table as astring and it will be handled.
func (db *DB) DropTable(models ...interface{}) (sql.Result, error) {
	query, err := db.DropTableSQL(models...)
	if err != nil {
		return nil, err
	}
	if isQL(db) {
		return db.ExecTx(query.Q, query.Args...)
	}
	return db.SQLCommon().Exec(query.Q, query.Args...)
}

//Automigrate creates tables that map to models if the tables don't exist yet in
//the database. This also takes care of situation where the models's fields have
//been updated(changed)
func (db *DB) Automigrate(models ...interface{}) (sql.Result, error) {
	query, err := db.AutomigrateSQL(models...)
	if err != nil {
		return nil, err
	}
	if isQL(db) {
		return db.ExecTx(query.Q, query.Args...)
	}
	return db.SQLCommon().Exec(query.Q, query.Args...)
}

//AutomigrateSQL generates sql query for running migrations on models.
func (db *DB) AutomigrateSQL(models ...interface{}) (*model.Expr, error) {
	// var buf bytes.Buffer
	buf := util.B.Get()
	defer func() {
		util.B.Put(buf)
	}()
	if isQL(db) {
		buf.WriteString("BEGIN TRANSACTION;\n")
	}
	keys := make(map[string]bool)
	for _, m := range models {
		e := db.NewEngine()
		defer engine.Put(e)

		// Firste we generate the SQL
		err := scope.Automigrate(e, m)
		if err != nil {
			return nil, err
		}
		if e.Scope.SQL != "" {
			i := strings.Index(e.Scope.SQL, "(")
			k := e.Scope.SQL[:i]
			if _, ok := keys[k]; !ok {
				buf.WriteString("\t" + e.Scope.SQL + ";\n")
				keys[k] = true
			}
		}
		if e.Scope.MultiExpr {
			for _, expr := range e.Scope.Exprs {
				k := expr.Q
				i := strings.Index(expr.Q, "(")
				if i > 0 {
					k = expr.Q[:i]
				}
				if _, ok := keys[k]; !ok {
					buf.WriteString("\t" + expr.Q + ";\n")
					keys[k] = true
				}
			}
		}
	}
	if isQL(db) {
		buf.WriteString("COMMIT;")
	}
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
// You can hijack the execution of the generated SQL by overriding
// model.HookCreateExec hook.
func (db *DB) Create(value interface{}) error {
	e := db.NewEngine()
	defer engine.Put(e)
	e.Scope.ContextValue(value)
	return hooks.Create(e)
}

//CreateSQL generates SQl query for creating a new record/records for value.
// The end query is wrapped under for ql dialectTRANSACTION block.
func (db *DB) CreateSQL(value interface{}) (*model.Expr, error) {
	var e *engine.Engine
	if db.e != nil {
		e = db.e
	} else {
		e = db.NewEngine()
	}
	defer engine.Put(e)
	e.Scope.ContextValue(value)
	err := hooks.CreateSQL(e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: e.Scope.SQL, Args: e.Scope.SQLVars}, nil
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
	defer engine.Put(e)
	e.Scope.ContextValue(value)
	err := hooks.UpdateSQL(e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: e.Scope.SQL, Args: e.Scope.SQLVars}, nil
}

// Save update value in database, if the value doesn't have primary key, will insert it
func (db *DB) Save(value interface{}) error {
	e := db.NewEngine()
	defer engine.Put(e)
	e.Scope.ContextValue(value)
	field, _ := scope.PrimaryField(e, value)
	if field == nil || field.IsBlank {
		return db.Create(value)
	}
	return hooks.Update(e)
}

//Model sets value as the database model. This model will be used for future
//calls on the returned DB e.g
//
//	db.Model(&user).Update("name","hero")
//
// You don't have to call db.Begin().Model() since this calls Begin automatically for you.
// It is safe for chaining.
func (db *DB) Model(value interface{}) *DB {
	c := db.clone()
	c.e.Scope.ContextValue(value)
	return c
}

//Update runs UPDATE queries.
func (db *DB) Update(attrs ...interface{}) error {
	return db.Updates(util.ToSearchableMap(attrs), true)
}

//Updates runs UPDATE query
func (db *DB) Updates(values interface{}, ignoreProtectedAttrs ...bool) error {
	if db.e == nil || db.e.Scope.Value == nil {
		return errmsg.ErrMissingModel
	}
	defer db.recycle()
	var ignore bool
	if len(ignoreProtectedAttrs) > 0 {
		ignore = ignoreProtectedAttrs[0]
	}
	db.e.Scope.Set(model.IgnoreProtectedAttrs, ignore)
	db.e.Scope.Set(model.UpdateInterface, values)
	return hooks.Update(db.e)
}

//UpdateSQL generates SQL that will be executed when you use db.Update
func (db *DB) UpdateSQL(attrs ...interface{}) (*model.Expr, error) {
	return db.UpdatesSQL(util.ToSearchableMap(attrs), true)
}

//UpdatesSQL generates sql that will be used when you run db.UpdatesSQL
func (db *DB) UpdatesSQL(values interface{}, ignoreProtectedAttrs ...bool) (*model.Expr, error) {
	if db.e == nil || db.e.Scope.Value == nil {
		return nil, errmsg.ErrMissingModel
	}
	defer db.recycle()
	var ignore bool
	if len(ignoreProtectedAttrs) > 0 {
		ignore = ignoreProtectedAttrs[0]
	}
	db.e.Scope.Set(model.IgnoreProtectedAttrs, ignore)
	db.e.Scope.Set(model.UpdateInterface, values)
	err := hooks.UpdateSQL(db.e)
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

//SingularTable enables or disables singular tables name. By default this is
//disabled, meaning table names are in plural.
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
		engine.Put(e)
	}
	return db.Dialect().HasTable(name)
}

//First  fetches the first record and order by primary key.
func (db *DB) First(out interface{}, where ...interface{}) error {
	db.Set(model.OrderByPK, "ASC")
	defer db.recycle()
	search.Inline(db.e, where...)
	search.Limit(db.e, 1)
	db.e.Scope.ContextValue(out)
	return hooks.Query(db.e)
}

//FirstSQL returns SQL query for retrieving the first record ordering by primary
//key.
func (db *DB) FirstSQL(out interface{}, where ...interface{}) (*model.Expr, error) {
	db.Set(model.OrderByPK, "ASC")
	search.Inline(db.e, where...)
	search.Limit(db.e, 1)
	db.e.Scope.ContextValue(out)
	err := hooks.QuerySQL(db.e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: db.e.Scope.SQL, Args: db.e.Scope.SQLVars}, nil
}

//Last finds the last record and order by primary key.
func (db *DB) Last(out interface{}, where ...interface{}) error {
	db.Set(model.OrderByPK, "DESC")
	search.Inline(db.e, where...)
	search.Limit(db.e, 1)
	db.e.Scope.ContextValue(out)
	return hooks.Query(db.e)
}

//LastSQL returns SQL query for retrieving the last record ordering by primary
//key.
func (db *DB) LastSQL(out interface{}, where ...interface{}) (*model.Expr, error) {
	db.Set(model.OrderByPK, "DESC")
	search.Inline(db.e, where...)
	search.Limit(db.e, 1)
	db.e.Scope.ContextValue(out)
	err := hooks.QuerySQL(db.e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: db.e.Scope.SQL, Args: db.e.Scope.SQLVars}, nil
}

// Limit specify the number of records to be retrieved
func (db *DB) Limit(limit interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Limit(db.e, limit)
	return db
}

// FindSQL generates SQL query for  finding records that match given conditions
func (db *DB) FindSQL(out interface{}, where ...interface{}) (*model.Expr, error) {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	defer db.recycle()
	search.Inline(db.e, where...)
	db.e.Scope.ContextValue(out)
	err := hooks.QuerySQL(db.e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: db.e.Scope.SQL, Args: db.e.Scope.SQLVars}, nil
}

// Find find records that match given conditions
func (db *DB) Find(out interface{}, where ...interface{}) error {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	defer db.recycle()
	search.Inline(db.e, where...)
	db.e.Scope.ContextValue(out)
	return hooks.Query(db.e)
}

// Attrs initialize struct with argument if record not found
func (db *DB) Attrs(attrs ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Attr(db.e, attrs...)
	return db
}

// Assign assign result with argument regardless it is found or not
func (db *DB) Assign(attrs ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Assign(db.e, attrs...)
	return db
}

// Group specify the group method on the find
func (db *DB) Group(query string) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	_ = search.Group(db.e, query)
	return db
}

// Having specify HAVING conditions for GROUP BY
func (db *DB) Having(query string, values ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Having(db.e, query, values...)
	return db
}

// Joins specify Joins conditions
func (db *DB) Joins(query string, args ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Join(db.e, query, args...)
	return db
}

// Offset specify the number of records to skip before starting to return the records
func (db *DB) Offset(offset interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Offset(db.e, offset)
	return db
}

// Order specify order when retrieve records from database, set reorder to
// `true` to overwrite defined conditions
//     db.Order("name DESC")
//     db.Order("name DESC", true) // reorder
func (db *DB) Order(value interface{}, reorder ...bool) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Order(db.e, value, reorder...)
	return db
}

// Select specify fields that you want to retrieve from database when querying,
// by default, will select all fields; When creating/updating, specify fields
// that you want to save to database
func (db *DB) Select(query interface{}, args ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Select(db.e, query, args...)
	return db
}

// Omit specify fields that you want to ignore when saving to database for
// creating, updating
func (db *DB) Omit(columns ...string) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Omit(db.e, columns...)
	return db
}

// Not filter records that don't match current conditions, similar to `Where`
func (db *DB) Not(query interface{}, args ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Not(db.e, query, args...)
	return db
}

// Or filter records that match before conditions or this one, similar to `Where`
func (db *DB) Or(query interface{}, args ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Or(db.e, query, args...)
	return db
}

// Where return a new relation, filter records with given conditions, accepts
//`map`, `struct` or `string` as conditions
func (db *DB) Where(query interface{}, args ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Where(db.e, query, args...)
	return db
}

// FirstOrInit find first matched record or initialize a new one with given
//conditions (only works with struct, map conditions)
func (db *DB) FirstOrInit(out interface{}, where ...interface{}) error {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	defer db.recycle()
	db.e.Scope.ContextValue(out)
	err := db.Begin().First(out, where...)
	if err != nil {
		if err != errmsg.ErrRecordNotFound {
			return err
		}
		search.Inline(db.e, where...)
		scope.Initialize(db.e)
		return nil
	}
	_, _ = scope.UpdatedAttrsWithValues(db.e, db.e.Search.AssignAttrs)
	return nil
}

// Begin gives back a fresh copy of DB ready for chaining methods that operates
// on the same model..
func (db *DB) Begin() *DB {
	return db.clone()
}

func (db *DB) recycle() {
	engine.Put(db.e)
	db.e = nil
}

// Table specify the table you would like to run db operations
func (db *DB) Table(name string) *DB {
	ndb := db.Begin()
	search.Table(ndb.e, name)
	return ndb
}

// Pluck used to query single column from a model as a map
//     var ages []int64
//     db.Find(&users).Pluck("age", &ages)
func (db *DB) Pluck(column string, value interface{}) error {
	dest := reflect.ValueOf(value)
	if dest.Kind() == reflect.Ptr {
		dest = dest.Elem()
	}
	defer db.recycle()
	search.Select(db.e, column)
	if dest.Kind() != reflect.Slice {
		return fmt.Errorf("results should be a slice, not %s", dest.Kind())
	}
	err := builder.PrepareQuery(db.e, db.e.Scope.Value)
	if err != nil {
		return err
	}
	rows, err := db.SQLCommon().Query(db.e.Scope.SQL, db.e.Scope.SQLVars...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		elem := reflect.New(dest.Type().Elem()).Interface()
		err := rows.Scan(elem)
		if err != nil {
			return err
		}
		dest.Set(reflect.Append(dest, reflect.ValueOf(elem).Elem()))
	}
	return nil
}

// Count get how many records for a model
func (db *DB) Count(value interface{}) error {
	query, ok := db.e.Search.Selects["query"]
	if !ok || regexes.CountingQuery.MatchString(fmt.Sprint(query)) {
		search.Select(db.e, "count(*)")
	}
	defer db.recycle()
	db.e.Search.IgnoreOrderQuery = true
	err := builder.PrepareQuery(db.e, db.e.Scope.Value)
	if err != nil {
		return err
	}
	return db.SQLCommon().QueryRow(db.e.Scope.SQL, db.e.Scope.SQLVars...).Scan(value)
}

// AddIndexSQL generates SQL to add index for columns with given name
func (db *DB) AddIndexSQL(indexName string, columns ...string) (*model.Expr, error) {
	if db.e == nil || db.e.Scope.Value == nil {
		return nil, errmsg.ErrMissingModel
	}
	err := builder.AddIndex(db.e, false, indexName, columns...)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: db.e.Scope.SQL, Args: db.e.Scope.SQLVars}, nil
}

// AddIndex add index for columns with given name
func (db *DB) AddIndex(indexName string, columns ...string) (sql.Result, error) {
	sql, err := db.AddIndexSQL(indexName, columns...)
	if err != nil {
		return nil, err
	}
	if isQL(db) {
		return db.ExecTx(util.WrapTX(sql.Q), sql.Args...)
	}
	return db.SQLCommon().Exec(sql.Q, sql.Args...)
}

// DropTableIfExists drop table if it is exist
func (db *DB) DropTableIfExists(values ...interface{}) error {
	for _, value := range values {
		if db.HasTable(value) {
			_, err := db.Begin().DropTable(value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Delete delete value match given conditions, if the value has primary key,
//then will including the primary key as condition
func (db *DB) Delete(value interface{}, where ...interface{}) error {
	e := db.NewEngine()
	defer engine.Put(e)
	e.Scope.ContextValue(value)
	search.Inline(e, where...)
	return hooks.Delete(e)
}

// DeleteSQL  generates SQL to delete value match given conditions, if the value has primary key,
//then will including the primary key as condition
func (db *DB) DeleteSQL(value interface{}, where ...interface{}) (*model.Expr, error) {
	e := db.NewEngine()
	defer engine.Put(e)
	e.Scope.ContextValue(value)
	search.Inline(e, where...)
	err := hooks.DeleteSQL(e)
	if err != nil {
		return nil, err
	}
	return &model.Expr{Q: e.Scope.SQL, Args: e.Scope.SQLVars}, nil
}

// UpdateColumn update attributes without callbacks
func (db *DB) UpdateColumn(attrs ...interface{}) error {
	return db.UpdateColumns(util.ToSearchableMap(attrs...))
}

// UpdateColumns update attributes without
func (db *DB) UpdateColumns(values interface{}) error {
	if db.e == nil || db.e.Scope.Value == nil {
		return errmsg.ErrMissingModel
	}
	defer db.recycle()
	db.e.Scope.Set(model.UpdateColumn, true)
	db.e.Scope.Set(model.SaveAssociations, false)
	db.e.Scope.Set(model.UpdateInterface, values)
	return hooks.Update(db.e)
}

// AddUniqueIndex add unique index for columns with given name
func (db *DB) AddUniqueIndex(indexName string, columns ...string) (sql.Result, error) {
	if db.e == nil || db.e.Scope.Value == nil {
		return nil, errmsg.ErrMissingModel
	}
	err := builder.AddIndex(db.e, true, indexName, columns...)
	if err != nil {
		return nil, err
	}
	if isQL(db) {
		return db.ExecTx(util.WrapTX(db.e.Scope.SQL), db.e.Scope.SQLVars...)
	}
	return db.SQLCommon().Exec(db.e.Scope.SQL, db.e.Scope.SQLVars...)
}

// RemoveIndex remove index with name
func (db *DB) RemoveIndex(indexName string) error {
	if db.e == nil || db.e.Scope.Value == nil {
		return errmsg.ErrMissingModel
	}
	defer db.recycle()
	return db.Dialect().RemoveIndex(
		scope.TableName(db.e, db.e.Scope.Value), indexName)
}

// DropColumn drop a column
func (db *DB) DropColumn(column string) (sql.Result, error) {
	if db.e == nil || db.e.Scope.Value == nil {
		return nil, errmsg.ErrMissingModel
	}
	defer db.recycle()
	db.e.Scope.SQL = fmt.Sprintf("ALTER TABLE %v DROP COLUMN %v",
		scope.QuotedTableName(db.e, db.e.Scope.Value), scope.Quote(db.e, column))
	if isQL(db) {
		return db.ExecTx(
			util.WrapTX(db.e.Scope.SQL), db.e.Scope.SQLVars...,
		)
	}
	return db.SQLCommon().Exec(db.e.Scope.SQL, db.e.Scope.SQLVars...)
}

// ModifyColumn modify column to type
func (db *DB) ModifyColumn(column string, typ string) (sql.Result, error) {
	if db.e == nil || db.e.Scope.Value == nil {
		return nil, errmsg.ErrMissingModel
	}
	defer db.recycle()
	if isQL(db) {
		return nil, errors.New("ngorm: ql does to support MODIFY column")
	}
	db.e.Scope.SQL = fmt.Sprintf("ALTER TABLE %v MODIFY %v %v",
		scope.QuotedTableName(db.e, db.e.Scope.Value), scope.Quote(db.e, column), typ)
	return db.ExecTx(
		util.WrapTX(db.e.Scope.SQL), db.e.Scope.SQLVars...,
	)
}

// Ping checks if you can connect to the database
func (db *DB) Ping() error {
	if dr, ok := db.db.SQLCommon.(*sql.DB); ok {
		return dr.Ping()
	}
	return errors.New("ngorm: ping not supported")
}

// Preload preload associations with given conditions
//    db.Preload("Orders", "state NOT IN (?)", "cancelled").Find(&users)
func (db *DB) Preload(column string, conditions ...interface{}) *DB {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	search.Preload(db.e, column, conditions...)
	return db
}

// FirstOrCreate find first matched record or create a new one with given
//conditions (only works with struct, map conditions)
func (db *DB) FirstOrCreate(out interface{}, where ...interface{}) error {
	if db.e == nil {
		db.e = db.NewEngine()
	}
	defer db.recycle()
	db.e.Scope.ContextValue(out)
	err := db.Begin().First(out, where...)
	if err != nil {
		if err != errmsg.ErrRecordNotFound {
			return err
		}

		// re use the existing engine
		db.e.Scope.SQLVars = nil
		db.e.Scope.SQL = ""

		search.Inline(db.e, where...)
		scope.Initialize(db.e)
		return db.Create(out)
	}
	if len(db.e.Search.AssignAttrs) > 0 {
		return db.Update(db.e.Search.AssignAttrs...)
	}
	return nil
}

// AddForeignKey adds foreign key to an existing table.
func (db *DB) AddForeignKey(field string, dest string, onDelete string, onUpdate string) error {
	sql, err := db.AddForeignKeySQL(field, dest, onDelete, onUpdate)
	if err != nil {
		return err
	}
	_, err = db.SQLCommon().Exec(sql)
	if err != nil {
		return fmt.Errorf("%v \n %s", err, sql)
	}
	return nil
}

// AddForeignKeySQL generates sql to adds foreign key to an existing table.
func (db *DB) AddForeignKeySQL(field string, dest string, onDelete string, onUpdate string) (string, error) {
	if db.e == nil || db.e.Scope.Value == nil {
		return "", errmsg.ErrMissingModel
	}
	defer db.recycle()
	if isQL(db) {
		return "", errors.New("ql does not support foreign key")
	}
	name := scope.TableName(db.e, db.e.Scope.Value)
	keyName := db.Dialect().BuildForeignKeyName(
		name, field, dest)

	if db.Dialect().HasForeignKey(name, keyName) {
		return "", errors.New("key already exists")
	}
	var query = `ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s;`
	sql := fmt.Sprintf(query,
		scope.QuotedTableName(db.e, db.e.Scope.Value),
		scope.Quote(db.e, keyName),
		scope.Quote(db.e, field), dest, onDelete, onUpdate)
	return sql, nil
}

// Association returns association object
func (db *DB) Association(column string) (*Association, error) {
	if db.e == nil || db.e.Scope.Value == nil {
		return nil, errmsg.ErrMissingModel
	}
	p, err := scope.PrimaryField(db.e, db.e.Scope.Value)
	if err != nil {
		return nil, err
	}
	if p.IsBlank {
		return nil, errors.New("primary field can not be blank")
	}
	field, err := scope.FieldByName(db.e, db.e.Scope.Value, column)
	if err != nil {
		return nil, err
	}
	ndb := db.Begin()
	ndb.e.Scope.ContextValue(db.e.Scope.Value)
	ndb.e.Scope.Set(model.AssociationSource, db.e.Scope.Value)
	if field.Relationship == nil || len(field.Relationship.ForeignFieldNames) == 0 {
		v := db.e.Scope.ValueOf()
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		return nil, fmt.Errorf("invalid association %v for %v", column, v.Type())
	}
	return &Association{db: ndb, column: column, field: field}, nil
}

func (db *DB) related(source, value interface{}, foreignKeys ...string) error {
	sdb := db.Begin()
	sdb.e.Scope.ContextValue(source)
	ndb := db.Begin()
	ndb.e.Scope.ContextValue(value)
	sdb.e.Scope.Set(model.AssociationSource, source)

	foreignKeys = append(foreignKeys, ndb.e.Scope.TypeName()+"ID")
	foreignKeys = append(foreignKeys, sdb.e.Scope.TypeName()+"ID")

	for _, foreignKey := range foreignKeys {
		fromField, err := scope.FieldByName(sdb.e, sdb.e.Scope.ValueOf(), foreignKey)
		if err != nil {
			toField, err := scope.FieldByName(ndb.e, value, foreignKey)
			if err != nil {
				continue
			}
			var pfv interface{}
			pk, err := scope.PrimaryField(sdb.e, source)
			if err != nil {
				pfv = 0
			} else {
				pfv = pk.Field.Interface()
			}
			sql := fmt.Sprintf("%v = ?",
				scope.Quote(ndb.e, toField.DBName))
			return ndb.Where(sql, pfv).Find(value)
		}

		if rel := fromField.Relationship; rel != nil {
			if rel.Kind == "many_to_many" {
				h := rel.JoinTableHandler
				if isQL(db) {
					err = scope.JoinWithQL(h, ndb.e, sdb.e.Scope.Value)
					if err != nil {
						return err
					}
				} else {
					err = scope.JoinWith(h, ndb.e, sdb.e.Scope.Value)
					if err != nil {
						return err
					}
				}

				return ndb.Find(value)
			} else if rel.Kind == "belongs_to" {
				for idx, foreignKey := range rel.ForeignDBNames {
					if field, ok := scope.FieldByName(sdb.e, sdb.e.Scope.ValueOf(), foreignKey); ok == nil {
						ndb = ndb.Where(fmt.Sprintf("%v = ?",
							scope.Quote(ndb.e, rel.AssociationForeignDBNames[idx])),
							field.Field.Interface())
					}
				}
				return ndb.Find(value)
			} else if rel.Kind == "has_many" || rel.Kind == "has_one" {
				for idx, foreignKey := range rel.ForeignDBNames {
					field, err := scope.FieldByName(sdb.e, sdb.e.Scope.ValueOf(), rel.AssociationForeignDBNames[idx])
					if err == nil {
						ndb = ndb.Where(fmt.Sprintf("%v = ?",
							scope.Quote(ndb.e, foreignKey)), field.Field.Interface())
					}

				}

				if rel.PolymorphicType != "" {
					ndb = ndb.Where(fmt.Sprintf("%v = ?",
						scope.Quote(ndb.e, rel.PolymorphicDBName)), rel.PolymorphicValue)
				}
				return ndb.Find(value)
			}
		} else {
			pk, err := scope.PrimaryKey(sdb.e, value)
			if err != nil {
				return err
			}
			sql := fmt.Sprintf("%v = ?",
				scope.Quote(sdb.e, pk))
			return ndb.Where(sql, fromField.Field.Interface()).Find(value)
		}
		return nil

	}
	return fmt.Errorf("invalid association %v", foreignKeys)
}

// Related get related associations
func (db *DB) Related(value interface{}, foreignKeys ...string) error {
	if db.e == nil || db.e.Scope.Value == nil {
		return errmsg.ErrMissingModel
	}
	return db.related(db.e.Scope.Value, value, foreignKeys...)
}
