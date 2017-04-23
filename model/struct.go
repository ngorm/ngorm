//Package model defines structures that are shared through out ngorm. They
//provide the base building blocks for ngorm abstractions.
package model

import (
	"database/sql"
	"reflect"
	"strings"
	"sync"
	"time"
)

// All important keys
const (
	OrderByPK               = "ngorm:order_by_primary_key"
	QueryDestination        = "ngorm:query_destination"
	QueryOption             = "ngorm:query_option"
	Query                   = "ngorm:query"
	HookAfterQuery          = "ngorm:query_after"
	HookQuerySQL            = "ngorm:query_sql_hook"
	HookQueryExec           = "ngorm:query_sql_exec"
	HookAfterFindQuery      = "ngorm:query_after_find"
	HookBeforeCreate        = "ngorm:before_create_hook"
	HookBeforeSave          = "ngorm:before_save_hook"
	Create                  = "ngorm:create"
	HookCreateExec          = "ngorm:create_exec"
	BeforeCreate            = "ngorm:before_create"
	AfterCreate             = "ngorm:after_create"
	HookAfterCreate         = "ngorm:after_create"
	HookAfterSave           = "ngorm:after_save_hook"
	UpdateAttrs             = "ngorm:update_attrs"
	TableOptions            = "ngorm:table_options"
	HookSaveBeforeAss       = "ngorm:save_before_associations"
	HookUpdateTimestamp     = "ngorm:update_time_stamp"
	BlankColWithValue       = "ngorm:blank_columns_with_default_value"
	InsertOptions           = "ngorm:insert_option"
	UpdateColumn            = "ngorm:update_column"
	HookBeforeUpdate        = "ngorm:before_update_hook"
	HookAfterUpdate         = "ngorm:after_update_hook"
	UpdateInterface         = "ngorm:update_interface"
	BeforeUpdate            = "ngorm:before_update"
	AfterUpdate             = "ngorm:after_update"
	HookAssignUpdatingAttrs = "ngorm:assign_updating_attrs_hook"
	HookCreateSQL           = "ngorm:create_sql"
	UpdateOptions           = "ngorm:update_option"
	Update                  = "ngorm:update"
	HookUpdateSQL           = "ngorm:update_sql_hook"
	HookUpdateExec          = "ngorm:update_exec_hook"
	IgnoreProtectedAttrs    = "ngorm:ignore_protected_attrs"
	DeleteOption            = "ngorm:delete_options"
	BeforeDelete            = "ngorm:before_delete"
	HookBeforeDelete        = "ngorm:before_delete_hook"
	AfterDelete             = "ngorm:after_delete"
	HookAfterDelete         = "ngorm:after_delete_hook"
	Delete                  = "ngorm:delete"
	DeleteSQL               = "ngorm:delete_sql"
	SaveAssociations        = "ngorm:save_associations"
	Preload                 = "ngorm:preload"
	HookSaveAfterAss        = "ngorm:save_after_association"
	AssociationSource       = "ngorm:association:source"
)

