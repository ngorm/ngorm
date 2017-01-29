// Package scope defines functions that operates on  engine.Engine and  enables
// operating on model values easily.
//
// Scope adds a layer of encapsulation on the model on which we are using to
// compose Queries or interact with the database
package scope

import (
	"database/sql"
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"strings"
	"time"

	"github.com/jinzhu/inflection"
	"github.com/ngorm/ngorm/engine"
	"github.com/ngorm/ngorm/errmsg"
	"github.com/ngorm/ngorm/model"
	"github.com/ngorm/ngorm/regexes"
	"github.com/ngorm/ngorm/util"
)

//Quote quotes the str into an SQL string. This makes sure sql strings have ""
//around them.
//
// For the case of a str which has a dot in it example one.two the string is
// quoted and becomes "one"."two" and the quote implementation is called from
// the e.Parent.Dialect.
//
// In case of a string without a dot example one it will be quoted using the
// current dialect e.Dialect
//
func Quote(e *engine.Engine, str string) string {
	if strings.Index(str, ".") != -1 {
		newStrs := []string{}
		for _, s := range strings.Split(str, ".") {
			newStrs = append(newStrs, e.Dialect.Quote(s))
		}
		return strings.Join(newStrs, ".")
	}
	return e.Dialect.Quote(str)
}

//Fields extracts []*model.Fields from value, value is obvously a struct or
//something. This is only done when e.Scope.Fields is nil, for the case of non
//nil value then *e.Scope.Fiedls is returned without computing anything.
func Fields(e *engine.Engine, value interface{}) ([]*model.Field, error) {
	var fields []*model.Field
	i := reflect.ValueOf(value)
	if i.Kind() == reflect.Ptr {
		i = i.Elem()
	}
	isStruct := i.Kind() == reflect.Struct
	m, err := GetModelStruct(e, value)
	if err != nil {
		return nil, err
	}
	for _, structField := range m.StructFields {
		if isStruct {
			fieldValue := i
			for _, name := range structField.Names {
				fieldValue = reflect.Indirect(fieldValue).FieldByName(name)
			}
			fields = append(fields, &model.Field{
				StructField: structField,
				Field:       fieldValue,
				IsBlank:     util.IsBlank(fieldValue)})
		} else {
			fields = append(fields, &model.Field{
				StructField: structField,
				IsBlank:     true})
		}
	}
	return fields, nil
}

//GetModelStruct construct a *model.Struct from value. This does not set
//the e.Scope.Value to value, you must set this value manually if you want to
//set the scope value.
//
// value must be a go struct or a slict of go struct. The computed *model.Struct is cached , so
// multiple calls to this function with the same value won't compute anything
// and return the cached copy. It is less unlikely that the structs will be
// changine at runtime.
//
// The value can implement engine.Tabler interface to help easily identify the
// table name for the model.
func GetModelStruct(e *engine.Engine, value interface{}) (*model.Struct, error) {
	var m model.Struct
	// Scope value can't be nil
	if value == nil {
		return nil, errors.New("nil value")
	}

	refType := reflect.ValueOf(value).Type()
	if refType.Kind() == reflect.Ptr {
		refType = refType.Elem()
	}
	if refType.Kind() == reflect.Slice {
		refType = refType.Elem()
		if refType.Kind() == reflect.Ptr {
			refType = refType.Elem()
		}
	}

	// Scope value need to be a struct
	if refType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%s is not supported, value should be a struct ", refType.Kind())
	}

	// Get Cached model struct
	if v := e.StructMap.Get(refType); v != nil {
		return v, nil
	}

	m.ModelType = refType

	// Set default table name
	if tabler, ok := reflect.New(refType).Interface().(engine.Tabler); ok {
		m.DefaultTableName = tabler.TableName()
	} else {
		tableName := util.ToDBName(refType.Name())

		// In case we have set SingulaTable to false, then we pluralize the
		// table name. For example session becomes sessions.
		if !e.SingularTable {
			tableName = inflection.Plural(tableName)
		}
		m.DefaultTableName = tableName
	}

	// Get all fields
	for i := 0; i < refType.NumField(); i++ {
		if fStruct := refType.Field(i); ast.IsExported(fStruct.Name) {
			field := &model.StructField{
				Struct:      fStruct,
				Name:        fStruct.Name,
				Names:       []string{fStruct.Name},
				Tag:         fStruct.Tag,
				TagSettings: model.ParseTagSetting(fStruct.Tag),
			}

			// is ignored field
			if _, ok := field.TagSettings["-"]; ok {
				field.IsIgnored = true
			} else {
				if _, ok := field.TagSettings["PRIMARY_KEY"]; ok {
					field.IsPrimaryKey = true
					m.PrimaryFields = append(m.PrimaryFields, field)
				}

				if _, ok := field.TagSettings["DEFAULT"]; ok {
					field.HasDefaultValue = true
				}

				if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok && !field.IsPrimaryKey {
					field.HasDefaultValue = true
				}

				inType := fStruct.Type
				for inType.Kind() == reflect.Ptr {
					inType = inType.Elem()
				}

				fieldValue := reflect.New(inType).Interface()
				if _, isScanner := fieldValue.(sql.Scanner); isScanner {
					// is scanner
					field.IsScanner, field.IsNormal = true, true
					if inType.Kind() == reflect.Struct {
						for i := 0; i < inType.NumField(); i++ {
							for key, value := range model.ParseTagSetting(inType.Field(i).Tag) {
								field.TagSettings[key] = value
							}
						}
					}
				} else if _, isTime := fieldValue.(*time.Time); isTime {
					// is time
					field.IsNormal = true
				} else if _, ok := field.TagSettings["EMBEDDED"]; ok || fStruct.Anonymous {
					// is embedded struct
					ms, err := GetModelStruct(e, fieldValue)
					if err != nil {
						return nil, err
					}
					for _, subField := range ms.StructFields {
						subField = subField.Clone()
						subField.Names = append([]string{fStruct.Name}, subField.Names...)
						if prefix, ok := field.TagSettings["EMBEDDED_PREFIX"]; ok {
							subField.DBName = prefix + subField.DBName
						}
						if subField.IsPrimaryKey {
							m.PrimaryFields = append(m.PrimaryFields, subField)
						}
						m.StructFields = append(m.StructFields, subField)
					}
					continue
				} else {
					// build relationships
					switch inType.Kind() {
					case reflect.Slice:
						defer func() {
							_ = buildRelationSlice(e, value, refType, &m, field)
						}()

					case reflect.Struct:
						defer func() {
							_ = buildRelationStruct(e, value, refType, &m, field)
						}()
					default:
						field.IsNormal = true
					}
				}
			}

			// Even it is ignored, also possible to decode db value into the field
			if value, ok := field.TagSettings["COLUMN"]; ok {
				field.DBName = value
			} else {
				field.DBName = util.ToDBName(fStruct.Name)
			}

			m.StructFields = append(m.StructFields, field)
		}
	}

	if len(m.PrimaryFields) == 0 {
		if field := GetForeignField("id", m.StructFields); field != nil {
			field.IsPrimaryKey = true
			m.PrimaryFields = append(m.PrimaryFields, field)
		}
	}

	e.StructMap.Set(refType, &m)
	return &m, nil
}

