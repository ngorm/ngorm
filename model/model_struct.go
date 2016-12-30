package model

import (
	"database/sql"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gernest/ngorm/util"
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

func GetForeignField(column string, fields []*StructField) *StructField {
	for _, field := range fields {
		if field.Name == column || field.DBName == column || field.DBName == util.ToDBName(column) {
			return field
		}
	}
	return nil
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

func ParseFieldStructForDialect(field *StructField) (fieldValue reflect.Value, sqlType string, size int, additionalType string) {
	// Get redirected field type
	var reflectType = field.Struct.Type
	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	// Get redirected field value
	fieldValue = reflect.Indirect(reflect.New(reflectType))

	// Get scanner's real value
	var getScannerValue func(reflect.Value)
	getScannerValue = func(value reflect.Value) {
		fieldValue = value
		if _, isScanner := reflect.New(fieldValue.Type()).Interface().(sql.Scanner); isScanner && fieldValue.Kind() == reflect.Struct {
			getScannerValue(fieldValue.Field(0))
		}
	}
	getScannerValue(fieldValue)

	// Default Size
	if num, ok := field.TagSettings["SIZE"]; ok {
		size, _ = strconv.Atoi(num)
	} else {
		size = 255
	}

	// Default type from tag setting
	additionalType = field.TagSettings["NOT NULL"] + " " + field.TagSettings["UNIQUE"]
	if value, ok := field.TagSettings["DEFAULT"]; ok {
		additionalType = additionalType + " DEFAULT " + value
	}

	return fieldValue, field.TagSettings["TYPE"], size, strings.TrimSpace(additionalType)
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
