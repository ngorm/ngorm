package hooks

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gernest/ngorm/builder"
	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/errmsg"
	"github.com/gernest/ngorm/model"
	"github.com/gernest/ngorm/scope"
	"github.com/gernest/ngorm/search"
	"github.com/gernest/ngorm/util"
)

//Query executes SQL querries.
func Query(b *Book, e *engine.Engine) error {
	var isSlice, isPtr bool
	var resultType reflect.Type
	results := reflect.ValueOf(e.Scope.Value)
	if results.Kind() == reflect.Ptr {
		results = results.Elem()
	}
	if orderBy, ok := e.Scope.Get(model.OrderByPK); ok {
		pf, err := scope.PrimaryField(e, e.Scope.Value)
		if err != nil {
		} else {
			search.Order(e, fmt.Sprintf("%v.%v %v",
				scope.QuotedTableName(e, e.Scope.Value), scope.Quote(e, pf.DBName), orderBy))
		}

	}
	if value, ok := e.Scope.Get(model.QueryDestination); ok {
		results = reflect.Indirect(reflect.ValueOf(value))
	}
	if kind := results.Kind(); kind == reflect.Slice {
		isSlice = true
		resultType = results.Type().Elem()
		results.Set(reflect.MakeSlice(results.Type(), 0, 0))

		if resultType.Kind() == reflect.Ptr {
			isPtr = true
			resultType = resultType.Elem()
		}
	} else if kind != reflect.Struct {
		return errors.New("unsupported destination, should be slice or struct")
	}
	err := builder.PrepareQuery(e, e.Scope.Value)
	if err != nil {
		return err
	}
	e.RowsAffected = 0
	if str, ok := e.Scope.Get(model.QueryOption); ok {
		e.Scope.SQL += util.AddExtraSpaceIfExist(fmt.Sprint(str))
	}

	rows, err := e.SQLDB.Query(e.Scope.SQL, e.Scope.SQLVars...)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	columns, _ := rows.Columns()
	for rows.Next() {
		e.RowsAffected++
		elem := results
		if isSlice {
			elem = reflect.New(resultType).Elem()
		}
		fields, err := scope.Fields(e, elem.Addr().Interface())
		if err != nil {
			return err
		}
		scope.Scan(rows, columns, fields)
		if isSlice {
			if isPtr {
				results.Set(reflect.Append(results, elem.Addr()))
			} else {
				results.Set(reflect.Append(results, elem))
			}
		}
	}
	return nil
}

//AfterQuery executes any call back after the  Qeery hook has been executed. Any
//callback registered with qeky model.HookQueryAfterFind will be executed.
func AfterQuery(b *Book, e *engine.Engine) error {
	af, ok := b.Query.Get(model.HookAfterFindQuery)
	if ok {
		return af.Exec(b, e)
	}
	return nil
}

//BeforeCreate a callback executed before crating anew record.
func BeforeCreate(b *Book, e *engine.Engine) error {
	bs, ok := b.Create.Get(model.HookBeforeSave)
	if ok {
		err := bs.Exec(b, e)
		if err != nil {
			return err
		}
	}
	bc, ok := b.Create.Get(model.HookBeforeCreate)
	if ok {
		err := bc.Exec(b, e)
		if err != nil {
			return err
		}
	}
	return nil
}

//Create the hook executed to create a new record.
func Create(b *Book, e *engine.Engine) error {
	var (
		columns, placeholders []string

		// The blank columns with default values
		cv []string
	)
	fds, err := scope.Fields(e, e.Scope.Value)
	if err != nil {
		return err
	}

	for _, field := range fds {
		if scope.ChangeableField(e, field) {
			if field.IsNormal {
				if field.IsBlank && field.HasDefaultValue {
					cv = append(cv, scope.Quote(e, field.DBName))
					e.Scope.Set(model.BlankColWithValue, cv)
				} else if !field.IsPrimaryKey || !field.IsBlank {
					columns = append(columns, scope.Quote(e, field.DBName))
					placeholders = append(placeholders, scope.AddToVars(e, field.Field.Interface()))
				}
			} else if field.Relationship != nil && field.Relationship.Kind == "belongs_to" {
				for _, foreignKey := range field.Relationship.ForeignDBNames {
					foreignField, err := scope.FieldByName(e, e.Scope.Value, foreignKey)
					if err != nil {
						return err
					}
					if !scope.ChangeableField(e, foreignField) {
						columns = append(columns, scope.Quote(e, foreignField.DBName))
						placeholders = append(placeholders, scope.AddToVars(e, foreignField.Field.Interface()))
					}
				}
			}
		}
	}

	var (
		returningColumn = "*"
		tableName       = scope.QuotedTableName(e, e.Scope.Value)

		extraOption string
	)

	primaryField, err := scope.PrimaryField(e, e.Scope.Value)
	if err != nil {
		return err
	}
	if str, ok := e.Scope.Get(model.InsertOptions); ok {
		extraOption = fmt.Sprint(str)
	}

	if primaryField != nil {
		returningColumn = scope.Quote(e, primaryField.DBName)
	}

	lastInsertIDReturningSuffix :=
		e.Dialect.LastInsertIDReturningSuffix(tableName, returningColumn)

	if len(columns) == 0 {
		sql := fmt.Sprintf(
			"INSERT INTO %v DEFAULT VALUES%v%v",
			tableName,
			util.AddExtraSpaceIfExist(extraOption),
			util.AddExtraSpaceIfExist(lastInsertIDReturningSuffix),
		)
		e.Scope.SQL = strings.Replace(sql, "$$", "?", -1)
	} else {
		sql := fmt.Sprintf(
			"INSERT INTO %v (%v) VALUES (%v)%v%v",
			scope.QuotedTableName(e, e.Scope.Value),
			strings.Join(columns, ","),
			strings.Join(placeholders, ","),
			util.AddExtraSpaceIfExist(extraOption),
			util.AddExtraSpaceIfExist(lastInsertIDReturningSuffix),
		)
		e.Scope.SQL = strings.Replace(sql, "$$", "?", -1)
	}

	return nil
}