//BuildRelationSlice builds relationship for a field of kind reflect.Slice. This
//updates the ModelStruct m accordingly.
//
//TODO: (gernest) Proper error handling.Make sure we return error, this is a lot
//of loggic and no any error should be absorbed.
func buildRelationSlice(e *engine.Engine, modelValue interface{}, refType reflect.Type, m *model.Struct, field *model.StructField) error {
	var (
		rel                    = &model.Relationship{}
		toScope                = reflect.New(field.Struct.Type).Interface()
		fks                    []string
		associationForeignKeys []string
		elemType               = field.Struct.Type
	)

	if fk := field.TagSettings["FOREIGNKEY"]; fk != "" {
		fks = strings.Split(field.TagSettings["FOREIGNKEY"], ",")
	}

	if fk := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; fk != "" {
		associationForeignKeys = strings.Split(field.TagSettings["ASSOCIATIONFOREIGNKEY"], ",")
	}

	for elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() == reflect.Struct {
		if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
			rel.Kind = "many_to_many"

			// if no foreign keys defined with tag
			if len(fks) == 0 {
				for _, field := range m.PrimaryFields {
					fks = append(fks, field.DBName)
				}
			}

			for _, fk := range fks {
				if foreignField := GetForeignField(fk, m.StructFields); foreignField != nil {
					// source foreign keys (db names)
					rel.ForeignFieldNames = append(rel.ForeignFieldNames, foreignField.DBName)
					// join table foreign keys for source
					joinTableDBName := util.ToDBName(refType.Name()) + "_" + foreignField.DBName
					rel.ForeignDBNames = append(rel.ForeignDBNames, joinTableDBName)
				}
			}

			// if no association foreign keys defined with tag
			if len(associationForeignKeys) == 0 {
				pf, err := PrimaryFields(e, toScope)
				if err != nil {
					return err
				}
				for _, field := range pf {
					associationForeignKeys = append(associationForeignKeys, field.DBName)
				}
			}

			for _, name := range associationForeignKeys {
				field, err := FieldByName(e, toScope, name)
				if err != nil {
					return err
				}
				// association foreign keys (db names)
				rel.AssociationForeignFieldNames = append(rel.AssociationForeignFieldNames, field.DBName)
				// join table foreign keys for association
				joinTableDBName := util.ToDBName(elemType.Name()) + "_" + field.DBName
				rel.AssociationForeignDBNames = append(rel.AssociationForeignDBNames, joinTableDBName)
			}

			joinTableHandler := &model.JoinTableHandler{}
			SetupJoinTable(joinTableHandler, rel, many2many, refType, elemType)
			rel.JoinTableHandler = joinTableHandler
			field.Relationship = rel
		} else {
			// User has many comments, associationType is User, comment use UserID as foreign key
			var associationType = refType.Name()
			ms, err := GetModelStruct(e, toScope)
			if err != nil {
				return err
			}
			var toFields = ms.StructFields
			rel.Kind = "has_many"

			if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
				// Dog has many toys, tag polymorphic is Owner, then associationType is Owner
				// Toy use OwnerID, OwnerType ('dogs') as foreign key
				if polymorphicType := GetForeignField(polymorphic+"Type", toFields); polymorphicType != nil {
					associationType = polymorphic
					rel.PolymorphicType = polymorphicType.Name
					rel.PolymorphicDBName = polymorphicType.DBName
					// if Dog has multiple set of toys set name of the set (instead of default 'dogs')
					if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
						rel.PolymorphicValue = value
					} else {
						rel.PolymorphicValue = e.Search.TableName
					}
					polymorphicType.IsForeignKey = true
				}
			}

			// if no foreign keys defined with tag
			if len(fks) == 0 {
				// if no association foreign keys defined with tag
				if len(associationForeignKeys) == 0 {
					for _, field := range m.PrimaryFields {
						fks = append(fks, associationType+field.Name)
						associationForeignKeys = append(associationForeignKeys, field.Name)
					}
				} else {
					// generate foreign keys from defined association foreign keys
					for _, scopeFieldName := range associationForeignKeys {
						if foreignField := GetForeignField(scopeFieldName, m.StructFields); foreignField != nil {
							fks = append(fks, associationType+foreignField.Name)
							associationForeignKeys = append(associationForeignKeys, foreignField.Name)
						}
					}
				}
			} else {
				// generate association foreign keys from foreign keys
				if len(associationForeignKeys) == 0 {
					for _, fk := range fks {
						if strings.HasPrefix(fk, associationType) {
							associationForeignKey := strings.TrimPrefix(fk, associationType)
							if foreignField := GetForeignField(associationForeignKey, m.StructFields); foreignField != nil {
								associationForeignKeys = append(associationForeignKeys, associationForeignKey)
							}
						}
					}
					if len(associationForeignKeys) == 0 && len(fks) == 1 {
						pk, err := PrimaryKey(e, modelValue)
						if err != nil {
							return err
						}
						associationForeignKeys = []string{pk}
					}
				} else if len(fks) != len(associationForeignKeys) {
					return errors.New("invalid foreign keys, should have same length")
				}
			}

			for idx, fk := range fks {
				if foreignField := GetForeignField(fk, toFields); foreignField != nil {
					if associationField := GetForeignField(associationForeignKeys[idx], m.StructFields); associationField != nil {
						// source foreign keys
						foreignField.IsForeignKey = true
						rel.AssociationForeignFieldNames = append(rel.AssociationForeignFieldNames, associationField.Name)
						rel.AssociationForeignDBNames = append(rel.AssociationForeignDBNames, associationField.DBName)

						// association foreign keys
						rel.ForeignFieldNames = append(rel.ForeignFieldNames, foreignField.Name)
						rel.ForeignDBNames = append(rel.ForeignDBNames, foreignField.DBName)
					}
				}
			}

			if len(rel.ForeignFieldNames) != 0 {
				field.Relationship = rel
			}
		}
	} else {
		field.IsNormal = true
	}
	return nil
}

