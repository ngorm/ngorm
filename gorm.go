package gorm

import (
	"database/sql"

	"github.com/gernest/ngorm/callback"
	"github.com/gernest/ngorm/dialects"
	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/logger"
)

// DB contains information for current db connection
type DB struct {
}

// Open initialize a new db connection, need to import driver first, e.g:
//
//     import _ "github.com/go-sql-driver/mysql"
//     func main() {
//       db, err := gorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local")
//     }
// GORM has wrapped some drivers, for easier to remember driver's import path, so you could import the mysql driver with
//    import _ "github.com/jinzhu/gorm/dialects/mysql"
//    // import _ "github.com/jinzhu/gorm/dialects/postgres"
//    // import _ "github.com/jinzhu/gorm/dialects/sqlite"
//    // import _ "github.com/jinzhu/gorm/dialects/mssql"
func Open(dialect string, args ...interface{}) (*DB, error) {
	return nil, nil
}

// Close close current db connection
func (s *DB) Close() error {
	return nil
}

// DB get `*sql.DB` from current connection
func (s *DB) DB() *sql.DB {
	return nil
}

// Dialect get dialect
func (s *DB) Dialect() dialects.Dialect {
	return nil
}

// New clone a new db connection without search conditions
func (s *DB) New() *DB {
	return nil
}

// NewScope create a scope for current operation
func (s *DB) NewScope(value interface{}) *engine.Scope {
	return nil
}

// Callback return `Callbacks` container, you could add/change/delete callbacks with it
//     db.Callback().Create().Register("update_created_at", updateCreated)
// Refer https://jinzhu.github.io/gorm/development.html#callbacks
func (s *DB) Callback() *callback.Callback {
	return nil
}

// SetLogger replace default logger
func (s *DB) SetLogger(log logger.Logger) {
}

// LogMode set log mode, `true` for detailed logs, `false` for no log, default, will only print error logs
func (s *DB) LogMode(enable bool) *DB {
	return nil
}

// BlockGlobalUpdate if true, generates an error on update/delete without where clause.
// This is to prevent eventual error with empty objects updates/deletions
func (s *DB) BlockGlobalUpdate(enable bool) *DB {
	return nil
}

// HasBlockGlobalUpdate return state of block
func (s *DB) HasBlockGlobalUpdate() bool {
	return false
}

// SingularTable use singular table by default
func (s *DB) SingularTable(enable bool) {
}

// Where return a new relation, filter records with given conditions, accepts `map`, `struct` or `string` as conditions, refer http://jinzhu.github.io/gorm/curd.html#query
func (s *DB) Where(query interface{}, args ...interface{}) *DB {
	return nil
}

// Or filter records that match before conditions or this one, similar to `Where`
func (s *DB) Or(query interface{}, args ...interface{}) *DB {
	return nil
}

// Not filter records that don't match current conditions, similar to `Where`
func (s *DB) Not(query interface{}, args ...interface{}) *DB {
	return nil
}

// Limit specify the number of records to be retrieved
func (s *DB) Limit(limit interface{}) *DB {
	return nil
}

// Offset specify the number of records to skip before starting to return the records
func (s *DB) Offset(offset interface{}) *DB {
	return nil
}

// Order specify order when retrieve records from database, set reorder to `true` to overwrite defined conditions
//     db.Order("name DESC")
//     db.Order("name DESC", true) // reorder
//     db.Order(gorm.Expr("name = ? DESC", "first")) // sql expression
func (s *DB) Order(value interface{}, reorder ...bool) *DB {
	return nil
}

// Select specify fields that you want to retrieve from database when querying, by default, will select all fields;
// When creating/updating, specify fields that you want to save to database
func (s *DB) Select(query interface{}, args ...interface{}) *DB {
	return nil
}

// Omit specify fields that you want to ignore when saving to database for creating, updating
func (s *DB) Omit(columns ...string) *DB {
	return nil
}

// Group specify the group method on the find
func (s *DB) Group(query string) *DB {
	return nil
}

// Having specify HAVING conditions for GROUP BY
func (s *DB) Having(query string, values ...interface{}) *DB {
	return nil
}

// Joins specify Joins conditions
//     db.Joins("JOIN emails ON emails.user_id = users.id AND emails.email = ?", "jinzhu@example.org").Find(&user)
func (s *DB) Joins(query string, args ...interface{}) *DB {
	return nil
}

