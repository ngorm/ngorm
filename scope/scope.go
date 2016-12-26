package scope

import (
	"database/sql"
	"errors"
	"go/ast"
	"reflect"
	"strings"
	"time"

	"github.com/gernest/gorm/engine"
	"github.com/gernest/gorm/model"
	"github.com/gernest/gorm/util"
	"github.com/jinzhu/inflection"
)

func Quote(e *engine.Engine, str string) string {
	if strings.Index(str, ".") != -1 {
		newStrs := []string{}
		for _, s := range strings.Split(str, ".") {
			newStrs = append(newStrs, e.Parent.Dialect.Quote(s))
		}
		return strings.Join(newStrs, ".")
	}
	return e.Dialect.Quote(str)
}

func Fields(e *engine.Engine, value interface{}) []*model.Field {
	if e.Scope.Fields == nil {
		var fields []*model.Field
		i := reflect.ValueOf(value)
		if i.Kind() == reflect.Ptr {
			i = i.Elem()
		}
		isStruct := i.Kind() == reflect.Struct

		for _, structField := range GetModelStruct(e, value).StructFields {
			if isStruct {
				fieldValue := i
				for _, name := range structField.Names {
					fieldValue = reflect.Indirect(fieldValue).FieldByName(name)
				}
				fields = append(fields, &model.Field{
					StructField: structField,
					Field:       fieldValue,
					IsBlank:     isBlank(fieldValue)})
			} else {
				fields = append(fields, &model.Field{
					StructField: structField,
					IsBlank:     true})
			}
		}
		e.Scope.Fields = &fields
	}

	return *e.Scope.Fields
}