//BuildRelationStruct builds relationship for a field of kind reflect.Struct . This
//updates the ModelStruct m accordingly.
//
//TODO: (gernest) Proper error handling.Make sure we return error, this is a lot
//of loggic and no any error should be absorbed.
func buildRelationStruct(e *engine.Engine, modelValue interface{}, refType reflect.Type, m *model.Struct, field *model.StructField) error {
	var (
		// user has one profile, associationType is User, profile use UserID as foreign key
		// user belongs to profile, associationType is Profile, user use ProfileID as foreign key
		associationType           = refType.Name()
		rel                       = &model.Relationship{}
		toScope                   = reflect.New(field.Struct.Type).Interface()
		tagForeignKeys            []string
		tagAssociationForeignKeys []string
	)
	ms, err := GetModelStruct(e, toScope)
	if err != nil {
		return err
	}
	toFields := ms.StructFields

	if fk := field.TagSettings["FOREIGNKEY"]; fk != "" {
		tagForeignKeys = strings.Split(field.TagSettings["FOREIGNKEY"], ",")
	}

	if fk := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; fk != "" {
		tagAssociationForeignKeys = strings.Split(field.TagSettings["ASSOCIATIONFOREIGNKEY"], ",")
	}

	if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
		// Cat has one toy, tag polymorphic is Owner, then associationType is Owner
		// Toy use OwnerID, OwnerType ('cats') as foreign key
		if polymorphicType := GetForeignField(polymorphic+"Type", toFields); polymorphicType != nil {
			associationType = polymorphic
			rel.PolymorphicType = polymorphicType.Name
			rel.PolymorphicDBName = polymorphicType.DBName
			// if Cat has several different types of toys set name for each (instead of default 'cats')
			if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
				rel.PolymorphicValue = value
			} else {
				rel.PolymorphicValue = TableName(e, modelValue)
			}
			polymorphicType.IsForeignKey = true
		}
	}

	// Has One
	{
		var fks = tagForeignKeys
		var associationForeignKeys = tagAssociationForeignKeys
		// if no foreign keys defined with tag
		if len(fks) == 0 {
			// if no association foreign keys defined with tag
			if len(associationForeignKeys) == 0 {
				for _, primaryField := range m.PrimaryFields {
					fks = append(fks, associationType+primaryField.Name)
					associationForeignKeys = append(associationForeignKeys, primaryField.Name)
				}
			} else {
				// generate foreign keys form association foreign keys
				for _, associationForeignKey := range tagAssociationForeignKeys {
					if foreignField := GetForeignField(associationForeignKey, m.StructFields); foreignField != nil {
						fks = append(fks, associationType+foreignField.Name)
						associationForeignKeys = append(associationForeignKeys, foreignField.Name)
					}
				}
			}
		} else {
			// generate association foreign keys from foreign keys
			if len(associationForeignKeys) == 0 {
				for _, fk := range fks {
					if strings.HasPrefix(fk, associationType) {
						associationForeignKey := strings.TrimPrefix(fk, associationType)
						if foreignField := GetForeignField(associationForeignKey, m.StructFields); foreignField != nil {
							associationForeignKeys = append(associationForeignKeys, associationForeignKey)
						}
					}
				}
				if len(associationForeignKeys) == 0 && len(fks) == 1 {
					pk, err := PrimaryKey(e, modelValue)
					if err != nil {
						return err
					}
					associationForeignKeys = []string{pk}
				}
			} else if len(fks) != len(associationForeignKeys) {
				return errors.New("invalid foreign keys, should have same length")
			}
		}

		for idx, fk := range fks {
			if foreignField := GetForeignField(fk, toFields); foreignField != nil {
				if scopeField := GetForeignField(associationForeignKeys[idx], m.StructFields); scopeField != nil {
					foreignField.IsForeignKey = true
					// source foreign keys
					rel.AssociationForeignFieldNames = append(rel.AssociationForeignFieldNames, scopeField.Name)
					rel.AssociationForeignDBNames = append(rel.AssociationForeignDBNames, scopeField.DBName)

					// association foreign keys
					rel.ForeignFieldNames = append(rel.ForeignFieldNames, foreignField.Name)
					rel.ForeignDBNames = append(rel.ForeignDBNames, foreignField.DBName)
				}
			}
		}
	}

	if len(rel.ForeignFieldNames) != 0 {
		rel.Kind = "has_one"
		field.Relationship = rel
	} else {
		var fks = tagForeignKeys
		var associationForeignKeys = tagAssociationForeignKeys

		if len(fks) == 0 {
			// generate foreign keys & association foreign keys
			if len(associationForeignKeys) == 0 {
				pf, err := PrimaryFields(e, toScope)
				if err != nil {
					return err
				}
				for _, primaryField := range pf {
					fks = append(fks, field.Name+primaryField.Name)
					associationForeignKeys = append(associationForeignKeys, primaryField.Name)
				}
			} else {
				// generate foreign keys with association foreign keys
				for _, associationForeignKey := range associationForeignKeys {
					if foreignField := GetForeignField(associationForeignKey, toFields); foreignField != nil {
						fks = append(fks, field.Name+foreignField.Name)
						associationForeignKeys = append(associationForeignKeys, foreignField.Name)
					}
				}
			}
		} else {
			// generate foreign keys & association foreign keys
			if len(associationForeignKeys) == 0 {
				for _, fk := range fks {
					if strings.HasPrefix(fk, field.Name) {
						associationForeignKey := strings.TrimPrefix(fk, field.Name)
						if foreignField := GetForeignField(associationForeignKey, toFields); foreignField != nil {
							associationForeignKeys = append(associationForeignKeys, associationForeignKey)
						}
					}
				}
				if len(associationForeignKeys) == 0 && len(fks) == 1 {
					pk, err := PrimaryKey(e, toScope)
					if err != nil {
						return err
					}
					associationForeignKeys = []string{pk}
				}
			} else if len(fks) != len(associationForeignKeys) {
				return errors.New("invalid foreign keys, should have same length")
			}
		}

		for idx, fk := range fks {
			if foreignField := GetForeignField(fk, m.StructFields); foreignField != nil {
				if associationField := GetForeignField(associationForeignKeys[idx], toFields); associationField != nil {
					foreignField.IsForeignKey = true

					// association foreign keys
					rel.AssociationForeignFieldNames = append(rel.AssociationForeignFieldNames, associationField.Name)
					rel.AssociationForeignDBNames = append(rel.AssociationForeignDBNames, associationField.DBName)

					// source foreign keys
					rel.ForeignFieldNames = append(rel.ForeignFieldNames, foreignField.Name)
					rel.ForeignDBNames = append(rel.ForeignDBNames, foreignField.DBName)
				}
			}
		}

		if len(rel.ForeignFieldNames) != 0 {
			rel.Kind = "belongs_to"
			field.Relationship = rel
		}
	}
	return nil
}