// Scopes pass current database connection to arguments `func(*DB) *DB`, which could be used to add conditions dynamically
//     func AmountGreaterThan1000(db *gorm.DB) *gorm.DB {
//         return db.Where("amount > ?", 1000)
//     }
//
//     func OrderStatus(status []string) func (db *gorm.DB) *gorm.DB {
//         return func (db *gorm.DB) *gorm.DB {
//             return db.Scopes(AmountGreaterThan1000).Where("status in (?)", status)
//         }
//     }
//
//     db.Scopes(AmountGreaterThan1000, OrderStatus([]string{"paid", "shipped"})).Find(&orders)
// Refer https://jinzhu.github.io/gorm/curd.html#scopes
func (s *DB) Scopes(funcs ...func(*DB) *DB) *DB {
	return nil
}

// Unscoped return all record including deleted record, refer Soft Delete https://jinzhu.github.io/gorm/curd.html#soft-delete
func (s *DB) Unscoped() *DB {
	return nil
}

// Attrs initialize struct with argument if record not found with `FirstOrInit` https://jinzhu.github.io/gorm/curd.html#firstorinit or `FirstOrCreate` https://jinzhu.github.io/gorm/curd.html#firstorcreate
func (s *DB) Attrs(attrs ...interface{}) *DB {
	return nil
}

// Assign assign result with argument regardless it is found or not with `FirstOrInit` https://jinzhu.github.io/gorm/curd.html#firstorinit or `FirstOrCreate` https://jinzhu.github.io/gorm/curd.html#firstorcreate
func (s *DB) Assign(attrs ...interface{}) *DB {
	return nil
}

// First find first record that match given conditions, order by primary key
func (s *DB) First(out interface{}, where ...interface{}) *DB {
	return nil
}

// Last find last record that match given conditions, order by primary key
func (s *DB) Last(out interface{}, where ...interface{}) *DB {
	return nil
}

// Find find records that match given conditions
func (s *DB) Find(out interface{}, where ...interface{}) *DB {
	return nil
}

// Scan scan value to a struct
func (s *DB) Scan(dest interface{}) *DB {
	return nil
}

// Row return `*sql.Row` with given conditions
func (s *DB) Row() *sql.Row {
	return nil
}

// Rows return `*sql.Rows` with given conditions
func (s *DB) Rows() (*sql.Rows, error) {
	return nil, nil
}

// ScanRows scan `*sql.Rows` to give struct
func (s *DB) ScanRows(rows *sql.Rows, result interface{}) error {
	return nil
}

// Pluck used to query single column from a model as a map
//     var ages []int64
//     db.Find(&users).Pluck("age", &ages)
func (s *DB) Pluck(column string, value interface{}) *DB {
	return nil
}

// Count get how many records for a model
func (s *DB) Count(value interface{}) *DB {
	return nil
}

// Related get related associations
func (s *DB) Related(value interface{}, foreignKeys ...string) *DB {
	return nil
}

// FirstOrInit find first matched record or initialize a new one with given conditions (only works with struct, map conditions)
// https://jinzhu.github.io/gorm/curd.html#firstorinit
func (s *DB) FirstOrInit(out interface{}, where ...interface{}) *DB {
	return nil
}

// FirstOrCreate find first matched record or create a new one with given conditions (only works with struct, map conditions)
// https://jinzhu.github.io/gorm/curd.html#firstorcreate
func (s *DB) FirstOrCreate(out interface{}, where ...interface{}) *DB {
	return nil
}

// Update update attributes with callbacks, refer: https://jinzhu.github.io/gorm/curd.html#update
func (s *DB) Update(attrs ...interface{}) *DB {
	return nil
}

// Updates update attributes with callbacks, refer: https://jinzhu.github.io/gorm/curd.html#update
func (s *DB) Updates(values interface{}, ignoreProtectedAttrs ...bool) *DB {
	return nil
}

// UpdateColumn update attributes without callbacks, refer: https://jinzhu.github.io/gorm/curd.html#update
func (s *DB) UpdateColumn(attrs ...interface{}) *DB {
	return nil
}

// UpdateColumns update attributes without callbacks, refer: https://jinzhu.github.io/gorm/curd.html#update
func (s *DB) UpdateColumns(values interface{}) *DB {
	return nil
}

// Save update value in database, if the value doesn't have primary key, will insert it
func (s *DB) Save(value interface{}) *DB {
	return nil
}

