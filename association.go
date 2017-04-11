package ngorm

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ngorm/ngorm/model"
	"github.com/ngorm/ngorm/scope"
	"github.com/ngorm/ngorm/util"
)

// Association provides utility functions for dealing with association queries
type Association struct {
	db     *DB
	column string
	field  *model.Field
}

// Find find out all related associations
func (a *Association) Find(v interface{}) error {
	return a.db.Related(v, a.column)
}

// Append append new associations for many2many, has_many, replace current association for has_one, belongs_to
func (a *Association) Append(values ...interface{}) error {
	if rel := a.field.Relationship; rel.Kind == "has_one" {
		return a.Replace(values...)
	}
	return a.Save(values...)
}

// Replace replace current associations with new one
func (a *Association) Replace(values ...interface{}) error {
	e := a.db.e
	rel := a.field.Relationship
	ndb := a.db.Begin()
	a.field.Set(reflect.Zero(a.field.Field.Type()))
	if err := a.Save(values...); err != nil {
		return err
	}
	switch rel.Kind {
	case "belongs_to":
		if len(values) == 0 {
			m := make(map[string]interface{})
			for _, foreignKey := range rel.ForeignDBNames {
				m[foreignKey] = nil
			}
			return a.db.Begin().Model(e.Scope.Value).UpdateColumn(m)
		}
	default:
		if rel.PolymorphicDBName != "" {
			ndb = ndb.Where(fmt.Sprintf("%v = ?",
				scope.Quote(e, rel.PolymorphicDBName)), rel.PolymorphicValue)
		}
		// Delete Relations except new created
		if len(values) > 0 {
			var fNames, fdDBNames []string
			if rel.Kind == "many_to_many" {
				// if many to many relations, get association fields name from association foreign keys
				se := e.Clone()
				v := reflect.New(a.field.Field.Type()).Interface()
				for idx, dbName := range rel.AssociationForeignFieldNames {
					if field, err := scope.FieldByName(se, v, dbName); err == nil {
						fNames = append(fNames, field.Name)
						fdDBNames = append(fdDBNames, rel.AssociationForeignDBNames[idx])
					}
				}
			} else {
				// If has one/many relations, use primary keys
				fds, err := scope.PrimaryFields(e.Clone(), reflect.New(a.field.Field.Type()).Interface())
				if err != nil {
					return err
				}
				for _, field := range fds {
					fNames = append(fNames, field.Name)
					fdDBNames = append(fdDBNames, field.DBName)
				}
			}

			newPrimaryKeys := util.ColumnAsArray(fNames, a.field.Field.Interface())

			if len(newPrimaryKeys) > 0 {
				sql := fmt.Sprintf("%v NOT IN (%v)",
					scope.ToQueryCondition(ndb.e, fdDBNames),
					scope.ToQueryMarks(newPrimaryKeys))
				ndb = ndb.Where(sql, util.ToQueryValues(newPrimaryKeys)...)
			}
		}
		if rel.Kind == "many_to_many" {
			// if many to many relations, delete related relations from join table
			var sourceForeignFieldNames []string

			for _, dbName := range rel.ForeignFieldNames {
				if field, err := scope.FieldByName(e, e.Scope.Value, dbName); err == nil {
					sourceForeignFieldNames = append(sourceForeignFieldNames, field.Name)
				}
			}

			if sourcePrimaryKeys := util.ColumnAsArray(sourceForeignFieldNames, e.Scope.Value); len(sourcePrimaryKeys) > 0 {
				ndb = ndb.Where(fmt.Sprintf("%v IN (%v)",
					scope.ToQueryCondition(e, rel.ForeignDBNames),
					scope.ToQueryMarks(sourcePrimaryKeys)),
					util.ToQueryValues(sourcePrimaryKeys)...)

				// association.setErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, newDB, relationship))
			}
		} else if rel.Kind == "has_one" || rel.Kind == "has_many" {
			// has_one or has_many relations, set foreign key to be nil (TODO or delete them?)
			var foreignKeyMap = map[string]interface{}{}
			for idx, foreignKey := range rel.ForeignDBNames {
				foreignKeyMap[foreignKey] = nil
				if field, err := scope.FieldByName(e, e.Scope.Value, rel.AssociationForeignFieldNames[idx]); err == nil {
					ndb = ndb.Where(fmt.Sprintf("%v = ?",
						scope.Quote(e, foreignKey)), field.Field.Interface())
				}
			}

			fieldValue := reflect.New(a.field.Field.Type()).Interface()
			return ndb.Model(fieldValue).UpdateColumn(foreignKeyMap)
		}

	}

	return nil
}