//FieldByName returns the field in the model struct value with name name.
//
//TODO:(gernest) return an error when the field is not found.
func FieldByName(e *engine.Engine, value interface{}, name string) (*model.Field, error) {
	dbName := util.ToDBName(name)
	fds, err := Fields(e, value)
	if err != nil {
		return nil, err
	}
	for _, field := range fds {
		if field.Name == name || field.DBName == name {
			return field, nil
		}
		if field.DBName == dbName {
			return field, nil
		}
	}
	return nil, errors.New("field not found")
}

//PrimaryFields returns fields that have PRIMARY_KEY tag from the struct value.
func PrimaryFields(e *engine.Engine, value interface{}) ([]*model.Field, error) {
	var fields []*model.Field
	fds, err := Fields(e, value)
	if err != nil {
		return nil, err
	}
	for _, field := range fds {
		if field.IsPrimaryKey {
			fields = append(fields, field)
		}
	}
	return fields, nil
}

//PrimaryField returns the field with name id, or any primary field that happens
//to be the one defined by the model value.
func PrimaryField(e *engine.Engine, value interface{}) (*model.Field, error) {
	m, err := GetModelStruct(e, value)
	if err != nil {
		return nil, err
	}
	if primaryFields := m.PrimaryFields; len(primaryFields) > 0 {
		if len(primaryFields) > 1 {
			field, err := FieldByName(e, value, "id")
			if err != nil {
				return nil, err
			}
			return field, nil
		}
		pf, err := PrimaryFields(e, value)
		if err != nil {
			return nil, err
		}
		return pf[0], nil
	}
	return nil, errors.New("no field found")
}

