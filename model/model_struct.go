package model

import (
	"database/sql"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Model struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// ModelStruct model definition
type ModelStruct struct {
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

func (structField *StructField) Clone() *StructField {
	clone := &StructField{
		DBName:          structField.DBName,
		Name:            structField.Name,
		Names:           structField.Names,
		IsPrimaryKey:    structField.IsPrimaryKey,
		IsNormal:        structField.IsNormal,
		IsIgnored:       structField.IsIgnored,
		IsScanner:       structField.IsScanner,
		HasDefaultValue: structField.HasDefaultValue,
		Tag:             structField.Tag,
		TagSettings:     map[string]string{},
		Struct:          structField.Struct,
		IsForeignKey:    structField.IsForeignKey,
		Relationship:    structField.Relationship,
	}

	for key, value := range structField.TagSettings {
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

type SafeModelStructsMap struct {
	m map[reflect.Type]*ModelStruct
	l *sync.RWMutex
}

func (s *SafeModelStructsMap) Set(key reflect.Type, value *ModelStruct) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

func (s *SafeModelStructsMap) Get(key reflect.Type) *ModelStruct {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

func NewModelStructsMap() *SafeModelStructsMap {
	return &SafeModelStructsMap{l: new(sync.RWMutex), m: make(map[reflect.Type]*ModelStruct)}
}

type Scope struct {
	Value           interface{}
	SQL             string
	SQLVars         []interface{}
	InstanceID      string
	PrimaryKeyField *Field
	SkipLeft        bool
	Fields          *[]*Field
	SelectAttrs     *[]string
}

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

type SearchPreload struct {
	Schema     string
	Conditions []interface{}
}

type SQLCommon interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Begin() (*sql.Tx, error)
}

// SQL expression
type Expr struct {
	Q    string
	Args []interface{}
}
