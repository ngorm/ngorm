//Package hooks contains callbacks/hooks used by ngorm.
package hooks

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ngorm/ngorm/builder"
	"github.com/ngorm/ngorm/dialects"
	"github.com/ngorm/ngorm/engine"
	"github.com/ngorm/ngorm/errmsg"
	"github.com/ngorm/ngorm/model"
	"github.com/ngorm/ngorm/scope"
	"github.com/ngorm/ngorm/search"
	"github.com/ngorm/ngorm/util"
)

//Query executes sql Query without transaction. This first executes QuerySQL
//which generates appropriate SQl query then QueryExec hook is executed to
//execute the generated query.
//
// If all is well HookAfterQuery is executed, if this hook is not registered
// then no error is returned.
func Query(e *engine.Engine) error {
	err := QuerySQL(e)
	if err != nil {
		return err
	}
	err = QueryExec(e)
	if err != nil {
		return err
	}
	return AfterQuery(e)
}

//QueryExec  executes SQL queries and scans the result to the pointer object
//that is in e.Scope.Value.
//
// The value stored in e.Scope.Value can only either be a struct or a slice
// other types are not supported.
//
// NOTE: queries are not executed in transaction context.
func QueryExec(e *engine.Engine) error {
	var isSlice, isPtr bool
	var resultType reflect.Type
	results := reflect.ValueOf(e.Scope.Value)
	if results.Kind() == reflect.Ptr {
		results = results.Elem()
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
	if e.RowsAffected == 0 && !isSlice {
		return errmsg.ErrRecordNotFound
	}
	return nil
}

//QuerySQL generates SQL for queries. This uses `builder.PrepareQuery` to build
//the desired SQL query.
func QuerySQL(e *engine.Engine) error {
	if orderBy, ok := e.Scope.Get(model.OrderByPK); ok {
		pf, err := scope.PrimaryField(e, e.Scope.ValueOf())
		if err != nil {
		} else {
			search.Order(e, fmt.Sprintf("%v%v %v",
				e.Dialect.QueryFieldName(
					scope.QuotedTableName(e, e.Scope.ValueOf())),
				scope.Quote(e, pf.DBName), orderBy))
		}

	}
	return builder.PrepareQuery(e, e.Scope.ValueOf())
}

//AfterQuery executes any call back after the  Query hook has been executed. Any
//callback registered with key model.HookQueryAfterFind will be executed.
func AfterQuery(e *engine.Engine) error {
	if e.Search.Preload != nil {
		err := Preload(e)
		if err != nil {
			return err
		}
	}
	return nil
}

//Create the hook executed to create a new record.
func Create(e *engine.Engine) error {
	err := CreateSQL(e)
	if err != nil {
		return err
	}
	err = CreateExec(e)
	if err != nil {
		return err
	}
	return AfterCreate(e)
}

func create(e *engine.Engine) error {
	var (
		cols, placeholders []string

		// The blank columns with default values
		cv []string
	)
	fds, err := scope.Fields(e, e.Scope.ValueOf())
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
					cols = append(cols, scope.Quote(e, field.DBName))
					placeholders = append(placeholders, scope.AddToVars(e, field.Field.Interface()))
				}
			} else if field.Relationship != nil && field.Relationship.Kind == "belongs_to" {
				for _, foreignKey := range field.Relationship.ForeignDBNames {
					foreignField, err := scope.FieldByName(e, e.Scope.ValueOf(), foreignKey)
					if err != nil {
						return err
					}
					if !scope.ChangeableField(e, foreignField) {
						cols = append(cols, scope.Quote(e, foreignField.DBName))
						placeholders = append(placeholders,
							scope.AddToVars(e, foreignField.Field.Interface()))
					}
				}
			}
		}
	}

	var (
		returningColumn = "*"
		tableName       = scope.QuotedTableName(e, e.Scope.ValueOf())

		extraOption string
	)

	primaryField, err := scope.PrimaryField(e, e.Scope.ValueOf())
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

	if len(cols) == 0 {
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
			scope.QuotedTableName(e, e.Scope.ValueOf()),
			strings.Join(cols, ","),
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
func CreateExec(e *engine.Engine) error {
	primaryField, err := scope.PrimaryField(e, e.Scope.ValueOf())
	if err != nil {
		return err
	}
	returningColumn := "*"
	if primaryField != nil {
		returningColumn = scope.Quote(e, primaryField.DBName)
	}
	tableName := scope.QuotedTableName(e, e.Scope.ValueOf())
	lastInsertIDReturningSuffix :=
		e.Dialect.LastInsertIDReturningSuffix(tableName, returningColumn)
	if lastInsertIDReturningSuffix == "" || primaryField == nil {
		var result sql.Result
		if dialects.IsQL(e.Dialect) {
			tx, err := e.SQLDB.Begin()
			if err != nil {
				return err
			}
			result, err = tx.Exec(e.Scope.SQL, e.Scope.SQLVars...)
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
		} else {
			result, err = e.SQLDB.Exec(e.Scope.SQL, e.Scope.SQLVars...)
			if err != nil {
				return err
			}
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

//AfterCreate executes hooks after Creating records
func AfterCreate(e *engine.Engine) error {
	if dialects.IsQL(e.Dialect) {
		QLAfterCreate(e)
	}
	return AfterAssociation(e)
}

//QLAfterCreate hook executed after a new record has been created. This is for
//ql dialect use only.
func QLAfterCreate(e *engine.Engine) error {
	ne := e.Clone()
	defer engine.Put(ne)
	ne.Scope.Set(model.IgnoreProtectedAttrs, true)
	ne.Scope.Set(model.UpdateInterface, util.ToSearchableMap(e.Scope.Value))
	ne.Scope.ContextValue(e.Scope.Value)

	err := UpdateSQL(ne)
	if err != nil {
		return err
	}
	err = fixWhere(ne.Scope)
	if err != nil {
		return err
	}
	return UpdateExec(ne)
}

func fixWhere(s *model.Scope) error {
	src := s.SQL
	i := " id = "
	rep := " id()= "
	w := "WHERE"
	lastWhere := strings.LastIndex(src, w)
	if lastWhere == -1 {
		return nil
	}
	lastID := strings.LastIndex(src, i)
	if lastID == -1 {
		return nil
	}
	if lastID < lastWhere {
		return nil
	}
	s.SQL = src[:lastID] + rep + src[lastID+len(i):]
	n := lastID + len(i) + 1
	ni, err := strconv.Atoi(string(src[n]))
	if err != nil {
		return err
	}
	ni--
	nv := s.SQLVars[ni]
	switch v := nv.(type) {
	case uint64:
		s.SQLVars[ni] = int64(v)
	}
	return nil
}

//BeforeUpdate handles preparations for updating records. This just calls two
//hooks.
//
//	model.HookBeforeSave
//
// If this hook succeeds then It calls
//
//	model.HookBeforeUpdate
func BeforeUpdate(e *engine.Engine) error {
	if !scope.HasConditions(e, e.Scope.Value) {
		return errors.New("missing WHERE condition for update")
	}

	// set timestamps if any
	err := UpdateTimestamp(e)
	if err != nil {
		return err
	}

	// assign update attrs
	err = AssignUpdatingAttrs(e)
	if err != nil {
		return err
	}

	// save before associations
	err = SaveBeforeAssociation(e)
	if err != nil {
		return err
	}
	return nil
}

//AfterUpdate handles things needed to be done after updating records. This just
//calls two hooks
//
//	model.HookAfterUpdate
//
// If this hook succeeds then It calls
//
//	model.HookAfterSave
func AfterUpdate(e *engine.Engine) error {
	if !scope.HasConditions(e, e.Scope.Value) {
		return errors.New("missing WHERE condition for update")
	}
	return AfterAssociation(e)
}

//UpdateTimestamp sets the value of UpdatedAt field.
func UpdateTimestamp(e *engine.Engine) error {
	if _, ok := e.Scope.Get(model.UpdateColumn); !ok {
		return scope.SetColumn(e, "UpdatedAt", time.Now())
	}
	return nil
}

//AssignUpdatingAttrs assigns value for the attributes that are supposed to be
//updated.
func AssignUpdatingAttrs(e *engine.Engine) error {
	if attrs, ok := e.Scope.Get(model.UpdateInterface); ok {
		if u, uok := scope.UpdatedAttrsWithValues(e, attrs); uok {
			e.Scope.Set(model.UpdateAttrs, u)
		}
	}
	return nil
}

//SaveBeforeAssociation saves associations on the model
func SaveBeforeAssociation(e *engine.Engine) error {
	if !scope.ShouldSaveAssociation(e) {
		return nil
	}
	fds, err := scope.Fields(e, e.Scope.Value)
	if err != nil {
		return err
	}
	for _, field := range fds {
		if ok, relationship := scope.SaveFieldAsAssociation(e, field); ok && relationship.Kind == "belongs_to" {
			fieldValue := field.Field.Addr().Interface()

			// For the fieldValue, we need to make sure the value is saved into
			// the database.
			//
			// We have two hooks to use here, one model.HookCreateSQL which will
			// build sql for creating the new record and model.HookCreateExec
			// which will execute the generates SQL.
			ne := e.Clone()
			defer engine.Put(ne)
			ne.Scope.ContextValue(fieldValue)
			err = Create(ne)
			if err != nil {
				return err
			}
			if len(relationship.ForeignFieldNames) != 0 {
				// set value's foreign key
				for idx, fieldName := range relationship.ForeignFieldNames {
					associationForeignName := relationship.AssociationForeignDBNames[idx]
					foreignField, err := scope.FieldByName(e, fieldValue, associationForeignName)
					if err != nil {
						//TODO log this?
					} else {
						err = scope.SetColumn(e, fieldName, foreignField.Field.Interface())
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

//AfterAssociation saves associations on the model
func AfterAssociation(e *engine.Engine) error {
	if !scope.ShouldSaveAssociation(e) {
		return nil
	}
	fds, err := scope.Fields(e, e.Scope.ValueOf())
	if err != nil {
		return err
	}
	for _, field := range fds {
		if ok, rel := scope.SaveFieldAsAssociation(e, field); ok {
			switch rel.Kind {
			case "has_many",
				"has_one",
				"many_to_many":
				value := field.Field
				switch value.Kind() {
				case reflect.Slice:
					for i := 0; i < value.Len(); i++ {
						ne := e.Clone()
						defer engine.Put(ne)
						vi := value.Index(i)
						var elem interface{}
						if vi.Kind() == reflect.Ptr {
							elem = vi.Interface()
						} else {
							elem = vi.Addr().Interface()
						}
						ne.Scope.ContextValue(elem)
						if rel.JoinTableHandler == nil && len(rel.ForeignFieldNames) != 0 {
							for idx, fieldName := range rel.ForeignFieldNames {
								associationForeignName := rel.AssociationForeignFieldNames[idx]
								for _, fd := range fds {
									if fd.Name == associationForeignName {
										err = scope.SetColumn(ne, fieldName, fd.Field.Interface())
										if err != nil {
											return err
										}
									}
								}
							}
						}
						if rel.PolymorphicType != "" {
							err = scope.SetColumn(ne, rel.PolymorphicType, rel.PolymorphicValue)
							if err != nil {
								return err
							}
						}
						err = Create(ne)
						if err != nil {
							return err
						}
						if h := rel.JoinTableHandler; h != nil {
							ne.Scope.SQL = ""
							ne.Scope.SQLVars = nil
							expr, err := scope.AddJoinRelation(h.TableName, h, ne, e.Scope.Value, ne.Scope.Value)
							if err != nil {
								return err
							}
							if dialects.IsQL(e.Dialect) {
								expr.Q = util.WrapTX(expr.Q)
								tx, err := ne.SQLDB.Begin()
								if err != nil {
									return err
								}
								_, err = tx.Exec(expr.Q, expr.Args...)
								if err != nil {
									tx.Rollback()
									return err
								}
								if err = tx.Commit(); err != nil {
									return err
								}
							} else {
								_, err = ne.SQLDB.Exec(expr.Q, expr.Args...)
								if err != nil {
									return err
								}
							}
						}

					}
				default:
					fieldValue := field.Field.Addr().Interface()
					ne := e.Clone()
					defer engine.Put(ne)
					ne.Scope.ContextValue(fieldValue)
					if rel.PolymorphicType != "" {
						err = scope.SetColumn(ne, rel.PolymorphicType, rel.PolymorphicValue)
						if err != nil {
							return err
						}
					}
					if len(rel.ForeignFieldNames) != 0 {
						// set value's foreign key
						for idx, fieldName := range rel.ForeignFieldNames {
							associationForeignName := rel.AssociationForeignFieldNames[idx]
							for _, fd := range fds {
								if fd.Name == associationForeignName {
									err = scope.SetColumn(ne, fieldName, fd.Field.Interface())
									if err != nil {
										return err
									}
								}
							}
						}
					}
					err = Create(ne)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

//CreateSQL generates SQL for creating new record
func CreateSQL(e *engine.Engine) error {
	if scope.ShouldSaveAssociation(e) {
		err := SaveBeforeAssociation(e)
		if err != nil {
			return err
		}
	}
	err := UpdateTimestamp(e)
	if err != nil {
		return err
	}
	err = create(e)
	if err != nil {
		return err
	}
	buf := util.B.Get()
	defer util.B.Put(buf)
	if e.Dialect.GetName() == "ql" || e.Dialect.GetName() == "ql-mem" {
		_, _ = buf.WriteString("BEGIN TRANSACTION;\n")
	}
	if e.Scope.MultiExpr {
		for _, expr := range e.Scope.Exprs {
			_, _ = buf.WriteString("\t" + expr.Q + ";\n")
		}
	}
	_, _ = buf.WriteString("\t" + e.Scope.SQL + ";\n")
	if e.Dialect.GetName() == "ql" || e.Dialect.GetName() == "ql-mem" {
		_, _ = buf.WriteString("COMMIT;")
	}
	e.Scope.SQL = buf.String()
	return nil
}

//UpdateSQL builds query for updating records.
func UpdateSQL(e *engine.Engine) error {
	var sqls []string
	err := AssignUpdatingAttrs(e)
	if err != nil {
		return err
	}
	if updateAttrs, ok := e.Scope.Get(model.UpdateAttrs); ok {
		for column, value := range updateAttrs.(map[string]interface{}) {
			sqls = append(sqls, fmt.Sprintf("%v = %v",
				scope.Quote(e, column),
				scope.AddToVars(e, value)))
		}
	} else {
		fds, err := scope.Fields(e, e.Scope.Value)
		if err != nil {
			return err
		}
		for _, field := range fds {
			if scope.ChangeableField(e, field) {
				if !field.IsPrimaryKey && field.IsNormal {
					sqls = append(sqls, fmt.Sprintf("%v = %v",
						scope.Quote(e, field.DBName),
						scope.AddToVars(e, field.Field.Interface())))
				} else if rel := field.Relationship; rel != nil && rel.Kind == "belongs_to" {
					for _, foreignKey := range rel.ForeignDBNames {
						foreignField, err := scope.FieldByName(e, e.Scope.Value, foreignKey)
						if err != nil {
							//TODO log this?
						} else {
							if !scope.ChangeableField(e, foreignField) {
								sqls = append(sqls,
									fmt.Sprintf("%v = %v",
										scope.Quote(e, foreignField.DBName),
										scope.AddToVars(e, foreignField.Field.Interface())))
							}
						}
					}
				}
			}
		}
	}

	var extraOption string
	if str, ok := e.Scope.Get(model.UpdateOptions); ok {
		extraOption = fmt.Sprint(str)
	}

	if len(sqls) > 0 {
		c, err := builder.CombinedCondition(e, e.Scope.Value)
		if err != nil {
			return err
		}
		e.Scope.SQL = fmt.Sprintf(
			"UPDATE %v SET %v%v%v",
			scope.QuotedTableName(e, e.Scope.Value),
			strings.Join(sqls, ", "),
			util.AddExtraSpaceIfExist(c),
			util.AddExtraSpaceIfExist(extraOption),
		)

	}
	if dialects.IsQL(e.Dialect) {
		e.Scope.SQL = util.WrapTX(e.Scope.SQL)
	}
	return nil
}

//UpdateExec executes UPDATE sql. This assumes the query is already in
//e.Scope.SQL.
func UpdateExec(e *engine.Engine) error {
	if e.Scope.SQL == "" {
		return errors.New("missing update sql ")
	}
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
	r, err := result.RowsAffected()
	if err != nil {
		return err
	}
	e.RowsAffected = r
	return tx.Commit()
}

//Update generates and executes sql query for updating records.This relies on
//two hooks.
//	model.HookUpdateSQL
// Which generates the sql for UPDATE
//
//	model.HookUpdateExec
//which executes the UPDATE sql.
func Update(e *engine.Engine) error {

	// run before update hooks
	err := BeforeUpdate(e)
	if err != nil {
		return err
	}

	// generate update sql
	err = UpdateSQL(e)
	if err != nil {
		return err
	}

	// execute update sql
	err = UpdateExec(e)
	if err != nil {
		return err
	}

	// execute update sql
	return AfterUpdate(e)
}

// DeleteSQL generatesSQL for deleting records.
func DeleteSQL(e *engine.Engine) error {
	var extraOption string
	if str, ok := e.Scope.Get(model.DeleteOption); ok {
		extraOption = fmt.Sprint(str)
	}

	if e.Dialect.HasColumn(scope.TableName(e, e.Scope.Value), "DeletedAt") {
		c, err := builder.CombinedCondition(e, e.Scope.Value)
		if err != nil {
			return err
		}
		e.Scope.SQL = fmt.Sprintf(
			"UPDATE %v SET deleted_at=%v%v%v",
			scope.QuotedTableName(e, e.Scope.Value),
			scope.AddToVars(e, e.Now()),
			util.AddExtraSpaceIfExist(c),
			util.AddExtraSpaceIfExist(extraOption),
		)
		if e.Dialect.GetName() == "ql" || e.Dialect.GetName() == "ql-mem" {
			e.Scope.SQL = util.WrapTX(e.Scope.SQL)
		}
	} else {
		c, err := builder.CombinedCondition(e, e.Scope.Value)
		if err != nil {
			return err
		}
		e.Scope.SQL = fmt.Sprintf(
			"DELETE FROM %v%v%v",
			scope.QuotedTableName(e, e.Scope.Value),
			util.AddExtraSpaceIfExist(c),
			util.AddExtraSpaceIfExist(extraOption),
		)
		if e.Dialect.GetName() == "ql" || e.Dialect.GetName() == "ql-mem" {
			e.Scope.SQL = util.WrapTX(e.Scope.SQL)
		}
	}
	return nil
}

// BeforeDelete is called before deleting any record
func BeforeDelete(e *engine.Engine) error {
	if !scope.HasConditions(e, e.Scope.Value) {
		return errors.New("Missing WHERE clause while deleting")
	}
	return nil
}

// Delete deletes records. This makes sure to call BeforeDelete hook before
// deleting anything and also calls AfterDelete before exiting.
func Delete(e *engine.Engine) error {
	err := BeforeDelete(e)
	if err != nil {
		return err
	}

	err = DeleteSQL(e)
	if err != nil {
		return err
	}

	if dialects.IsQL(e.Dialect) {
		tx, err := e.SQLDB.Begin()
		if err != nil {
			return err
		}
		result, err := tx.Exec(e.Scope.SQL, e.Scope.SQLVars...)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		a, err := result.RowsAffected()
		if err != nil {
			return err
		}
		e.RowsAffected = a
		err = tx.Commit()
		if err != nil {
			return err
		}
	} else {
		result, err := e.SQLDB.Exec(e.Scope.SQL, e.Scope.SQLVars...)
		if err != nil {
			return err
		}
		a, err := result.RowsAffected()
		if err != nil {
			return err
		}
		e.RowsAffected = a
	}
	return nil
}

// Preload executes preload conditions.
func Preload(e *engine.Engine) error {
	if e.Search.Preload == nil {
		return nil
	}

	preloadedMap := map[string]bool{}
	fields, err := scope.Fields(e, e.Scope.Value)
	if err != nil {
		return err
	}

	for _, preload := range e.Search.Preload {
		var (
			preloadFields = strings.Split(preload.Schema, ".")
			cs            = e
			currentFields = fields
		)

		for idx, preloadField := range preloadFields {
			var conds []interface{}

			if cs == nil {
				continue
			}

			// if not preloaded
			if preloadKey := strings.Join(preloadFields[:idx+1], "."); !preloadedMap[preloadKey] {

				// assign search conditions to last preload
				if idx == len(preloadFields)-1 {
					conds = preload.Conditions
				}

				for _, field := range currentFields {
					if field.Name != preloadField || field.Relationship == nil {
						continue
					}

					switch field.Relationship.Kind {
					case "has_one":
						err = PreloadHasOne(cs, field, conds)
						if err != nil {
							return err
						}
					case "has_many":
						err = PreloadHasMany(cs, field, conds)
						if err != nil {
							return err
						}
					case "belongs_to":
						err = PreloadBelongsTo(cs, field, conds)
						if err != nil {
							return err
						}
					case "many_to_many":
						err = PreloadManyToMany(cs, field, conds)
						if err != nil {
							return err
						}
					default:
						return fmt.Errorf("hooks: can't preload %sunsupported relation",
							field.Relationship.Kind)
					}
					preloadedMap[preloadKey] = true
					break
				}

				if !preloadedMap[preloadKey] {
					m, err := scope.GetModelStruct(e, e.Scope.Value)
					if err != nil {
						return err
					}
					return fmt.Errorf("can't preload field %s for %s",
						preloadField, m.ModelType)
				}
			}

			// preload next level
			if idx < len(preloadFields)-1 {
				cs, err = ColumnAsScope(cs, preloadField)
				if err != nil {
					return err
				}
				currentFields, err = scope.Fields(cs, cs.Scope.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// PreloadBelongsTo preloads belongs_to relationship
func PreloadBelongsTo(e *engine.Engine, field *model.Field, conditions []interface{}) error {
	relation := field.Relationship

	// preload conditions
	pdb, pCond := PreloadDBWithConditions(e, conditions)

	// get relations's primary keys
	primaryKeys := util.ColumnAsArray(relation.ForeignFieldNames, e.Scope.Value)
	if len(primaryKeys) == 0 {
		return nil
	}

	// find relations
	query := fmt.Sprintf("%v IN (%v)",
		scope.ToQueryCondition(e, relation.AssociationForeignDBNames),
		util.ToQueryMarks(primaryKeys))
	values := util.ToQueryValues(primaryKeys)

	results := util.MakeSlice(field.Struct.Type)
	search.Where(pdb, query, values...)
	search.Inline(pdb, pCond...)
	pdb.Scope.ContextValue(results)

	err := Query(pdb)
	if err != nil {
		return err
	}

	// assign find results
	rVal := reflect.ValueOf(results)
	if rVal.Kind() == reflect.Ptr {
		rVal = rVal.Elem()
	}
	iScopeVal := reflect.ValueOf(e.Scope.Value)
	if iScopeVal.Kind() == reflect.Ptr {
		iScopeVal = iScopeVal.Elem()
	}

	for i := 0; i < rVal.Len(); i++ {
		result := rVal.Index(i)
		if iScopeVal.Kind() == reflect.Slice {
			value := util.GetValueFromFields(result, relation.AssociationForeignFieldNames)
			for j := 0; j < iScopeVal.Len(); j++ {
				object := iScopeVal.Index(j)
				if object.Kind() == reflect.Ptr {
					object = object.Elem()
				}
				if util.EqualAsString(util.GetValueFromFields(object, relation.ForeignFieldNames), value) {
					object.FieldByName(field.Name).Set(result)
				}
			}
		} else {
			err := field.Set(result)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PreloadManyToMany preloads many_to_many relation
func PreloadManyToMany(e *engine.Engine, field *model.Field, conditions []interface{}) error {
	var (
		relation         = field.Relationship
		joinTableHandler = relation.JoinTableHandler
		fieldType        = field.Struct.Type.Elem()
		foreignKeyValue  interface{}
		foreignKeyType   = reflect.ValueOf(&foreignKeyValue).Type()
		linkHash         = map[string][]reflect.Value{}
		isPtr            bool
	)

	if fieldType.Kind() == reflect.Ptr {
		isPtr = true
		fieldType = fieldType.Elem()
	}

	var sourceKeys = []string{}
	for _, key := range joinTableHandler.Source.ForeignKeys {
		sourceKeys = append(sourceKeys, key.DBName)
	}

	// preload conditions
	preloadDB, preloadConditions := PreloadDBWithConditions(e, conditions)

	// generate query with join table
	newScope := e.Clone()
	defer engine.Put(newScope)
	newScope.Scope.ContextValue(reflect.New(fieldType).Interface())
	search.Table(newScope, scope.TableName(newScope, newScope.Scope.Value))
	search.Select(newScope, "*")

	preloadDB, err := JoinWith(preloadDB, joinTableHandler, joinTableHandler, e.Scope.Value)
	if err != nil {
		return err
	}

	// preload inline conditions
	if len(preloadConditions) > 0 {
		search.Where(preloadDB, preloadConditions[0], preloadConditions[1:]...)
	}

	err = builder.PrepareQuery(preloadDB, preloadDB.Scope.Value)
	if err != nil {
		return err
	}

	rows, err := preloadDB.SQLDB.Query(preloadDB.Scope.SQL, preloadDB.Scope.SQLVars...)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	for rows.Next() {
		var (
			elem = reflect.New(fieldType).Elem()
		)
		fields, err := scope.Fields(e, elem.Addr().Interface())
		if err != nil {
			return err
		}

		// register foreign keys in join tables
		var joinTableFields []*model.Field
		for _, sourceKey := range sourceKeys {
			joinTableFields = append(joinTableFields, &model.Field{
				StructField: &model.StructField{DBName: sourceKey, IsNormal: true},
				Field:       reflect.New(foreignKeyType).Elem()})
		}

		scope.Scan(rows, columns, append(fields, joinTableFields...))

		var foreignKeys = make([]interface{}, len(sourceKeys))
		// generate hashed forkey keys in join table
		for idx, joinTableField := range joinTableFields {
			if !joinTableField.Field.IsNil() {
				foreignKeys[idx] = joinTableField.Field.Elem().Interface()
			}
		}
		hashedSourceKeys := util.ToString(foreignKeys)

		if isPtr {
			linkHash[hashedSourceKeys] = append(linkHash[hashedSourceKeys], elem.Addr())
		} else {
			linkHash[hashedSourceKeys] = append(linkHash[hashedSourceKeys], elem)
		}
	}

	// assign find results
	indirectScopeValue := reflect.ValueOf(e.Scope.Value)
	if indirectScopeValue.Kind() == reflect.Ptr {
		indirectScopeValue = indirectScopeValue.Elem()
	}
	var (
		fieldsSourceMap   = map[string][]reflect.Value{}
		foreignFieldNames = []string{}
	)

	for _, dbName := range relation.ForeignFieldNames {
		if field, err := scope.FieldByName(e, e.Scope.Value, dbName); err == nil {
			foreignFieldNames = append(foreignFieldNames, field.Name)
		}
	}

	if indirectScopeValue.Kind() == reflect.Slice {
		for j := 0; j < indirectScopeValue.Len(); j++ {
			object := indirectScopeValue.Index(j)
			if object.Kind() == reflect.Ptr {
				object = object.Elem()
			}
			key := util.ToString(util.GetValueFromFields(object, foreignFieldNames))
			fieldsSourceMap[key] = append(fieldsSourceMap[key], object.FieldByName(field.Name))
		}
	} else if indirectScopeValue.IsValid() {
		key := util.ToString(util.GetValueFromFields(indirectScopeValue, foreignFieldNames))
		fieldsSourceMap[key] = append(fieldsSourceMap[key], indirectScopeValue.FieldByName(field.Name))
	}
	for source, link := range linkHash {
		for i, field := range fieldsSourceMap[source] {
			//If not 0 this means Value is a pointer and we already added preloaded models to it
			if fieldsSourceMap[source][i].Len() != 0 {
				continue
			}
			field.Set(reflect.Append(fieldsSourceMap[source][i], link...))
		}

	}
	return nil
}

// JoinWith does sql join
func JoinWith(e *engine.Engine, s, handler *model.JoinTableHandler, source interface{}) (*engine.Engine, error) {
	ne := e.Clone()
	defer engine.Put(ne)
	ne.Scope.ContextValue(source)
	tableName := handler.TableName
	quotedTableName := scope.Quote(ne, tableName)
	var (
		joinConditions []string
		values         []interface{}
	)
	m, err := scope.GetModelStruct(ne, source)
	if err != nil {
		return nil, err
	}

	if s.Source.ModelType == m.ModelType {
		destinationTableName := scope.QuotedTableName(ne, reflect.New(s.Destination.ModelType).Interface())
		for _, foreignKey := range s.Destination.ForeignKeys {
			joinConditions = append(joinConditions, fmt.Sprintf("%v.%v = %v.%v",
				quotedTableName, scope.Quote(e, foreignKey.DBName),
				destinationTableName, scope.Quote(ne, foreignKey.AssociationDBName)))
		}

		var foreignDBNames []string
		var foreignFieldNames []string

		for _, foreignKey := range s.Source.ForeignKeys {
			foreignDBNames = append(foreignDBNames, foreignKey.DBName)
			if field, err := scope.FieldByName(ne, source, foreignKey.AssociationDBName); err == nil {
				foreignFieldNames = append(foreignFieldNames, field.Name)
			}
		}

		foreignFieldValues := util.ColumnAsArray(foreignFieldNames, e.Scope.Value)

		var condString string
		if len(foreignFieldValues) > 0 {
			var quotedForeignDBNames []string
			for _, dbName := range foreignDBNames {
				quotedForeignDBNames = append(quotedForeignDBNames, tableName+"."+dbName)
			}

			condString = fmt.Sprintf("%v IN (%v)",
				scope.ToQueryCondition(e, quotedForeignDBNames),
				util.ToQueryMarks(foreignFieldValues))

			keys := util.ColumnAsArray(foreignFieldNames, e.Scope.Value)
			values = append(values, util.ToQueryValues(keys))
		} else {
			condString = fmt.Sprintf("1 <> 1")
		}
		search.Join(ne, fmt.Sprintf("INNER JOIN %v ON %v",
			quotedTableName, strings.Join(joinConditions, " AND ")))
		search.Where(ne, condString, util.ToQueryValues(foreignFieldValues)...)
		return ne, nil
	}
	return nil, errors.New("wrong source type for join table handler")
}

// ColumnAsScope returnsnew Engine withthe value of the column used asscope.
func ColumnAsScope(e *engine.Engine, column string) (*engine.Engine, error) {
	iv := reflect.ValueOf(e.Scope.Value)
	if iv.Kind() == reflect.Ptr {
		iv = iv.Elem()
	}

	switch iv.Kind() {
	case reflect.Slice:
		m, err := scope.GetModelStruct(e, e.Scope.Value)
		if err != nil {
			return nil, err
		}
		if fieldStruct, ok := m.ModelType.FieldByName(column); ok {
			fieldType := fieldStruct.Type
			if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}

			// a map of results
			rm := map[interface{}]bool{}

			results := reflect.New(reflect.SliceOf(reflect.PtrTo(fieldType))).Elem()

			for i := 0; i < iv.Len(); i++ {
				result := iv.Index(i)
				if result.Kind() == reflect.Ptr {
					result = result.Elem()
				}
				result = result.FieldByName(column)
				if result.Kind() == reflect.Ptr {
					result = result.Elem()
				}
				if result.Kind() == reflect.Slice {
					for j := 0; j < result.Len(); j++ {
						if elem := result.Index(j); elem.CanAddr() && rm[elem.Addr()] != true {
							rm[elem.Addr()] = true
							results = reflect.Append(results, elem.Addr())
						}
					}
				} else if result.CanAddr() && rm[result.Addr()] != true {
					rm[result.Addr()] = true
					results = reflect.Append(results, result.Addr())
				}
			}
			ne := e.Clone()
			ne.Scope.ContextValue(results.Interface())
			return ne, nil
		}
	case reflect.Struct:
		if field := iv.FieldByName(column); field.CanAddr() {
			ne := e.Clone()
			ne.Scope.ContextValue(field.Addr().Interface())
			return ne, nil
		}
	}
	return nil, errors.New("can get engine out of column " + column)
}

// PreloadHasOne preloads has_one relation
func PreloadHasOne(e *engine.Engine, field *model.Field, conditions []interface{}) error {
	rel := field.Relationship

	// get relations's primary keys
	primaryKeys := util.ColumnAsArray(rel.AssociationForeignFieldNames, e.Scope.Value)
	if len(primaryKeys) == 0 {
		return nil
	}

	// preload conditions
	pdb, pCond := PreloadDBWithConditions(e, conditions)

	// find relations
	query := fmt.Sprintf("%v IN (%v)",
		scope.ToQueryCondition(e, rel.ForeignDBNames),
		util.ToQueryMarks(primaryKeys))
	values := util.ToQueryValues(primaryKeys)
	if rel.PolymorphicType != "" {
		query += fmt.Sprintf(" AND %v = ?", scope.Quote(e, rel.PolymorphicDBName))
		values = append(values, rel.PolymorphicValue)
	}

	results := util.MakeSlice(field.Struct.Type)
	search.Where(pdb, query, values...)
	search.Inline(pdb, pCond...)
	pdb.Scope.ContextValue(results)

	err := Query(pdb)
	if err != nil {
		return err
	}

	// assign find results
	rVal := reflect.ValueOf(results)
	if rVal.Kind() == reflect.Ptr {
		rVal = rVal.Elem()
	}
	iScopeVal := reflect.ValueOf(e.Scope.Value)
	if iScopeVal.Kind() == reflect.Ptr {
		iScopeVal = iScopeVal.Elem()
	}

	if iScopeVal.Kind() == reflect.Slice {
		for j := 0; j < iScopeVal.Len(); j++ {
			for i := 0; i < rVal.Len(); i++ {
				result := rVal.Index(i)
				foreignValues := util.GetValueFromFields(result, rel.ForeignFieldNames)
				iVal := iScopeVal.Index(j)
				if iVal.Kind() == reflect.Ptr {
					iVal = iVal.Elem()
				}
				if util.EqualAsString(util.GetValueFromFields(iVal, rel.AssociationForeignFieldNames), foreignValues) {
					iVal.FieldByName(field.Name).Set(result)
					break
				}
			}
		}
	} else {
		for i := 0; i < rVal.Len(); i++ {
			result := rVal.Index(i)
			err := field.Set(result)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PreloadHasMany preloads has_many relation
func PreloadHasMany(e *engine.Engine, field *model.Field, conditions []interface{}) error {
	rel := field.Relationship

	// get relations's primary keys
	primaryKeys := util.ColumnAsArray(rel.AssociationForeignFieldNames, e.Scope.Value)
	if len(primaryKeys) == 0 {
		return nil
	}

	// preload conditions
	pdb, pCond := PreloadDBWithConditions(e, conditions)

	// find relations
	query := fmt.Sprintf("%v IN (%v)",
		scope.ToQueryCondition(e, rel.ForeignDBNames),
		util.ToQueryMarks(primaryKeys))
	values := util.ToQueryValues(primaryKeys)
	if rel.PolymorphicType != "" {
		query += fmt.Sprintf(" AND %v = ?",
			scope.Quote(e, rel.PolymorphicDBName))
		values = append(values, rel.PolymorphicValue)
	}

	results := util.MakeSlice(field.Struct.Type)
	search.Where(pdb, query, values...)
	search.Inline(pdb, pCond...)
	pdb.Scope.ContextValue(results)

	err := Query(pdb)
	if err != nil {
		return err
	}

	// assign find results
	rVal := reflect.ValueOf(results)
	if rVal.Kind() == reflect.Ptr {
		rVal = rVal.Elem()
	}
	iScopeVal := reflect.ValueOf(e.Scope.Value)
	if iScopeVal.Kind() == reflect.Ptr {
		iScopeVal = iScopeVal.Elem()
	}

	if iScopeVal.Kind() == reflect.Slice {
		preloadMap := make(map[string][]reflect.Value)
		for i := 0; i < rVal.Len(); i++ {
			result := rVal.Index(i)
			foreignValues := util.GetValueFromFields(result, rel.ForeignFieldNames)
			preloadMap[util.ToString(foreignValues)] = append(preloadMap[util.ToString(foreignValues)], result)
		}

		for j := 0; j < iScopeVal.Len(); j++ {
			object := iScopeVal.Index(j)
			if object.Kind() == reflect.Ptr {
				object = object.Elem()
			}
			objectRealValue := util.GetValueFromFields(object, rel.AssociationForeignFieldNames)
			f := object.FieldByName(field.Name)
			if results, ok := preloadMap[util.ToString(objectRealValue)]; ok {
				f.Set(reflect.Append(f, results...))
			} else {
				f.Set(reflect.MakeSlice(f.Type(), 0, 0))
			}
		}
	} else {
		err := field.Set(rVal)
		if err != nil {
			return err
		}
	}
	return nil
}

// PreloadDBWithConditions returns engine with preload conditions set
func PreloadDBWithConditions(e *engine.Engine, conditions []interface{}) (*engine.Engine, []interface{}) {
	var (
		preloadDB         = e.Clone()
		preloadConditions []interface{}
	)

	for _, condition := range conditions {
		preloadConditions = append(preloadConditions, condition)
	}
	return preloadDB, preloadConditions
}