// TableName returns a string representation of the possible name of the table
// that is mapped to the model value.
//
// If it happens that the model value implements engine.Tabler interface then we
// go with it.
//
// In case we are in search mode, the Tablename inside the e.Search.TableName is
// what we use.
func TableName(e *engine.Engine, value interface{}) string {
	if e.Search != nil && len(e.Search.TableName) > 0 {
		return e.Search.TableName
	}

	if tabler, ok := value.(engine.Tabler); ok {
		return tabler.TableName()
	}

	if tabler, ok := value.(engine.DBTabler); ok {
		return tabler.TableName(e)
	}
	ms, err := GetModelStruct(e, value)
	if err != nil {
		//TODO log this?
		return ""
	}
	return ms.DefaultTableName
}

//PrimaryKey returns the name of the primary key for the model value
func PrimaryKey(e *engine.Engine, value interface{}) (string, error) {
	pf, err := PrimaryField(e, value)
	if err != nil {
		return "", err
	}
	return pf.DBName, nil
}

//QuotedTableName  returns a quoted table name.
func QuotedTableName(e *engine.Engine, value interface{}) string {
	if e.Search != nil && len(e.Search.TableName) > 0 {
		if strings.Index(e.Search.TableName, " ") != -1 {
			return e.Search.TableName
		}
		return Quote(e, e.Search.TableName)
	}

	return Quote(e, TableName(e, value))
}

//AddToVars add value to e.Scope.SQLVars it returns  the positional binding of
//the values.
//
// The way positional arguments are handled inthe database/sql package relies on
// database specific setting.
//
// For instance in ql
//    $1 will bind the value of the first argument.
//
// The returned string depends on implementation provided by the
// Dialect.BindVar, the number that is passed to BindVar is based on the number
// of items stored in e.Scope.SQLVars. So if the length is 4 it might be $4 for
// the ql dialect.
//
// It is possible to supply *model.Expr as value. The expression will be
// evaluated accordingly by replacing each occurrence of ? in *model.Expr.Q with
// the positional binding of the *model.Expr.Arg item.
func AddToVars(e *engine.Engine, value interface{}) string {
	if expr, ok := value.(*model.Expr); ok {
		exp := expr.Q
		for _, arg := range expr.Args {
			exp = strings.Replace(exp, "?", AddToVars(e, arg), 1)
		}
		return exp
	}

	e.Scope.SQLVars = append(e.Scope.SQLVars, value)
	return e.Dialect.BindVar(len(e.Scope.SQLVars))
}

//HasColumn returns true if the modelValue has column of name column.
func HasColumn(e *engine.Engine, modelValue interface{}, column string) bool {
	ms, err := GetModelStruct(e, modelValue)
	if err != nil {
		//TODO log this?
		return false
	}
	for _, field := range ms.StructFields {
		if field.IsNormal && (field.Name == column || field.DBName == column) {
			return true
		}
	}
	return false
}

//GetForeignField return the foreign field among the supplied fields.
func GetForeignField(column string, fields []*model.StructField) *model.StructField {
	for _, field := range fields {
		if field.Name == column || field.DBName == column || field.DBName == util.ToDBName(column) {
			return field
		}
	}
	return nil
}

//Scan scans restult from the rows into fields.
func Scan(rows *sql.Rows, columns []string, fields []*model.Field) {
	var (
		ignored            interface{}
		values             = make([]interface{}, len(columns))
		selectFields       []*model.Field
		selectedColumnsMap = map[string]int{}
		resetFields        = map[int]*model.Field{}
	)

	for index, column := range columns {
		values[index] = &ignored

		selectFields = fields
		if idx, ok := selectedColumnsMap[column]; ok {
			selectFields = selectFields[idx+1:]
		}

		for fieldIndex, field := range selectFields {
			if field.DBName == column {
				if field.Field.Kind() == reflect.Ptr {
					values[index] = field.Field.Addr().Interface()
				} else {
					reflectValue := reflect.New(reflect.PtrTo(field.Struct.Type))
					reflectValue.Elem().Set(field.Field.Addr())
					values[index] = reflectValue.Interface()
					resetFields[index] = field
				}

				selectedColumnsMap[column] = fieldIndex

				if field.IsNormal {
					break
				}
			}
		}
	}
	err := rows.Scan(values...)
	if err != nil {
		fmt.Println(err)
	}

	for index, field := range resetFields {
		if v := reflect.ValueOf(values[index]).Elem().Elem(); v.IsValid() {
			field.Field.Set(v)
		}
	}
}

//SetColumn sets the column value.
func SetColumn(e *engine.Engine, column interface{}, value interface{}) error {
	var updateAttrs = map[string]interface{}{}
	if attrs, ok := e.Scope.Get(model.UpdateAttrs); ok {
		if u, ok := attrs.(map[string]interface{}); ok {
			updateAttrs = u
			defer e.Scope.Set(model.UpdateAttrs, updateAttrs)
		}
	}

	if field, ok := column.(*model.Field); ok {
		updateAttrs[field.DBName] = value
		return field.Set(value)
	} else if name, ok := column.(string); ok {
		var (
			dbName           = util.ToDBName(name)
			mostMatchedField *model.Field
		)
		fds, err := Fields(e, e.Scope.Value)
		if err != nil {
			return err
		}
		for _, field := range fds {
			if field.DBName == value {
				updateAttrs[field.DBName] = value
				return field.Set(value)
			}
			if (field.DBName == dbName) || (field.Name == name && mostMatchedField == nil) {
				mostMatchedField = field
			}
		}

		if mostMatchedField != nil {
			updateAttrs[mostMatchedField.DBName] = value
			return mostMatchedField.Set(value)
		}
	}
	return errors.New("could not convert column to field")
}