//CreateExec executes the INSERT query and assigns primary key if it is not set
//assuming the primary key is the ID field.
func CreateExec(b *Book, e *engine.Engine) error {
	primaryField, err := scope.PrimaryField(e, e.Scope.Value)
	if err != nil {
		return err
	}
	returningColumn := "*"
	if primaryField != nil {
		returningColumn = scope.Quote(e, primaryField.DBName)
	}
	tableName := scope.QuotedTableName(e, e.Scope.Value)
	lastInsertIDReturningSuffix :=
		e.Dialect.LastInsertIDReturningSuffix(tableName, returningColumn)
	if lastInsertIDReturningSuffix == "" || primaryField == nil {
		tx, err := e.SQLDB.Begin()
		if err != nil {
			return err
		}
		result, err := tx.Exec(e.Scope.SQL, e.Scope.SQLVars...)
		if err != nil {
			rerr := tx.Rollback()
			if rerr != nil {
				return rerr
			}
			return err
		}
		err = tx.Commit()
		if err != nil {
			return err
		}
		// set rows affected count
		e.RowsAffected, _ = result.RowsAffected()

		// set primary value to primary field
		if primaryField != nil && primaryField.IsBlank {
			primaryValue, err := result.LastInsertId()
			if err != nil {
				return err
			}
			_ = primaryField.Set(primaryValue)
		}
	} else {
		if primaryField.Field.CanAddr() {
			err := e.SQLDB.QueryRow(
				e.Scope.SQL,
				e.Scope.SQLVars...,
			).Scan(primaryField.Field.Addr().Interface())
			if err != nil {
				return err
			}
			primaryField.IsBlank = false
			e.RowsAffected = 1
		} else {
			return errmsg.ErrUnaddressable
		}
	}
	return nil
}

//AfterCreate hook executed after a new record has been created.
func AfterCreate(b *Book, e *engine.Engine) error {
	ac, ok := b.Create.Get(model.HookAfterCreate)
	if ok {
		err := ac.Exec(b, e)
		if err != nil {
			return err
		}
	}
	as, ok := b.Create.Get(model.HookAfterSave)
	if ok {
		err := as.Exec(b, e)
		if err != nil {
			return err
		}
	}
	return nil
}

//BeforeUpdate handles preparations for updating records. This just calls two
//hooks.
//
//	model.HookBeforeSave
//
// If this hook succeds then It calls
//
//	model.HookBeforeUpdate
func BeforeUpdate(b *Book, e *engine.Engine) error {
	if !scope.HasConditions(e, e.Scope.Value) {
		return errors.New("missing WHERE condition for update")
	}
	if _, ok := e.Scope.Get(model.UpdateColumn); !ok {
		if bs, ok := b.Save.Get(model.HookBeforeSave); ok {
			err := bs.Exec(b, e)
			if err != nil {
				return err
			}
		}
		if bu, ok := b.Update.Get(model.HookBeforeUpdate); ok {
			err := bu.Exec(b, e)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//AfterUpdate handles things needed to be done after updating records. This just
//calls two hooks
//
//	model.HookAfterUpdate
//
// If this hook succeds then It calls
//
//	model.HookAfterSave
func AfterUpdate(b *Book, e *engine.Engine) error {
	if !scope.HasConditions(e, e.Scope.Value) {
		return errors.New("missing WHERE condition for update")
	}
	if _, ok := e.Scope.Get(model.UpdateColumn); !ok {
		if au, ok := b.Update.Get(model.HookAfterUpdate); ok {
			err := au.Exec(b, e)
			if err != nil {
				return err
			}
		}
		if as, ok := b.Save.Get(model.HookAfterSave); ok {
			err := as.Exec(b, e)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//UpdtaeTimestamp sets the value of UpdatedAt field.
func UpdtaeTimestamp(b *Book, e *engine.Engine) error {
	if _, ok := e.Scope.Get(model.UpdateColumn); !ok {
		return scope.SetColumn(e, "UpdatedAt", time.Now())
	}
	return nil
}
