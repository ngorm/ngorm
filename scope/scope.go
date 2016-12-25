package scope

import (
	"database/sql"
	"go/ast"
	"reflect"
	"strings"
	"time"

	"github.com/gernest/gorm/base"
	"github.com/gernest/gorm/engine"
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

func Fields(e *engine.Engine) []*base.Field {
	if e.Scope.Fields == nil {
		var fields []*base.Field
		i := reflect.ValueOf(e.Scope.Value)
		if i.Kind() == reflect.Ptr {
			i = i.Elem()
		}
		isStruct := i.Kind() == reflect.Struct

		for _, structField := range GetModelStruct(e).StructFields {
			if isStruct {
				fieldValue := i
				for _, name := range structField.Names {
					fieldValue = reflect.Indirect(fieldValue).FieldByName(name)
				}
				fields = append(fields, &base.Field{
					StructField: structField,
					Field:       fieldValue,
					IsBlank:     isBlank(fieldValue)})
			} else {
				fields = append(fields, &base.Field{
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

func GetModelStruct(e *engine.Engine) *base.ModelStruct {
	var modelStruct base.ModelStruct
	// Scope value can't be nil
	if e.Scope.Value == nil {
		return &modelStruct
	}

	reflectType := reflect.ValueOf(e.Scope.Value).Type()
	for reflectType.Kind() == reflect.Slice || reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	// Scope value need to be a struct
	if reflectType.Kind() != reflect.Struct {
		return &modelStruct
	}

	// Get Cached model struct
	if value := e.StructMap.Get(reflectType); value != nil {
		return value
	}

	modelStruct.ModelType = reflectType

	// Set default table name
	if tabler, ok := reflect.New(reflectType).Interface().(base.Tabler); ok {
		modelStruct.DefaultTableName = tabler.TableName()
	} else {
		tableName := base.ToDBName(reflectType.Name())
		//if scope.db == nil || !scope.db.parent.singularTable {
		//tableName = inflection.Plural(tableName)
		//}
		modelStruct.DefaultTableName = tableName
	}

	// Get all fields
	for i := 0; i < reflectType.NumField(); i++ {
		if fieldStruct := reflectType.Field(i); ast.IsExported(fieldStruct.Name) {
			field := &base.StructField{
				Struct:      fieldStruct,
				Name:        fieldStruct.Name,
				Names:       []string{fieldStruct.Name},
				Tag:         fieldStruct.Tag,
				TagSettings: base.ParseTagSetting(fieldStruct.Tag),
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
							for key, value := range base.ParseTagSetting(indirectType.Field(i).Tag) {
								field.TagSettings[key] = value
							}
						}
					}
				} else if _, isTime := fieldValue.(*time.Time); isTime {
					// is time
					field.IsNormal = true
				} else if _, ok := field.TagSettings["EMBEDDED"]; ok || fieldStruct.Anonymous {
					// is embedded struct
					//for _, subField := range scope.New(fieldValue).GetStructFields() {
					//subField = subField.clone()
					//subField.Names = append([]string{fieldStruct.Name}, subField.Names...)
					//if prefix, ok := field.TagSettings["EMBEDDED_PREFIX"]; ok {
					//subField.DBName = prefix + subField.DBName
					//}
					//if subField.IsPrimaryKey {
					//modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, subField)
					//}
					//modelStruct.StructFields = append(modelStruct.StructFields, subField)
					//}
					continue
				} else {
					// build relationships
					switch indirectType.Kind() {
					case reflect.Slice:
						//defer func(field *base.StructField) {
						//var (
						//relationship           = &Relationship{}
						//toScope                = scope.New(reflect.New(field.Struct.Type).Interface())
						//foreignKeys            []string
						//associationForeignKeys []string
						//elemType               = field.Struct.Type
						//)

						//if foreignKey := field.TagSettings["FOREIGNKEY"]; foreignKey != "" {
						//foreignKeys = strings.Split(field.TagSettings["FOREIGNKEY"], ",")
						//}

						//if foreignKey := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
						//associationForeignKeys = strings.Split(field.TagSettings["ASSOCIATIONFOREIGNKEY"], ",")
						//}

						//for elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
						//elemType = elemType.Elem()
						//}

						//if elemType.Kind() == reflect.Struct {
						//if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
						//relationship.Kind = "many_to_many"

						//// if no foreign keys defined with tag
						//if len(foreignKeys) == 0 {
						//for _, field := range modelStruct.PrimaryFields {
						//foreignKeys = append(foreignKeys, field.DBName)
						//}
						//}

						//for _, foreignKey := range foreignKeys {
						//if foreignField := getForeignField(foreignKey, modelStruct.StructFields); foreignField != nil {
						//// source foreign keys (db names)
						//relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.DBName)
						//// join table foreign keys for source
						//joinTableDBName := ToDBName(reflectType.Name()) + "_" + foreignField.DBName
						//relationship.ForeignDBNames = append(relationship.ForeignDBNames, joinTableDBName)
						//}
						//}

						//// if no association foreign keys defined with tag
						//if len(associationForeignKeys) == 0 {
						//for _, field := range toScope.PrimaryFields() {
						//associationForeignKeys = append(associationForeignKeys, field.DBName)
						//}
						//}

						//for _, name := range associationForeignKeys {
						//if field, ok := toScope.FieldByName(name); ok {
						//// association foreign keys (db names)
						//relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, field.DBName)
						//// join table foreign keys for association
						//joinTableDBName := ToDBName(elemType.Name()) + "_" + field.DBName
						//relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, joinTableDBName)
						//}
						//}

						//joinTableHandler := JoinTableHandler{}
						//joinTableHandler.Setup(relationship, many2many, reflectType, elemType)
						//relationship.JoinTableHandler = &joinTableHandler
						//field.Relationship = relationship
						//} else {
						//// User has many comments, associationType is User, comment use UserID as foreign key
						//var associationType = reflectType.Name()
						//var toFields = toScope.GetStructFields()
						//relationship.Kind = "has_many"

						//if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
						//// Dog has many toys, tag polymorphic is Owner, then associationType is Owner
						//// Toy use OwnerID, OwnerType ('dogs') as foreign key
						//if polymorphicType := getForeignField(polymorphic+"Type", toFields); polymorphicType != nil {
						//associationType = polymorphic
						//relationship.PolymorphicType = polymorphicType.Name
						//relationship.PolymorphicDBName = polymorphicType.DBName
						//// if Dog has multiple set of toys set name of the set (instead of default 'dogs')
						//if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
						//relationship.PolymorphicValue = value
						//} else {
						//relationship.PolymorphicValue = scope.TableName()
						//}
						//polymorphicType.IsForeignKey = true
						//}
						//}

						//// if no foreign keys defined with tag
						//if len(foreignKeys) == 0 {
						//// if no association foreign keys defined with tag
						//if len(associationForeignKeys) == 0 {
						//for _, field := range modelStruct.PrimaryFields {
						//foreignKeys = append(foreignKeys, associationType+field.Name)
						//associationForeignKeys = append(associationForeignKeys, field.Name)
						//}
						//} else {
						//// generate foreign keys from defined association foreign keys
						//for _, scopeFieldName := range associationForeignKeys {
						//if foreignField := getForeignField(scopeFieldName, modelStruct.StructFields); foreignField != nil {
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
						//if associationField := getForeignField(associationForeignKeys[idx], modelStruct.StructFields); associationField != nil {
						//// source foreign keys
						//foreignField.IsForeignKey = true
						//relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, associationField.Name)
						//relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationField.DBName)

						//// association foreign keys
						//relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
						//relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
						//}
						//}
						//}

						//if len(relationship.ForeignFieldNames) != 0 {
						//field.Relationship = relationship
						//}
						//}
						//} else {
						//field.IsNormal = true
						//}
						//}(field)
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
				field.DBName = base.ToDBName(fieldStruct.Name)
			}

			modelStruct.StructFields = append(modelStruct.StructFields, field)
		}
	}

	if len(modelStruct.PrimaryFields) == 0 {
		if field := base.GetForeignField("id", modelStruct.StructFields); field != nil {
			field.IsPrimaryKey = true
			modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
		}
	}

	e.StructMap.Set(reflectType, &modelStruct)
	return &modelStruct
}
