package ngorm

import (
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

// Append append new associations for many2many, has_many, replace current
// association for has_one, belongs_to
//
// This wraps around Association.Save, verbatim  meaning you can have the same
// effect with Save method.
func (a *Association) Append(values ...interface{}) error {
	return a.Save(values...)
}

// Save save passed values as associations. This expects to have a single value
// for a has_one, belongs_to relationships. You can pass one or more values for
// many_to_many relationship.
func (a *Association) Save(values ...interface{}) error {
	if len(values) > 0 {
		e := a.db.e
		field := a.field
		rel := field.Relationship
		var v reflect.Value
		if rel.Kind == "has_one" {
			if len(values) > 1 {
				return fmt.Errorf("relation %s expect one struct value got %d", rel.Kind, len(values))
			}
			v = reflect.New(field.Field.Type())
			err := a.Find(v.Interface())
			if err != nil {
				return err
			}
			vp := v
			v = v.Elem()
			ov := reflect.ValueOf(values[0])
			if ov.Kind() == reflect.Ptr {
				ov = ov.Elem()
			}
			ovTyp := ov.Type()
			for i := 0; i < ovTyp.NumField(); i++ {
				fTyp := ovTyp.Field(i)
				fv := ov.FieldByName(fTyp.Name)
				if !isZero(fv) {
					fEv := v.FieldByName(fTyp.Name)
					fEv.Set(fv)
				}
			}
			field.Field.Set(v)
			return a.db.Begin().Save(vp.Interface())
		}
		v = reflect.MakeSlice(field.Struct.Type, 0, 0)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		for _, value := range values {
			fv := reflect.ValueOf(value)
			if fv.Kind() == reflect.Ptr {
				fv = fv.Elem()
			}
			v = reflect.Append(v, fv)
		}
		field.Field.Set(v)
		return a.db.Begin().Save(e.Scope.Value)
	}
	return nil
}

func isZero(v reflect.Value) bool {
	return v.Interface() == reflect.Zero(v.Type()).Interface()
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
				util.ToQueryMarks(primaryKeys)),
			util.ToQueryValues(primaryKeys)...,
		)
	} else if rel.Kind == "belongs_to" {
		primaryKeys := util.ColumnAsArray(rel.ForeignFieldNames, a.db.e.Scope.Value)
		query = query.Where(
			fmt.Sprintf("%v IN (%v)",
				scope.ToQueryCondition(a.db.e, rel.AssociationForeignDBNames),
				util.ToQueryMarks(primaryKeys)),
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
