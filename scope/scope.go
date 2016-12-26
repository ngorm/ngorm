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
	var modelStruct model.ModelStruct
	// Scope value can't be nil
	if value == nil {
		return &modelStruct
	}

	reflectType := reflect.ValueOf(value).Type()
	for reflectType.Kind() == reflect.Slice || reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	// Scope value need to be a struct
	if reflectType.Kind() != reflect.Struct {
		return &modelStruct
	}

	// Get Cached model struct
	if v := e.StructMap.Get(reflectType); value != nil {
		return v
	}

	modelStruct.ModelType = reflectType

	// Set default table name
	if tabler, ok := reflect.New(reflectType).Interface().(engine.Tabler); ok {
		modelStruct.DefaultTableName = tabler.TableName()
	} else {
		tableName := util.ToDBName(reflectType.Name())
		if e.Parent.SingularTable {
			tableName = inflection.Plural(tableName)
		}
		modelStruct.DefaultTableName = tableName
	}

	// Get all fields
	for i := 0; i < reflectType.NumField(); i++ {
		if fieldStruct := reflectType.Field(i); ast.IsExported(fieldStruct.Name) {
			field := &model.StructField{
				Struct:      fieldStruct,
				Name:        fieldStruct.Name,
				Names:       []string{fieldStruct.Name},
				Tag:         fieldStruct.Tag,
				TagSettings: model.ParseTagSetting(fieldStruct.Tag),
			}

			// is ignored field
			if _, ok := field.TagSettings["-"]; ok {
				field.IsIgnored = true
			} else {
				if _, ok := field.TagSettings["PRIMARY_KEY"]; ok {
					field.IsPrimaryKey = true
					modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
				}

				if _, ok := field.TagSettings["DEFAULT"]; ok {
					field.HasDefaultValue = true
				}

				if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok && !field.IsPrimaryKey {
					field.HasDefaultValue = true
				}

				indirectType := fieldStruct.Type
				for indirectType.Kind() == reflect.Ptr {
					indirectType = indirectType.Elem()
				}

				fieldValue := reflect.New(indirectType).Interface()
				if _, isScanner := fieldValue.(sql.Scanner); isScanner {
					// is scanner
					field.IsScanner, field.IsNormal = true, true
					if indirectType.Kind() == reflect.Struct {
						for i := 0; i < indirectType.NumField(); i++ {
							for key, value := range model.ParseTagSetting(indirectType.Field(i).Tag) {
								field.TagSettings[key] = value
							}
						}
					}
				} else if _, isTime := fieldValue.(*time.Time); isTime {
					// is time
					field.IsNormal = true
				} else if _, ok := field.TagSettings["EMBEDDED"]; ok || fieldStruct.Anonymous {
					// is embedded struct
					for _, subField := range GetModelStruct(e, fieldValue).StructFields {
						subField = subField.Clone()
						subField.Names = append([]string{fieldStruct.Name}, subField.Names...)
						if prefix, ok := field.TagSettings["EMBEDDED_PREFIX"]; ok {
							subField.DBName = prefix + subField.DBName
						}
						if subField.IsPrimaryKey {
							modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, subField)
						}
						modelStruct.StructFields = append(modelStruct.StructFields, subField)
					}
					continue
				} else {
					// build relationships
					switch indirectType.Kind() {
					case reflect.Slice:
						defer func(field *model.StructField) {
							var (
								rel                    = &model.Relationship{}
								toScope                = reflect.New(field.Struct.Type).Interface()
								foreignKeys            []string
								associationForeignKeys []string
								elemType               = field.Struct.Type
							)

							if foreignKey := field.TagSettings["FOREIGNKEY"]; foreignKey != "" {
								foreignKeys = strings.Split(field.TagSettings["FOREIGNKEY"], ",")
							}

							if foreignKey := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
								associationForeignKeys = strings.Split(field.TagSettings["ASSOCIATIONFOREIGNKEY"], ",")
							}

							for elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
								elemType = elemType.Elem()
							}

							if elemType.Kind() == reflect.Struct {
								if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
									rel.Kind = "many_to_many"

									// if no foreign keys defined with tag
									if len(foreignKeys) == 0 {
										for _, field := range modelStruct.PrimaryFields {
											foreignKeys = append(foreignKeys, field.DBName)
										}
									}

									for _, foreignKey := range foreignKeys {
										if foreignField := model.GetForeignField(foreignKey, modelStruct.StructFields); foreignField != nil {
											// source foreign keys (db names)
											rel.ForeignFieldNames = append(rel.ForeignFieldNames, foreignField.DBName)
											// join table foreign keys for source
											joinTableDBName := util.ToDBName(reflectType.Name()) + "_" + foreignField.DBName
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
									//joinTableHandler.Setup(relationship, many2many, reflectType, elemType)
									//relationship.JoinTableHandler = &joinTableHandler
									field.Relationship = rel
								} else {
									// User has many comments, associationType is User, comment use UserID as foreign key
									var associationType = reflectType.Name()
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
									if len(foreignKeys) == 0 {
										// if no association foreign keys defined with tag
										if len(associationForeignKeys) == 0 {
											for _, field := range modelStruct.PrimaryFields {
												foreignKeys = append(foreignKeys, associationType+field.Name)
												associationForeignKeys = append(associationForeignKeys, field.Name)
											}
										} else {
											// generate foreign keys from defined association foreign keys
											for _, scopeFieldName := range associationForeignKeys {
												if foreignField := model.GetForeignField(scopeFieldName, modelStruct.StructFields); foreignField != nil {
													foreignKeys = append(foreignKeys, associationType+foreignField.Name)
													associationForeignKeys = append(associationForeignKeys, foreignField.Name)
												}
											}
										}
									} else {
										// generate association foreign keys from foreign keys
										if len(associationForeignKeys) == 0 {
											for _, foreignKey := range foreignKeys {
												if strings.HasPrefix(foreignKey, associationType) {
													associationForeignKey := strings.TrimPrefix(foreignKey, associationType)
													if foreignField := model.GetForeignField(associationForeignKey, modelStruct.StructFields); foreignField != nil {
														associationForeignKeys = append(associationForeignKeys, associationForeignKey)
													}
												}
											}
											if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
												associationForeignKeys = []string{PrimaryKey(e, e.Scope.Value)}
											}
										} else if len(foreignKeys) != len(associationForeignKeys) {
											e.AddError(errors.New("invalid foreign keys, should have same length"))
											return
										}
									}

									for idx, foreignKey := range foreignKeys {
										if foreignField := model.GetForeignField(foreignKey, toFields); foreignField != nil {
											if associationField := model.GetForeignField(associationForeignKeys[idx], modelStruct.StructFields); associationField != nil {
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
						}(field)
					case reflect.Struct:
						//defer func(field *StructField) {
						//var (
						//// user has one profile, associationType is User, profile use UserID as foreign key
						//// user belongs to profile, associationType is Profile, user use ProfileID as foreign key
						//associationType           = reflectType.Name()
						//relationship              = &Relationship{}
						//toScope                   = scope.New(reflect.New(field.Struct.Type).Interface())
						//toFields                  = toScope.GetStructFields()
						//tagForeignKeys            []string
						//tagAssociationForeignKeys []string
						//)

						//if foreignKey := field.TagSettings["FOREIGNKEY"]; foreignKey != "" {
						//tagForeignKeys = strings.Split(field.TagSettings["FOREIGNKEY"], ",")
						//}

						//if foreignKey := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
						//tagAssociationForeignKeys = strings.Split(field.TagSettings["ASSOCIATIONFOREIGNKEY"], ",")
						//}

						//if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
						//// Cat has one toy, tag polymorphic is Owner, then associationType is Owner
						//// Toy use OwnerID, OwnerType ('cats') as foreign key
						//if polymorphicType := getForeignField(polymorphic+"Type", toFields); polymorphicType != nil {
						//associationType = polymorphic
						//relationship.PolymorphicType = polymorphicType.Name
						//relationship.PolymorphicDBName = polymorphicType.DBName
						//// if Cat has several different types of toys set name for each (instead of default 'cats')
						//if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
						//relationship.PolymorphicValue = value
						//} else {
						//relationship.PolymorphicValue = scope.TableName()
						//}
						//polymorphicType.IsForeignKey = true
						//}
						//}

						//// Has One
						//{
						//var foreignKeys = tagForeignKeys
						//var associationForeignKeys = tagAssociationForeignKeys
						//// if no foreign keys defined with tag
						//if len(foreignKeys) == 0 {
						//// if no association foreign keys defined with tag
						//if len(associationForeignKeys) == 0 {
						//for _, primaryField := range modelStruct.PrimaryFields {
						//foreignKeys = append(foreignKeys, associationType+primaryField.Name)
						//associationForeignKeys = append(associationForeignKeys, primaryField.Name)
						//}
						//} else {
						//// generate foreign keys form association foreign keys
						//for _, associationForeignKey := range tagAssociationForeignKeys {
						//if foreignField := getForeignField(associationForeignKey, modelStruct.StructFields); foreignField != nil {
						//foreignKeys = append(foreignKeys, associationType+foreignField.Name)
						//associationForeignKeys = append(associationForeignKeys, foreignField.Name)
						//}
						//}
						//}
						//} else {
						//// generate association foreign keys from foreign keys
						//if len(associationForeignKeys) == 0 {
						//for _, foreignKey := range foreignKeys {
						//if strings.HasPrefix(foreignKey, associationType) {
						//associationForeignKey := strings.TrimPrefix(foreignKey, associationType)
						//if foreignField := getForeignField(associationForeignKey, modelStruct.StructFields); foreignField != nil {
						//associationForeignKeys = append(associationForeignKeys, associationForeignKey)
						//}
						//}
						//}
						//if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
						//associationForeignKeys = []string{scope.PrimaryKey()}
						//}
						//} else if len(foreignKeys) != len(associationForeignKeys) {
						//scope.Err(errors.New("invalid foreign keys, should have same length"))
						//return
						//}
						//}

						//for idx, foreignKey := range foreignKeys {
						//if foreignField := getForeignField(foreignKey, toFields); foreignField != nil {
						//if scopeField := getForeignField(associationForeignKeys[idx], modelStruct.StructFields); scopeField != nil {
						//foreignField.IsForeignKey = true
						//// source foreign keys
						//relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, scopeField.Name)
						//relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, scopeField.DBName)

						//// association foreign keys
						//relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
						//relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
						//}
						//}
						//}
						//}

						//if len(relationship.ForeignFieldNames) != 0 {
						//relationship.Kind = "has_one"
						//field.Relationship = relationship
						//} else {
						//var foreignKeys = tagForeignKeys
						//var associationForeignKeys = tagAssociationForeignKeys

						//if len(foreignKeys) == 0 {
						//// generate foreign keys & association foreign keys
						//if len(associationForeignKeys) == 0 {
						//for _, primaryField := range toScope.PrimaryFields() {
						//foreignKeys = append(foreignKeys, field.Name+primaryField.Name)
						//associationForeignKeys = append(associationForeignKeys, primaryField.Name)
						//}
						//} else {
						//// generate foreign keys with association foreign keys
						//for _, associationForeignKey := range associationForeignKeys {
						//if foreignField := getForeignField(associationForeignKey, toFields); foreignField != nil {
						//foreignKeys = append(foreignKeys, field.Name+foreignField.Name)
						//associationForeignKeys = append(associationForeignKeys, foreignField.Name)
						//}
						//}
						//}
						//} else {
						//// generate foreign keys & association foreign keys
						//if len(associationForeignKeys) == 0 {
						//for _, foreignKey := range foreignKeys {
						//if strings.HasPrefix(foreignKey, field.Name) {
						//associationForeignKey := strings.TrimPrefix(foreignKey, field.Name)
						//if foreignField := getForeignField(associationForeignKey, toFields); foreignField != nil {
						//associationForeignKeys = append(associationForeignKeys, associationForeignKey)
						//}
						//}
						//}
						//if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
						//associationForeignKeys = []string{toScope.PrimaryKey()}
						//}
						//} else if len(foreignKeys) != len(associationForeignKeys) {
						//scope.Err(errors.New("invalid foreign keys, should have same length"))
						//return
						//}
						//}

						//for idx, foreignKey := range foreignKeys {
						//if foreignField := getForeignField(foreignKey, modelStruct.StructFields); foreignField != nil {
						//if associationField := getForeignField(associationForeignKeys[idx], toFields); associationField != nil {
						//foreignField.IsForeignKey = true

						//// association foreign keys
						//relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, associationField.Name)
						//relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationField.DBName)

						//// source foreign keys
						//relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
						//relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
						//}
						//}
						//}

						//if len(relationship.ForeignFieldNames) != 0 {
						//relationship.Kind = "belongs_to"
						//field.Relationship = relationship
						//}
						//}
						//}(field)
					default:
						field.IsNormal = true
					}
				}
			}

			// Even it is ignored, also possible to decode db value into the field
			if value, ok := field.TagSettings["COLUMN"]; ok {
				field.DBName = value
			} else {
				field.DBName = util.ToDBName(fieldStruct.Name)
			}

			modelStruct.StructFields = append(modelStruct.StructFields, field)
		}
	}

	if len(modelStruct.PrimaryFields) == 0 {
		if field := model.GetForeignField("id", modelStruct.StructFields); field != nil {
			field.IsPrimaryKey = true
			modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
		}
	}

	e.StructMap.Set(reflectType, &modelStruct)
	return &modelStruct
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