//SelectAttrs returns the attributes in the select query.
func SelectAttrs(e *engine.Engine) []string {
	if e.Scope.SelectAttrs == nil {
		attrs := []string{}
		for _, value := range e.Search.Selects {
			if str, ok := value.(string); ok {
				attrs = append(attrs, str)
			} else if strs, ok := value.([]string); ok {
				attrs = append(attrs, strs...)
			} else if strs, ok := value.([]interface{}); ok {
				for _, str := range strs {
					attrs = append(attrs, fmt.Sprintf("%v", str))
				}
			}
		}
		e.Scope.SelectAttrs = &attrs
	}
	return *e.Scope.SelectAttrs
}

//ChangeableField returns true if the field's value can be changed.
func ChangeableField(e *engine.Engine, field *model.Field) bool {
	if selectAttrs := SelectAttrs(e); len(selectAttrs) > 0 {
		for _, attr := range selectAttrs {
			if field.Name == attr || field.DBName == attr {
				return true
			}
		}
		return false
	}

	for _, attr := range e.Search.Omits {
		if field.Name == attr || field.DBName == attr {
			return false
		}
	}

	return true
}

//CreateTable generates CREATE TABLE SQL
func CreateTable(e *engine.Engine, value interface{}) error {
	var tags []string
	var primaryKeys []string
	var primaryKeyInColumnType = false
	m, err := GetModelStruct(e, value)
	if err != nil {
		return err
	}

	for _, field := range m.StructFields {
		if field.IsNormal {
			sqlTag, err := e.Dialect.DataTypeOf(field)
			if err != nil {

				return err
			}

			// Check if the primary key constraint was specified as
			// part of the column type. If so, we can only support
			// one column as the primary key.
			if strings.Contains(strings.ToLower(sqlTag), "primary key") {
				primaryKeyInColumnType = true
			}

			tags = append(tags, Quote(e, field.DBName)+" "+sqlTag)
		}

		if field.IsPrimaryKey {
			primaryKeys = append(primaryKeys, Quote(e, field.DBName))
		}
		err = CreateJoinTable(e, field)
		if err != nil {
			e.Log.Info(err.Error() + field.Name)
			return err
		}
	}

	var primaryKeyStr string
	if len(primaryKeys) > 0 && !primaryKeyInColumnType {
		primaryKeyStr = e.Dialect.PrimaryKey(primaryKeys)
		if primaryKeyStr != "" {
			primaryKeyStr = ", " + primaryKeyStr
		}
	}
	var options string
	opts, ok := e.Scope.Get(model.TableOptions)
	if ok {
		options = opts.(string)
	}
	e.Scope.SQL = fmt.Sprintf("CREATE TABLE %v (%v %v) %s",
		QuotedTableName(e, value), strings.Join(tags, ","),
		primaryKeyStr, options)
	return AutoIndex(e, value)
}

//CreateJoinTable creates a join table that handles many to many relationship.
//
//For instance if users have many to many relation to languages then the join
//table will be users_language and containing keys that point to both users and
//languages table.
func CreateJoinTable(e *engine.Engine, field *model.StructField) error {
	if rel := field.Relationship; rel != nil && rel.JoinTableHandler != nil {
		j := rel.JoinTableHandler
		if e.Dialect.HasTable(j.TableName) {
			return nil
		}
		value := reflect.New(field.Struct.Type).Interface()
		var sqlTypes, primaryKeys []string
		for idx, fieldName := range rel.ForeignFieldNames {
			f, err := FieldByName(e, value, fieldName)
			if err != nil {
				return err
			}
			fk := f.Clone()
			fk.IsPrimaryKey = false
			fk.TagSettings["IS_JOINTABLE_FOREIGNKEY"] = "true"
			delete(fk.TagSettings, "AUTO_INCREMENT")
			data, err := e.Dialect.DataTypeOf(fk)
			if err != nil {
				return err
			}
			sqlTypes = append(sqlTypes,
				Quote(e, rel.ForeignDBNames[idx])+" "+data)
			primaryKeys = append(primaryKeys, Quote(e, rel.ForeignDBNames[idx]))

		}

		for idx, fieldName := range rel.AssociationForeignFieldNames {
			field, err := FieldByName(e, value, fieldName)
			if err != nil {
				return err
			}
			fk := field.Clone()
			fk.IsPrimaryKey = false
			fk.TagSettings["IS_JOINTABLE_FOREIGNKEY"] = "true"
			delete(fk.TagSettings, "AUTO_INCREMENT")
			data, err := e.Dialect.DataTypeOf(fk)
			if err != nil {
				return err
			}
			sqlTypes = append(sqlTypes,
				Quote(e, rel.AssociationForeignDBNames[idx])+" "+data)
			primaryKeys = append(primaryKeys, Quote(e, rel.AssociationForeignDBNames[idx]))
		}
		var primaryKeyStr string
		if len(primaryKeys) > 0 {
			primaryKeyStr = e.Dialect.PrimaryKey(primaryKeys)
			if primaryKeyStr != "" {
				primaryKeyStr = ", " + primaryKeyStr
			}
		}
		if primaryKeyStr != "" {
			primaryKeyStr = ", PRIMARY KEY (" + primaryKeyStr + ")"
		}
		var tableOpts string
		opts, ok := e.Scope.Get(model.TableOptions)
		if ok {
			tableOpts = opts.(string)
		}

		sql := fmt.Sprintf("CREATE TABLE %v (%v %v) %s",
			Quote(e, j.TableName),
			strings.Join(sqlTypes, ","),
			primaryKeyStr, tableOpts)
		if !e.Scope.MultiExpr {
			e.Scope.MultiExpr = true
		}
		e.Scope.Exprs = append(e.Scope.Exprs, &model.Expr{Q: sql})
	}
	return nil
}

