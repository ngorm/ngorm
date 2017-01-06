package model

import (
	"database/sql"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	OrderByPK          = "ngorm:order_by_primary_key"
	QueryDestination   = "ngorm:query_destination"
	QueryOption        = "ngorm:query_option"
	Query              = "ngorm:query"
	HookQueryAfter     = "ngorm:query_after"
	HookQueryAfterFind = "ngorm:query_after_find"
	HookBeforeCreate   = "ngorm:before_create_hook"
	HookBeforeSave     = "ngorm:before_save_hook"
	Create             = "ngorm:create"
	BeforeCreate       = "ngorm:before_create"
	AfterCreate        = "ngorm:after_create"
	HookAfterCreate    = "ngorm:after_create"
	HookAfterSave      = "ngorm:after_save_hook"
	UpdateAttrs        = "ngorm:update_attrs"
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
	ID        uint `gorm:"primary_key"`
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
	m map[reflect.Type]*Struct
	l *sync.RWMutex
}

//Set stores value witht the given key.
func (s *SafeStructsMap) Set(key reflect.Type, value *Struct) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

//Get retrieves the value stored with the gived key.
func (s *SafeStructsMap) Get(key reflect.Type) *Struct {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

//NewStructsMap returns a safe map for storing *Struct objects.
func NewStructsMap() *SafeStructsMap {
	return &SafeStructsMap{l: new(sync.RWMutex), m: make(map[reflect.Type]*Struct)}
}

//Scope is the scope level of SQL building.
type Scope struct {
	Value           interface{}
	SQL             string
	SQLVars         []interface{}
	InstanceID      string
	PrimaryKeyField *Field
	SkipLeft        bool
	SelectAttrs     *[]string
	mu              sync.RWMutex
	data            map[string]interface{}
}

func (s *Scope) Set(key string, value interface{}) {
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()
}

func (s *Scope) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	v, ok := s.data[key]
	s.mu.RUnlock()
	return v, ok
}

//Search is the search level of SQL buidling
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
}

// Expr is SQL expression
type Expr struct {
	Q    string
	Args []interface{}
}
