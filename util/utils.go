// Package util contains utility functions that are used in dirrent places of
// the codebase.
package util

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/ngorm/ngorm/errmsg"
	"github.com/oxtoacart/bpool"
)

var B *bpool.BufferPool

func init() {
	B = bpool.NewBufferPool(1000)
}

// Copied from golint
var commonInitialisms = []string{"API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SSH", "TLS", "TTL", "UI", "UID", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XSRF", "XSS"}
var commonInitialismsReplacer *strings.Replacer

func init() {
	var commonInitialismsForReplacer []string
	for _, initialism := range commonInitialisms {
		commonInitialismsForReplacer = append(commonInitialismsForReplacer, initialism, strings.Title(strings.ToLower(initialism)))
	}
	commonInitialismsReplacer = strings.NewReplacer(commonInitialismsForReplacer...)
}

type safeMap struct {
	m map[string]string
	l *sync.RWMutex
}

func (s *safeMap) Set(key string, value string) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

func (s *safeMap) Get(key string) string {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

func newSafeMap() *safeMap {
	return &safeMap{l: new(sync.RWMutex), m: make(map[string]string)}
}

var smap = newSafeMap()

type strCase bool

const (
	lower strCase = false
	upper strCase = true
)

// ToDBName convert string to db name
func ToDBName(name string) string {
	if v := smap.Get(name); v != "" {
		return v
	}
	if name == "" {
		return ""
	}
	var (
		lastCase, currCase, nextCase strCase
	)
	value := commonInitialismsReplacer.Replace(name)
	buf := B.Get()
	defer func() {
		B.Put(buf)
	}()

	for i, v := range value[:len(value)-1] {
		nextCase = strCase(value[i+1] >= 'A' && value[i+1] <= 'Z')
		if i > 0 {
			if currCase == upper {
				if lastCase == upper && nextCase == upper {
					buf.WriteRune(v)
				} else {
					if value[i-1] != '_' && value[i+1] != '_' {
						buf.WriteRune('_')
					}
					buf.WriteRune(v)
				}
			} else {
				buf.WriteRune(v)
			}
		} else {
			currCase = upper
			buf.WriteRune(v)
		}
		lastCase = currCase
		currCase = nextCase
	}

	buf.WriteByte(value[len(value)-1])

	s := strings.ToLower(buf.String())
	smap.Set(name, s)
	return s
}

func indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	return reflectValue
}

func ToQueryMarks(primaryValues [][]interface{}) string {
	// var results []string
	results := make([]string, len(primaryValues))
	for i := 0; i < len(primaryValues); i++ {
		marks := make([]string, len(primaryValues[i]))
		for j := 0; j < len(primaryValues[i]); j++ {
			marks[j] = "?"
		}
		if len(marks) > 1 {
			results[i] = fmt.Sprintf("(%v)", strings.Join(marks, ","))
		} else {
			results[i] = strings.Join(marks, "")
		}
	}
	return strings.Join(results, ",")
}

//IsBlank returns true if the value represent a zero value of the specified value ype.
func IsBlank(value reflect.Value) bool {
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func EqualAsString(a interface{}, b interface{}) bool {
	return ToString(a) == ToString(b)
}

func ToString(str interface{}) string {
	if values, ok := str.([]interface{}); ok {
		results := make([]string, len(values))
		for i := 0; i < len(values); i++ {
			results[i] = ToString(values[i])
		}
		return strings.Join(results, "_")
	} else if bytes, ok := str.([]byte); ok {
		return string(bytes)
	} else if reflectValue := reflect.Indirect(reflect.ValueOf(str)); reflectValue.IsValid() {
		return fmt.Sprintf("%v", reflectValue.Interface())
	}
	return ""
}

func MakeSlice(elemType reflect.Type) interface{} {
	if elemType.Kind() == reflect.Slice {
		elemType = elemType.Elem()
	}
	sliceType := reflect.SliceOf(elemType)
	slice := reflect.New(sliceType)
	slice.Elem().Set(reflect.MakeSlice(sliceType, 0, 0))
	return slice.Interface()
}

// GetValueFromFields return given fields's value
func GetValueFromFields(value reflect.Value, fieldNames []string) (results []interface{}) {
	// If value is a nil pointer, Indirect returns a zero Value!
	// Therefor we need to check for a zero value,
	// as FieldByName could panic
	if indirectValue := reflect.Indirect(value); indirectValue.IsValid() {
		for _, fieldName := range fieldNames {
			if fieldValue := indirectValue.FieldByName(fieldName); fieldValue.IsValid() {
				result := fieldValue.Interface()
				if r, ok := result.(driver.Valuer); ok {
					result, _ = r.Value()
				}
				results = append(results, result)
			}
		}
	}
	return
}

//AddExtraSpaceIfExist adds an extra space  at the beginning of the string
//if the string is not empty.
func AddExtraSpaceIfExist(str string) string {
	if str != "" {
		return " " + str
	}
	return ""
}

//GetInterfaceAsSQL returns sql value representation of the value.
func GetInterfaceAsSQL(value interface{}) (string, error) {
	switch value.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", value), nil
	}

	return "", errmsg.ErrInvalidSQL
}

//ToSearchableMap transform attrs to a map.
func ToSearchableMap(attrs ...interface{}) (result interface{}) {
	if len(attrs) > 1 {
		if str, ok := attrs[0].(string); ok {
			result = map[string]interface{}{str: attrs[1]}
		}
	} else if len(attrs) == 1 {
		if attr, ok := attrs[0].(map[string]interface{}); ok {
			result = attr
		}

		if attr, ok := attrs[0].(interface{}); ok {
			result = attr
		}
	}
	return
}

// WrapTX returnstx intranstaction block
func WrapTX(tx string) string {
	buf := B.Get()
	defer func() {
		B.Put(buf)
	}()
	buf.WriteString("BEGIN TRANSACTION;\n")
	buf.WriteString("\t")
	buf.WriteString(tx)
	buf.WriteString(";\n")
	buf.WriteString("COMMIT;")
	return buf.String()
}

// ColumnAsArray returns an array of column values
func ColumnAsArray(columns []string, values ...interface{}) (results [][]interface{}) {
	var indirectValue reflect.Value
	for _, value := range values {
		indirectValue = reflect.ValueOf(value)
		if indirectValue.Kind() == reflect.Ptr {
			indirectValue = indirectValue.Elem()
		}

		switch indirectValue.Kind() {
		case reflect.Slice:
			for i := 0; i < indirectValue.Len(); i++ {
				var result []interface{}
				object := indirectValue.Index(i)
				if object.Kind() == reflect.Ptr {
					object = object.Elem()
				}
				var hasValue = false
				for _, column := range columns {
					field := object.FieldByName(column)
					if hasValue || !IsBlank(field) {
						hasValue = true
					}
					result = append(result, field.Interface())
				}

				if hasValue {
					results = append(results, result)
				}
			}
		case reflect.Struct:
			var result []interface{}
			var hasValue = false
			for _, column := range columns {
				field := indirectValue.FieldByName(column)
				if hasValue || !IsBlank(field) {
					hasValue = true
				}
				result = append(result, field.Interface())
			}

			if hasValue {
				results = append(results, result)
			}
		}
	}
	return
}

func ToQueryValues(values [][]interface{}) (results []interface{}) {
	for _, value := range values {
		for _, v := range value {
			results = append(results, v)
		}
	}
	return
}
