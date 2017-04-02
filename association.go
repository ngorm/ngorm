package ngorm

import (
	"errors"
	"reflect"

	"github.com/ngorm/ngorm/model"
	"github.com/ngorm/ngorm/scope"
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