// Create insert the value into database
func (s *DB) Create(value interface{}) *DB {
	return nil
}

// Delete delete value match given conditions, if the value has primary key, then will including the primary key as condition
func (s *DB) Delete(value interface{}, where ...interface{}) *DB {
	return nil
}

// Raw use raw sql as conditions, won't run it unless invoked by other methods
//    db.Raw("SELECT name, age FROM users WHERE name = ?", 3).Scan(&result)
func (s *DB) Raw(sql string, values ...interface{}) *DB {
	return nil
}

// Exec execute raw sql
func (s *DB) Exec(sql string, values ...interface{}) *DB {
	return nil
}

// Model specify the model you would like to run db operations
//    // update all users's name to `hello`
//    db.Model(&User{}).Update("name", "hello")
//    // if user's primary key is non-blank, will use it as condition, then will only update the user's name to `hello`
//    db.Model(&user).Update("name", "hello")
func (s *DB) Model(value interface{}) *DB {
	return nil
}

// Table specify the table you would like to run db operations
func (s *DB) Table(name string) *DB {
	return nil
}

// Debug start debug mode
func (s *DB) Debug() *DB {
	return nil
}

// Begin begin a transaction
func (s *DB) Begin() *DB {
	return nil
}

// Commit commit a transaction
func (s *DB) Commit() *DB {
	return nil
}

// Rollback rollback a transaction
func (s *DB) Rollback() *DB {
	return nil
}

// NewRecord check if value's primary key is blank
func (s *DB) NewRecord(value interface{}) bool {
	return false
}

// RecordNotFound check if returning ErrRecordNotFound error
func (s *DB) RecordNotFound() bool {
	return false
}

// CreateTable create table for models
func (s *DB) CreateTable(models ...interface{}) *DB {
	return nil
}

// DropTable drop table for models
func (s *DB) DropTable(values ...interface{}) *DB {
	return nil
}

// DropTableIfExists drop table if it is exist
func (s *DB) DropTableIfExists(values ...interface{}) *DB {
	return nil
}

// HasTable check has table or not
func (s *DB) HasTable(value interface{}) bool {
	return false
}

// AutoMigrate run auto migration for given models, will only add missing fields, won't delete/change current data
func (s *DB) AutoMigrate(values ...interface{}) *DB {
	return nil
}

// ModifyColumn modify column to type
func (s *DB) ModifyColumn(column string, typ string) *DB {
	return nil
}

// DropColumn drop a column
func (s *DB) DropColumn(column string) *DB {
	return nil
}

// AddIndex add index for columns with given name
func (s *DB) AddIndex(indexName string, columns ...string) *DB {
	return nil
}

// AddUniqueIndex add unique index for columns with given name
func (s *DB) AddUniqueIndex(indexName string, columns ...string) *DB {
	return nil
}

// RemoveIndex remove index with name
func (s *DB) RemoveIndex(indexName string) *DB {
	return nil
}

// AddForeignKey Add foreign key to the given scope, e.g:
//     db.Model(&User{}).AddForeignKey("city_id", "cities(id)", "RESTRICT", "RESTRICT")
func (s *DB) AddForeignKey(field string, dest string, onDelete string, onUpdate string) *DB {
	return nil
}

// Association start `Association Mode` to handler relations things easir in that mode, refer: https://jinzhu.github.io/gorm/associations.html#association-mode
//func (s *DB) Association(column string) *Association {
//return nil
//}

// Preload preload associations with given conditions
//    db.Preload("Orders", "state NOT IN (?)", "cancelled").Find(&users)
func (s *DB) Preload(column string, conditions ...interface{}) *DB {
	return nil
}

// Set set setting by name, which could be used in callbacks, will clone a new db, and update its setting
func (s *DB) Set(name string, value interface{}) *DB {
	return nil
}

// InstantSet instant set setting, will affect current db
func (s *DB) InstantSet(name string, value interface{}) *DB {
	return nil
}

// Get get setting by name
//func (s *DB) Get(name string) (value interface{}, ok bool) {
//}

//// SetJoinTableHandler set a model's join table handler for a relation
//func (s *DB) SetJoinTableHandler(source interface{}, column string, handler JoinTableHandlerInterface) {
//}

// AddError add error to the db
func (s *DB) AddError(err error) error {
	return nil
}

// GetErrors get happened errors from the db
func (s *DB) GetErrors() []error {
	return nil
}