// Save save passed values as associations
func (a *Association) Save(values ...interface{}) error {
	rel := a.field.Relationship
	e := a.db.e
	field := a.field

	saveAssociation := func(reflectValue reflect.Value) error {
		// value has to been pointer
		if reflectValue.Kind() != reflect.Ptr {
			reflectPtr := reflect.New(reflectValue.Type())
			reflectPtr.Elem().Set(reflectValue)
			reflectValue = reflectPtr
		}

		// value has to been saved for many2many
		if rel.Kind == "many_to_many" {
			_, err := scope.PrimaryField(e.Clone(), reflectValue.Interface())
			if err == nil {
				return a.db.Begin().Save(reflectValue.Interface())
			}
			return nil

		}

		// Assign Fields
		var fieldType = field.Field.Type()
		var setFieldBackToValue, setSliceFieldBackToValue bool
		if reflectValue.Type().AssignableTo(fieldType) {
			field.Set(reflectValue)
		} else if reflectValue.Type().Elem().AssignableTo(fieldType) {
			// if field's type is struct, then need to set value back to argument after save
			setFieldBackToValue = true
			field.Set(reflectValue.Elem())
		} else if fieldType.Kind() == reflect.Slice {
			if reflectValue.Type().AssignableTo(fieldType.Elem()) {
				field.Set(reflect.Append(field.Field, reflectValue))
			} else if reflectValue.Type().Elem().AssignableTo(fieldType.Elem()) {
				// if field's type is slice of struct, then need to set value back to argument after save
				setSliceFieldBackToValue = true
				field.Set(reflect.Append(field.Field, reflectValue.Elem()))
			}
		}

		if rel.Kind == "many_to_many" {
			// association.setErr(relationship.JoinTableHandler.Add(relationship.JoinTableHandler, scope.NewDB(), scope.Value, reflectValue.Interface()))
		} else {
			err := a.db.Begin().Select(field.Name).Save(e.Scope.Value)
			if err != nil {
				return err
			}
			if setFieldBackToValue {
				reflectValue.Elem().Set(field.Field)
			} else if setSliceFieldBackToValue {
				reflectValue.Elem().Set(field.Field.Index(field.Field.Len() - 1))
			}
		}
		return nil
	}

	for _, value := range values {
		reflectValue := reflect.ValueOf(value)
		indirectReflectValue := reflect.Indirect(reflectValue)
		if indirectReflectValue.Kind() == reflect.Struct {
			saveAssociation(reflectValue)
		} else if indirectReflectValue.Kind() == reflect.Slice {
			for i := 0; i < indirectReflectValue.Len(); i++ {
				err := saveAssociation(indirectReflectValue.Index(i))
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("invalid value type")
		}
	}
	return nil
}

// Count return the count of current associations
func (a *Association) Count() (int, error) {
	var (
		count      = 0
		rel        = a.field.Relationship
		fieldValue = a.field.Field.Interface()
		query      = a.db.Begin().Model(fieldValue)
	)
	if rel.Kind == "many_to_many" {
		err := scope.JoinWith(rel.JoinTableHandler, query.e, a.db.e.Scope.Value)
		if err != nil {
			return 0, err
		}
	} else if rel.Kind == "has_many" || rel.Kind == "has_one" {
		primaryKeys := util.ColumnAsArray(rel.AssociationForeignFieldNames, a.db.e.Scope.Value)
		query = query.Where(
			fmt.Sprintf("%v IN (%v)",
				scope.ToQueryCondition(a.db.e, rel.ForeignDBNames),
				scope.ToQueryMarks(primaryKeys)),
			util.ToQueryValues(primaryKeys)...,
		)
	} else if rel.Kind == "belongs_to" {
		primaryKeys := util.ColumnAsArray(rel.ForeignFieldNames, a.db.e.Scope.Value)
		query = query.Where(
			fmt.Sprintf("%v IN (%v)",
				scope.ToQueryCondition(a.db.e, rel.AssociationForeignDBNames),
				scope.ToQueryMarks(primaryKeys)),
			util.ToQueryValues(primaryKeys)...,
		)
	}

	if rel.PolymorphicType != "" {
		query = query.Where(
			fmt.Sprintf("%v%v = ?",
				a.db.e.Dialect.QueryFieldName(
					scope.QuotedTableName(a.db.e, fieldValue)),
				scope.Quote(a.db.e, rel.PolymorphicDBName)),
			rel.PolymorphicValue,
		)
	}
	err := query.Count(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