func isBlank(value reflect.Value) bool {
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func GetModelStruct(e *engine.Engine, value interface{}) *model.ModelStruct {
	var m model.ModelStruct
	// Scope value can't be nil
	if value == nil {
		return &m
	}

	refType := reflect.ValueOf(value).Type()
	for refType.Kind() == reflect.Slice || refType.Kind() == reflect.Ptr {
		refType = refType.Elem()
	}

	// Scope value need to be a struct
	if refType.Kind() != reflect.Struct {
		return &m
	}

	// Get Cached model struct
	if v := e.StructMap.Get(refType); value != nil {
		return v
	}

	m.ModelType = refType

	// Set default table name
	if tabler, ok := reflect.New(refType).Interface().(engine.Tabler); ok {
		m.DefaultTableName = tabler.TableName()
	} else {
		tableName := util.ToDBName(refType.Name())
		if e.Parent.SingularTable {
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
					for _, subField := range GetModelStruct(e, fieldValue).StructFields {
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
						BuildRelationSlice(e, refType, &m, field)
					case reflect.Struct:
						BuildRelationStruct(e, refType, &m, field)
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
		if field := model.GetForeignField("id", m.StructFields); field != nil {
			field.IsPrimaryKey = true
			m.PrimaryFields = append(m.PrimaryFields, field)
		}
	}

	e.StructMap.Set(refType, &m)
	return &m
}

func BuildRelationSlice(e *engine.Engine, refType reflect.Type, m *model.ModelStruct, field *model.StructField) {
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
				if foreignField := model.GetForeignField(fk, m.StructFields); foreignField != nil {
					// source foreign keys (db names)
					rel.ForeignFieldNames = append(rel.ForeignFieldNames, foreignField.DBName)
					// join table foreign keys for source
					joinTableDBName := util.ToDBName(refType.Name()) + "_" + foreignField.DBName
					rel.ForeignDBNames = append(rel.ForeignDBNames, joinTableDBName)
				}
			}

			// if no association foreign keys defined with tag
			if len(associationForeignKeys) == 0 {
				for _, field := range PrimaryFields(e, toScope) {
					associationForeignKeys = append(associationForeignKeys, field.DBName)
				}
			}

			for _, name := range associationForeignKeys {
				if field, ok := FieldByName(e, toScope, name); ok {
					// association foreign keys (db names)
					rel.AssociationForeignFieldNames = append(rel.AssociationForeignFieldNames, field.DBName)
					// join table foreign keys for association
					joinTableDBName := util.ToDBName(elemType.Name()) + "_" + field.DBName
					rel.AssociationForeignDBNames = append(rel.AssociationForeignDBNames, joinTableDBName)
				}
			}

			//joinTableHandler := JoinTableHandler{}
			//joinTableHandler.Setup(relationship, many2many, refType, elemType)
			//relationship.JoinTableHandler = &joinTableHandler
			field.Relationship = rel
		} else {
			// User has many comments, associationType is User, comment use UserID as foreign key
			var associationType = refType.Name()
			var toFields = GetModelStruct(e, toScope).StructFields
			rel.Kind = "has_many"

			if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
				// Dog has many toys, tag polymorphic is Owner, then associationType is Owner
				// Toy use OwnerID, OwnerType ('dogs') as foreign key
				if polymorphicType := model.GetForeignField(polymorphic+"Type", toFields); polymorphicType != nil {
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
						if foreignField := model.GetForeignField(scopeFieldName, m.StructFields); foreignField != nil {
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
							if foreignField := model.GetForeignField(associationForeignKey, m.StructFields); foreignField != nil {
								associationForeignKeys = append(associationForeignKeys, associationForeignKey)
							}
						}
					}
					if len(associationForeignKeys) == 0 && len(fks) == 1 {
						associationForeignKeys = []string{PrimaryKey(e, e.Scope.Value)}
					}
				} else if len(fks) != len(associationForeignKeys) {
					e.AddError(errors.New("invalid foreign keys, should have same length"))
					return
				}
			}

			for idx, fk := range fks {
				if foreignField := model.GetForeignField(fk, toFields); foreignField != nil {
					if associationField := model.GetForeignField(associationForeignKeys[idx], m.StructFields); associationField != nil {
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
}

func BuildRelationStruct(e *engine.Engine, refType reflect.Type, m *model.ModelStruct, field *model.StructField) {
	var (
		// user has one profile, associationType is User, profile use UserID as foreign key
		// user belongs to profile, associationType is Profile, user use ProfileID as foreign key
		associationType           = refType.Name()
		rel                       = &model.Relationship{}
		toScope                   = reflect.New(field.Struct.Type).Interface()
		toFields                  = GetModelStruct(e, toScope).StructFields
		tagForeignKeys            []string
		tagAssociationForeignKeys []string
	)

	if fk := field.TagSettings["FOREIGNKEY"]; fk != "" {
		tagForeignKeys = strings.Split(field.TagSettings["FOREIGNKEY"], ",")
	}

	if fk := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; fk != "" {
		tagAssociationForeignKeys = strings.Split(field.TagSettings["ASSOCIATIONFOREIGNKEY"], ",")
	}

	if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
		// Cat has one toy, tag polymorphic is Owner, then associationType is Owner
		// Toy use OwnerID, OwnerType ('cats') as foreign key
		if polymorphicType := model.GetForeignField(polymorphic+"Type", toFields); polymorphicType != nil {
			associationType = polymorphic
			rel.PolymorphicType = polymorphicType.Name
			rel.PolymorphicDBName = polymorphicType.DBName
			// if Cat has several different types of toys set name for each (instead of default 'cats')
			if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
				rel.PolymorphicValue = value
			} else {
				rel.PolymorphicValue = TableName(e, e.Scope.Value)
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
					if foreignField := model.GetForeignField(associationForeignKey, m.StructFields); foreignField != nil {
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
						if foreignField := model.GetForeignField(associationForeignKey, m.StructFields); foreignField != nil {
							associationForeignKeys = append(associationForeignKeys, associationForeignKey)
						}
					}
				}
				if len(associationForeignKeys) == 0 && len(fks) == 1 {
					associationForeignKeys = []string{PrimaryKey(e, e.Scope.Value)}
				}
			} else if len(fks) != len(associationForeignKeys) {
				e.AddError(errors.New("invalid foreign keys, should have same length"))
				return
			}
		}

		for idx, fk := range fks {
			if foreignField := model.GetForeignField(fk, toFields); foreignField != nil {
				if scopeField := model.GetForeignField(associationForeignKeys[idx], m.StructFields); scopeField != nil {
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
				for _, primaryField := range PrimaryFields(e, toScope) {
					fks = append(fks, field.Name+primaryField.Name)
					associationForeignKeys = append(associationForeignKeys, primaryField.Name)
				}
			} else {
				// generate foreign keys with association foreign keys
				for _, associationForeignKey := range associationForeignKeys {
					if foreignField := model.GetForeignField(associationForeignKey, toFields); foreignField != nil {
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
						if foreignField := model.GetForeignField(associationForeignKey, toFields); foreignField != nil {
							associationForeignKeys = append(associationForeignKeys, associationForeignKey)
						}
					}
				}
				if len(associationForeignKeys) == 0 && len(fks) == 1 {
					associationForeignKeys = []string{PrimaryKey(e, toScope)}
				}
			} else if len(fks) != len(associationForeignKeys) {
				e.AddError(errors.New("invalid foreign keys, should have same length"))
				return
			}
		}

		for idx, fk := range fks {
			if foreignField := model.GetForeignField(fk, m.StructFields); foreignField != nil {
				if associationField := model.GetForeignField(associationForeignKeys[idx], toFields); associationField != nil {
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
}
func FieldByName(e *engine.Engine, value interface{}, name string) (*model.Field, bool) {
	var mostMatchedField *model.Field
	dbName := util.ToDBName(name)
	for _, field := range Fields(e, value) {
		if field.Name == name || field.DBName == name {
			return field, true
		}
		if field.DBName == dbName {
			mostMatchedField = field
		}
	}
	return mostMatchedField, mostMatchedField != nil
}

func PrimaryFields(e *engine.Engine, value interface{}) (fields []*model.Field) {
	for _, field := range Fields(e, value) {
		if field.IsPrimaryKey {
			fields = append(fields, field)
		}
	}
	return fields
}

func PrimaryField(e *engine.Engine, value interface{}) *model.Field {
	if primaryFields := GetModelStruct(e, e.Scope.Value).PrimaryFields; len(primaryFields) > 0 {
		if len(primaryFields) > 1 {
			if field, ok := FieldByName(e, value, "id"); ok {
				return field
			}
		}
		return PrimaryFields(e, value)[0]
	}
	return nil
}

func TableName(e *engine.Engine, value interface{}) string {
	if e.Search != nil && len(e.Search.TableName) > 0 {
		return e.Search.TableName
	}

	if tabler, ok := e.Scope.Value.(engine.Tabler); ok {
		return tabler.TableName()
	}

	if tabler, ok := e.Scope.Value.(engine.DbTabler); ok {
		return tabler.TableName(e)
	}
	return GetModelStruct(e, value).DefaultTableName
}

func PrimaryKey(e *engine.Engine, value interface{}) string {
	if field := PrimaryField(e, value); field != nil {
		return field.DBName
	}
	return ""
}