//Model defines common fields that are used for defining SQL Tables. This is a
//helper that you can embed in your own struct definition.
//
// By embedding this, there is no need to define the supplied fields. For
// example.
//
//  type User struct {
//    Model
//    Name string
//  }
// Is the same as this
//  type User   struct {
//    ID        uint `gorm:"primary_key"`
//    CreatedAt time.Time
//    UpdatedAt time.Time
//    DeletedAt *time.Time `sql:"index"`
//    Name      string
//  }
type Model struct {
	ID        int64 `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

//Struct model definition
type Struct struct {
	PrimaryFields    []*StructField
	StructFields     []*StructField
	ModelType        reflect.Type
	DefaultTableName string
}

// StructField model field's struct definition
type StructField struct {

	// DBName is the name of the field as it is seen in the database, for
	// instance a field ID can be represented in the database as id.
	DBName          string
	Name            string
	Names           []string
	IsPrimaryKey    bool
	IsNormal        bool
	IsIgnored       bool
	IsScanner       bool
	HasDefaultValue bool
	Tag             reflect.StructTag
	TagSettings     map[string]string
	Struct          reflect.StructField
	IsForeignKey    bool
	Relationship    *Relationship
}

//Clone retruns a deep copy of the StructField
func (s *StructField) Clone() *StructField {
	clone := &StructField{
		DBName:          s.DBName,
		Name:            s.Name,
		Names:           s.Names,
		IsPrimaryKey:    s.IsPrimaryKey,
		IsNormal:        s.IsNormal,
		IsIgnored:       s.IsIgnored,
		IsScanner:       s.IsScanner,
		HasDefaultValue: s.HasDefaultValue,
		Tag:             s.Tag,
		TagSettings:     map[string]string{},
		Struct:          s.Struct,
		IsForeignKey:    s.IsForeignKey,
		Relationship:    s.Relationship,
	}

	for key, value := range s.TagSettings {
		clone.TagSettings[key] = value
	}

	return clone
}

// Relationship described the relationship between models
type Relationship struct {
	Kind                         string
	PolymorphicType              string
	PolymorphicDBName            string
	PolymorphicValue             string
	ForeignFieldNames            []string
	ForeignDBNames               []string
	AssociationForeignFieldNames []string
	AssociationForeignDBNames    []string
	JoinTableHandler             *JoinTableHandler
}

//ParseTagSetting returns a map[string]string for the tags that are set.
func ParseTagSetting(tags reflect.StructTag) map[string]string {
	setting := map[string]string{}
	for _, str := range []string{tags.Get("sql"), tags.Get("gorm")} {
		tags := strings.Split(str, ";")
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
			if len(v) >= 2 {
				setting[k] = strings.Join(v[1:], ":")
			} else {
				setting[k] = k
			}
		}
	}
	return setting
}

//SafeStructsMap provide safe storage and accessing of *Struct.
type SafeStructsMap struct {
	v  []*Struct
	mu sync.RWMutex
}

//Set safely stores value
func (s *SafeStructsMap) Set(value *Struct) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v = append(s.v, value)
}

//Get retrieves the value stored with the given key.
func (s *SafeStructsMap) Get(key reflect.Type) *Struct {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.v {
		if v.ModelType == key {
			return v
		}
	}
	return nil
}

//NewStructsMap returns a safe map for storing *Struct objects.
func NewStructsMap() *SafeStructsMap {
	return &SafeStructsMap{}
}

//Scope is the scope level of SQL building.
type Scope struct {
	Value       interface{}
	SQL         string
	SQLVars     []interface{}
	SelectAttrs *[]string
	MultiExpr   bool
	Exprs       []*Expr
	mu          sync.RWMutex
	data        map[string]interface{}
}

//NewScope return an empty scope. The scope is initialized to allow Set, and Get
//methods to work.
func NewScope() *Scope {
	return &Scope{
		data: make(map[string]interface{}),
	}
}

//Set sets a scope specific key value. This is only available in the scope.
func (s *Scope) Set(key string, value interface{}) {
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()
}

//Get retrieves the value with key from the scope.
func (s *Scope) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	v, ok := s.data[key]
	s.mu.RUnlock()
	return v, ok
}

//GetAll returns all values stored in this context.
func (s *Scope) GetAll() map[string]interface{} {
	s.mu.RLock()
	a := make(map[string]interface{})
	for k, v := range s.data {
		a[k] = v
	}
	s.mu.RUnlock()
	return a
}

// TypeName returns the name of the type contained in Scope.Value
func (s *Scope) TypeName() string {
	val := reflect.ValueOf(s.Value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	for typ.Kind() == reflect.Slice || typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}

//Search is the search level of SQL building
type Search struct {
	WhereConditions  []map[string]interface{}
	OrConditions     []map[string]interface{}
	NotConditions    []map[string]interface{}
	HavingConditions []map[string]interface{}
	JoinConditions   []map[string]interface{}
	InitAttrs        []interface{}
	AssignAttrs      []interface{}
	Selects          map[string]interface{}
	Omits            []string
	Orders           []interface{}
	Preload          []SearchPreload
	Offset           interface{}
	Limit            interface{}
	Group            string
	TableName        string
	TableNames       []string
	Raw              bool
	Unscoped         bool
	IgnoreOrderQuery bool
}

//SearchPreload is the preload search condition.
type SearchPreload struct {
	Schema     string
	Conditions []interface{}
}

//SQLCommon is the interface for SQL database interactions.
type SQLCommon interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Begin() (*sql.Tx, error)
	Close() error
}

// Expr is SQL expression
type Expr struct {
	Q    string
	Args []interface{}
}

//JoinTableForeignKey info that point to a key to use in join table.
type JoinTableForeignKey struct {
	DBName            string
	AssociationDBName string
}

// JoinTableSource is a struct that contains model type and foreign keys
type JoinTableSource struct {
	ModelType   reflect.Type
	ForeignKeys []JoinTableForeignKey
}

// JoinTableHandler default join table handler
type JoinTableHandler struct {
	TableName   string          `sql:"-"`
	Source      JoinTableSource `sql:"-"`
	Destination JoinTableSource `sql:"-"`
}