//AutoIndex generates CREATE INDEX SQL
func AutoIndex(e *engine.Engine, value interface{}) error {
	var indexes = map[string][]string{}
	var uniqueIndexes = map[string][]string{}
	m, err := GetModelStruct(e, value)
	if err != nil {
		return err
	}

	for _, field := range m.StructFields {
		if name, ok := field.TagSettings["INDEX"]; ok {
			names := strings.Split(name, ",")

			for _, name := range names {
				if name == "INDEX" || name == "" {
					name = fmt.Sprintf("idx_%v_%v", TableName(e, value), field.DBName)
				}
				indexes[name] = append(indexes[name], field.DBName)
			}
		}

		if name, ok := field.TagSettings["UNIQUE_INDEX"]; ok {
			names := strings.Split(name, ",")

			for _, name := range names {
				if name == "UNIQUE_INDEX" || name == "" {
					name = fmt.Sprintf("uix_%v_%v", TableName(e, value), field.DBName)
				}
				uniqueIndexes[name] = append(uniqueIndexes[name], field.DBName)
			}
		}
	}

	for name, columns := range indexes {
		err = AddIndex(e, false, value, name, columns...)
		if err != nil {
			return err
		}
	}

	for name, columns := range uniqueIndexes {
		err = AddIndex(e, true, value, name, columns...)
		if err != nil {
			return err
		}
	}

	return nil
}

//AddIndex add extra queries fo creating database index. The indexes are packed
//on e.Sope.Exprs and it sets the e.Scope.MultiExpr to true signaling that there
//are additional multiple SQL queries bundled in the e.Scope.
//
// if unique is true this will generate CREATE UNIQUE INDEX and in case of false
// it generates CREATE INDEX.
func AddIndex(e *engine.Engine, unique bool, value interface{}, indexName string, column ...string) error {
	if e.Dialect.HasIndex(TableName(e, value), indexName) {
		return nil
	}
	var columns []string
	for _, name := range column {
		if regexes.Column.MatchString(name) {
			name = Quote(e, name)
		}
		columns = append(columns, name)
	}

	sqlCreate := "CREATE INDEX"
	if unique {
		sqlCreate = "CREATE UNIQUE INDEX"
	}
	if !e.Scope.MultiExpr {
		e.Scope.MultiExpr = true
	}

	//NOTE: I removed whereSQl on the create index.
	sql := fmt.Sprintf("%s %v ON %v(%v)", sqlCreate,
		indexName, QuotedTableName(e, value), strings.Join(columns, ", "))
	e.Scope.Exprs = append(e.Scope.Exprs, &model.Expr{Q: sql})
	return nil
}

//DropTable generates SQL query for DROP TABLE.
//
// The Generated SQL is not wrapped in a transaction. All state altering queries
// must be wrapped in a transaction block.
//
// We don't need to rap the transaction block at this level so as to enable
// flexibility of combining multiple querries that will be wrapped under the
// same transaction.
func DropTable(e *engine.Engine, value interface{}) error {
	e.Scope.SQL = fmt.Sprintf("DROP TABLE %v", QuotedTableName(e, value))
	return nil
}

//Automigrate generates  sql for creting database table for model value if the
//table doesnt exist yet. It also alters fields if the model has been updated.
//
// NOTE For the case of an updated model which will need to alter the table to
// reflect the new changes, the SQL is stored under e.Scope.Exprs. The caller
// must be aware of this, and remember to chceck if e.Scope.MultiExpr is true so
// as to get the additional SQL.
func Automigrate(e *engine.Engine, value interface{}) error {
	tableName := TableName(e, value)
	quotedTableName := QuotedTableName(e, value)
	if !e.Dialect.HasTable(tableName) {
		return CreateTable(e, value)
	}
	m, err := GetModelStruct(e, value)
	if err != nil {
		return err
	}
	for _, field := range m.StructFields {
		if !e.Dialect.HasColumn(tableName, field.DBName) {
			if field.IsNormal {
				sqlTag, err := e.Dialect.DataTypeOf(field)
				if err != nil {
					return err
				}
				if !e.Scope.MultiExpr {
					e.Scope.MultiExpr = true
				}
				expr := &model.Expr{
					Q: fmt.Sprintf("ALTER TABLE %v ADD %v %v;", quotedTableName,
						Quote(e, field.DBName), sqlTag),
				}
				e.Scope.Exprs = append(e.Scope.Exprs, expr)
			}
		}
		err = CreateJoinTable(e, field)
		if err != nil {
			return err
		}
	}
	return AutoIndex(e, value)
}

//ShouldSaveAssociation return true if indeed we want the association to me
//model to be saved.
//
// This relies on a context value that is set at scope level with key
// model.SaveAssociation.This key must store a boolean value. It is possible to
// store the value as string "skip" if you want to skip the saving.
//
//TODO: There is really no need for the skip string, a boolean false is enough
//since it will make the return value false and skip saving associations.
func ShouldSaveAssociation(e *engine.Engine) bool {
	s, ok := e.Scope.Get(model.SaveAssociations)
	if ok {
		if v, k := s.(bool); k {
			return v
		}
		if v, k := s.(string); k && v == "skip" {
			return false
		}
	}
	return true
}

//HasConditions return true if engine e has any relevant condition for narrowing
//down queries.
//
// This goes down like this
//
// 	* the primary field is not zero or
// 	* there is WHERE condition or
// 	* there is OR condition or
// 	* there is NOT condition
func HasConditions(e *engine.Engine, modelValue interface{}) bool {
	f, err := PrimaryField(e, modelValue)
	return err == nil && !f.IsBlank ||
		len(e.Search.WhereConditions) > 0 ||
		len(e.Search.OrConditions) > 0 ||
		len(e.Search.NotConditions) > 0
}

//UpdatedAttrsWithValues returns a map of field names with value. This takes
//value, and updates any field that needts to be updated by adding the field
//name mapped to the new field value to the returned map results.
//
// That applies if the value is a struct. Any other type of values are handeled
// by ConvertInterfaceToMap function.
func UpdatedAttrsWithValues(e *engine.Engine, value interface{}) (results map[string]interface{}, hasUpdate bool) {
	v := reflect.ValueOf(e.Scope.Value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ConvertInterfaceToMap(e, value, false), true
	}

	results = map[string]interface{}{}
	for key, value := range ConvertInterfaceToMap(e, value, true) {
		field, err := FieldByName(e, e.Scope.Value, key)
		if err != nil {
			//TODO return error?
		} else {
			if ChangeableField(e, field) {

				if _, ok := value.(*model.Expr); ok {
					hasUpdate = true
					results[field.DBName] = value
				} else {

					err := field.Set(value)
					if field.IsNormal {
						hasUpdate = true
						if err == errmsg.ErrUnaddressable {
							results[field.DBName] = value
						} else {
							results[field.DBName] = field.Field.Interface()
						}
					}
				}
			}
		}
	}
	return
}

//ConvertInterfaceToMap tries to convert value into a map[string]interface{}
//
// The map keys are field names, and the values are the supposed field values.
// This cunction only supports maps, []interface{} and structs.
//
// For [interface{}, if the first value is a string, then it must be a
// succession of key pair values like
//	["name","gernest","age",1000]
//
// That user case arises when the optional argument slice like args... is used
// where the type is string.
//
// 	// Provided you have a function
//	func some(args...string){}
//	//and you use to pass the following argumens.
//	some("name?","gernest")
//	//passsing args to this function will yield map["name?"]="gernest"
func ConvertInterfaceToMap(e *engine.Engine, values interface{}, withIgnoredField bool) map[string]interface{} {
	var attrs = map[string]interface{}{}
	switch value := values.(type) {
	case map[string]interface{}:
		return value
	case []interface{}:
		if len(value) > 0 {
			switch value[0].(type) {
			case string:

				// If the first key is a string. The whole slice is treated as a
				// succesive ke,value pairs.
				size := len(value)
				pos := 0
				for pos < size {
					if pos+1 < size {
						attrs[fmt.Sprint(value[pos])] = value[pos+1]
					}
					pos += 2
				}
			default:
				for _, v := range value {
					for key, value := range ConvertInterfaceToMap(e, v, withIgnoredField) {
						attrs[key] = value
					}
				}
			}
		}

	case interface{}:
		reflectValue := reflect.ValueOf(values)

		switch reflectValue.Kind() {
		case reflect.Map:
			for _, key := range reflectValue.MapKeys() {
				attrs[util.ToDBName(key.Interface().(string))] = reflectValue.MapIndex(key).Interface()
			}
		default:
			f, err := Fields(e, values)
			if err != nil {
			} else {
				for _, field := range f {
					if !field.IsBlank && (withIgnoredField || !field.IsIgnored) {
						attrs[field.DBName] = field.Field.Interface()
					}
				}
			}

		}
	}
	return attrs
}

//SaveFieldAsAssociation saves associations.
//
// This returns relationship that can be saved, the relationship is taken from
// field provided that the field is not blank,is changeable and is not an
// ignored field.
//
// Only works if the field has tag SAVE_ASSOCIATION
func SaveFieldAsAssociation(e *engine.Engine, field *model.Field) (bool, *model.Relationship) {
	if ChangeableField(e, field) && !field.IsBlank && !field.IsIgnored {
		if value, ok := field.TagSettings["SAVE_ASSOCIATIONS"]; !ok || (value != "false" && value != "skip") {
			if relationship := field.Relationship; relationship != nil {
				return true, relationship
			}
		}
	}
	return false, nil
}

//Initialize initializes value for e.Scope.Value There are three areas where we
//look for values to initialize the model with.
//
// 	e.Seach.WhereConditions
// 	e.Search.InitAttrs
//	e.Search.AssignAttr
//
func Initialize(e *engine.Engine) {
	for _, clause := range e.Search.WhereConditions {
		UpdatedAttrsWithValues(e, clause["query"])
	}
	UpdatedAttrsWithValues(e, e.Search.InitAttrs)
	UpdatedAttrsWithValues(e, e.Search.AssignAttrs)
}
