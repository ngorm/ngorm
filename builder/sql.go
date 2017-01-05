// Package builder contains functions for SQL building. Most of the functions
// builds the SQL from the enine.Engine, and uptates the struct in a convenient
// manner.
//
// Be aware that, you should not pass a raw engine.Engine as some of the
// functions assumes engine.Engine.Search or engine.Engine.Scope is properly set.
package builder

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/model"
	"github.com/gernest/ngorm/regexes"
	"github.com/gernest/ngorm/scope"
)

//Where buiilds the sql where condition. The clause is a map
//of two important keys, one is query and the second is args. It is possible to
//use a struct instead of a map for clause, but for now we can stick with this
//else we will need to do a giant refactoring.
//
// query value can be of several types.
//
//  string,
//  int int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
//  sql.Nu[]int,
//  []int8, []int16, []int32, []int64, []uint, []uint8,
//  []uint16,[]uint32,[]uint64, []string, []interface{}
//  map[string]interface{}:
//  struct
//
// Note that if you supply a query as a struct then it should be a model.
// Example of a clause is,
//  map[string]interface{}{"query": query, "args": values}
// Where query can be anything of the above types and values is possibly a slice
// of positional values. Positional values are values which will be inserted in
// place of a placeholder e.g ?. For instance s querry,
//
//  select * from home where item=? && importance =?
// Then we can pass
//
//  []interface}{"milk", "critical"}
//
// The args slice has "milk" as the first thing and "critical" as the second.
// Now we can reconstruct the querry after appling the positional argument and
// get the following.
//
//  select * from home where item="milk" && importance="critical"
//
// In real case, the way the positional arguments are bound is database
// specific. For example ql uses $1,$2,$3 etc but also supports ?. You don't
// have to worry about this, it is automatically handled by the supported
// database dialects.
func Where(e *engine.Engine, modelValue interface{}, clause map[string]interface{}) (str string, err error) {
	switch value := clause["query"].(type) {
	case string:
		if regexes.IsNumber.MatchString(value) {
			return PrimaryCondition(e, modelValue, scope.AddToVars(e, value))
		} else if value != "" {
			str = fmt.Sprintf("(%v)", value)
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, sql.NullInt64:
		return PrimaryCondition(e, modelValue, scope.AddToVars(e, value))
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []string, []interface{}:
		pk, err := scope.PrimaryKey(e, modelValue)
		if err != nil {
			return "", err
		}
		str = fmt.Sprintf("(%v.%v IN (?))", scope.QuotedTableName(e, modelValue),
			scope.Quote(e, pk))
		clause["args"] = []interface{}{value}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			if value != nil {
				sqls = append(sqls, fmt.Sprintf("(%v.%v = %v)",
					scope.QuotedTableName(e, modelValue),
					scope.Quote(e, key), scope.AddToVars(e, value)))
			} else {
				sqls = append(sqls, fmt.Sprintf("(%v.%v IS NULL)",
					scope.QuotedTableName(e, modelValue), scope.Quote(e, key)))
			}
		}
		return strings.Join(sqls, " AND "), nil
	default:
		v := reflect.ValueOf(value)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() == reflect.Struct {
			var sqls []string
			fds, err := scope.Fields(e, value)
			if err != nil {
				return "", err
			}
			for _, field := range fds {
				if !field.IsIgnored && !field.IsBlank {
					sqls = append(sqls, fmt.Sprintf("(%v.%v = %v)",
						scope.QuotedTableName(e, value),
						scope.Quote(e, field.DBName),
						scope.AddToVars(e, field.Field.Interface())))
				}
			}
			return strings.Join(sqls, " AND "), nil
		}
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.ValueOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			if bytes, ok := arg.([]byte); ok {
				str = strings.Replace(str, "?", scope.AddToVars(e, bytes), 1)
			} else if values := reflect.ValueOf(arg); values.Len() > 0 {
				var tempMarks []string
				for i := 0; i < values.Len(); i++ {
					tempMarks = append(tempMarks, scope.AddToVars(e, values.Index(i).Interface()))
				}
				str = strings.Replace(str, "?", strings.Join(tempMarks, ","), 1)
			} else {
				str = strings.Replace(str, "?",
					scope.AddToVars(e, &model.Expr{Q: "NULL"}), 1)
			}
		default:
			if valuer, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = valuer.Value()
			}

			str = strings.Replace(str, "?", scope.AddToVars(e, arg), 1)
		}
	}
	return
}

//PrimaryCondition generates WHERE clause with the value set for primary key.
//This will return an error if the modelValue doesn't have primary key, the
//reason for modelValue not to have a primary key might be due to the modelValue
//not being a valid ngorm model, please check scope.PrimaryKey for more details.
//
// So, if the modelValue has primary key field id, and the value supplied is an
// integrer 14.The string generated will be id=$1 provided value is the first
// positional argument. Practially speaking it is the same as id=14.
func PrimaryCondition(e *engine.Engine, modelValue, value interface{}) (string, error) {
	pk, err := scope.PrimaryKey(e, modelValue)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%v.%v = %v)", scope.QuotedTableName(e, modelValue),
		scope.Quote(e, pk), value), nil
}

func WhereSQL(e *engine.Engine, modelValue interface{}) (sql string, err error) {
	var (
		quotedTableName                                = scope.QuotedTableName(e, modelValue)
		primaryConditions, andConditions, orConditions []string
	)

	if !e.Search.Unscoped && scope.HasColumn(e, modelValue, "deleted_at") {
		sql := fmt.Sprintf("%v.deleted_at IS NULL", quotedTableName)
		primaryConditions = append(primaryConditions, sql)
	}

	f, err := scope.PrimaryField(e, modelValue)
	if err != nil {
		return "", err
	}
	if !(f == nil || f.IsBlank) {
		pfs, err := scope.PrimaryFields(e, modelValue)
		if err != nil {
			return "", err
		}
		for _, field := range pfs {
			sql := fmt.Sprintf("%v.%v = %v", quotedTableName,
				scope.Quote(e, field.DBName), scope.AddToVars(e, field.Field.Interface()))
			primaryConditions = append(primaryConditions, sql)
		}
	}

	for _, clause := range e.Search.WhereConditions {
		sql, err := Where(e, modelValue, clause)
		if err != nil {
			return "", err
		}
		andConditions = append(andConditions, sql)
	}

	for _, clause := range e.Search.OrConditions {
		sql, err := Where(e, modelValue, clause)
		if err != nil {
			return "", err
		}
		orConditions = append(orConditions, sql)
	}

	for _, clause := range e.Search.NotConditions {
		sql, err := Not(e, modelValue, clause)
		if err != nil {
			return "", err
		}
		andConditions = append(andConditions, sql)
	}

	orSQL := strings.Join(orConditions, " OR ")
	combinedSQL := strings.Join(andConditions, " AND ")
	if len(combinedSQL) > 0 {
		if len(orSQL) > 0 {
			combinedSQL = combinedSQL + " OR " + orSQL
		}
	} else {
		combinedSQL = orSQL
	}

	if len(primaryConditions) > 0 {
		sql = "WHERE " + strings.Join(primaryConditions, " AND ")
		if len(combinedSQL) > 0 {
			sql = sql + " AND (" + combinedSQL + ")"
		}
	} else if len(combinedSQL) > 0 {
		sql = "WHERE " + combinedSQL
	}
	return
}

//Not generates sql for NOT condition. This accepts clause with two keys, one
//for query and the other for args( positional arguments)
//
// query value can be of several types.
//
//  string,
//  int int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
//  sql.Nu[]int,
//  []int8, []int16, []int32, []int64, []uint, []uint8,
//  []uint16,[]uint32,[]uint64, []string, []interface{}
//  map[string]interface{}:
//  struct
func Not(e *engine.Engine, modelValue interface{}, clause map[string]interface{}) (str string, err error) {
	var notEqualSQL string
	primaryKey, err := scope.PrimaryKey(e, modelValue)
	if err != nil {
		return "", err
	}
	switch value := clause["query"].(type) {
	case string:
		if regexes.IsNumber.MatchString(value) {
			id, _ := strconv.Atoi(value)
			return fmt.Sprintf("(%v <> %v)", scope.Quote(e, primaryKey), id), nil
		} else if regexes.Comparison.MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			notEqualSQL = fmt.Sprintf("NOT (%v)", value)
		} else {
			str = fmt.Sprintf("(%v.%v NOT IN (?))", scope.QuotedTableName(e, modelValue), scope.Quote(e, value))
			notEqualSQL = fmt.Sprintf("(%v.%v <> ?)", scope.QuotedTableName(e, modelValue), scope.Quote(e, value))
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, sql.NullInt64:
		return fmt.Sprintf("(%v.%v <> %v)", scope.QuotedTableName(e, modelValue), scope.Quote(e, primaryKey), value), nil
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []string:
		if reflect.ValueOf(value).Len() > 0 {
			str = fmt.Sprintf("(%v.%v NOT IN (?))", scope.QuotedTableName(e, modelValue), scope.Quote(e, primaryKey))
			clause["args"] = []interface{}{value}
		} else {
			return "", nil
		}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			if value != nil {
				sqls = append(sqls, fmt.Sprintf("(%v.%v <> %v)",
					scope.QuotedTableName(e, modelValue),
					scope.Quote(e, key), scope.AddToVars(e, value)))
			} else {
				sqls = append(sqls, fmt.Sprintf("(%v.%v IS NOT NULL)", scope.QuotedTableName(e, modelValue), scope.Quote(e, key)))
			}
		}
		return strings.Join(sqls, " AND "), nil
	case interface{}:
		v := reflect.ValueOf(value)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() == reflect.Struct {
			var sqls []string
			fds, err := scope.Fields(e, value)
			if err != nil {
				return "", err
			}
			for _, field := range fds {
				if !field.IsBlank {
					sqls = append(sqls, fmt.Sprintf("(%v.%v <> %v)",
						scope.QuotedTableName(e, modelValue),
						scope.Quote(e, field.DBName),
						scope.AddToVars(e, field.Field.Interface())))
				}
			}
			return strings.Join(sqls, " AND "), nil
		}

	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.ValueOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			if bytes, ok := arg.([]byte); ok {
				str = strings.Replace(str, "?", scope.AddToVars(e, bytes), 1)
			} else if values := reflect.ValueOf(arg); values.Len() > 0 {
				var tempMarks []string
				for i := 0; i < values.Len(); i++ {
					tempMarks = append(tempMarks, scope.AddToVars(e, values.Index(i).Interface()))
				}
				str = strings.Replace(str, "?", strings.Join(tempMarks, ","), 1)
			} else {
				str = strings.Replace(str, "?", scope.AddToVars(e, &model.Expr{Q: "NULL"}), 1)
			}
		default:
			if scanner, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = scanner.Value()
			}
			str = strings.Replace(notEqualSQL, "?", scope.AddToVars(e, arg), 1)
		}
	}
	return
}

func SelectSQL(e *engine.Engine, modelValue interface{}) string {
	if len(e.Search.Selects) == 0 {
		if len(e.Search.JoinConditions) > 0 {
			return fmt.Sprintf("%v.*", scope.QuotedTableName(e, modelValue))
		}
		return "*"
	}
	return Select(e, modelValue, e.Search.Selects)
}

//Select builds select query
func Select(e *engine.Engine, modelValue interface{}, clause map[string]interface{}) (str string) {
	switch value := clause["query"].(type) {
	case string:
		str = value
	case []string:
		str = strings.Join(value, ", ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.ValueOf(arg).Kind() {
		case reflect.Slice:
			values := reflect.ValueOf(arg)
			var tempMarks []string
			for i := 0; i < values.Len(); i++ {
				tempMarks = append(tempMarks, scope.AddToVars(e, values.Index(i).Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(tempMarks, ","), 1)
		default:
			if valuer, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = valuer.Value()
			}
			str = strings.Replace(str, "?", scope.AddToVars(e, arg), 1)
		}
	}
	return
}

func JoinSQL(e *engine.Engine, modelValue interface{}) (string, error) {
	var j []string
	for _, clause := range e.Search.JoinConditions {
		sql, err := Where(e, modelValue, clause)
		if err != nil {
			return "", err
		}
		j = append(j, strings.TrimSuffix(strings.TrimPrefix(sql, "("), ")"))
	}
	return strings.Join(j, " ") + " ", nil
}

func OrderSQL(e *engine.Engine, modelValue interface{}) string {
	if len(e.Search.Orders) == 0 || e.Search.IgnoreOrderQuery {
		return ""
	}

	var orders []string
	for _, order := range e.Search.Orders {
		if str, ok := order.(string); ok {
			if regexes.Column.MatchString(str) {
				str = scope.Quote(e, str)
			}
			orders = append(orders, str)
		} else if expr, ok := order.(*model.Expr); ok {
			exp := expr.Q
			for _, arg := range expr.Args {
				exp = strings.Replace(exp, "?", scope.AddToVars(e, arg), 1)
			}
			orders = append(orders, exp)
		}
	}
	return " ORDER BY " + strings.Join(orders, ",")
}

//LimitAndOffsetSQL generates SQL for LIMIT and OFFSET. This relies on the
//implementation defined by the engine.Engine.Dialect.
func LimitAndOffsetSQL(e *engine.Engine) string {
	return e.Dialect.LimitAndOffsetSQL(e.Search.Limit, e.Search.Offset)
}

//GroupSQL generates GROUP BY SQL. This returns an empty string when
//engine.Engine.Search.Group is empty.
func GroupSQL(e *engine.Engine) string {
	if len(e.Search.Group) == 0 {
		return ""
	}
	return " GROUP BY " + e.Search.Group
}

func HavingSQL(e *engine.Engine, modelValue interface{}) (string, error) {
	if len(e.Search.HavingConditions) == 0 {
		return "", errors.New("no having search conditions found")
	}
	var andConditions []string
	for _, clause := range e.Search.HavingConditions {
		sql, err := Where(e, modelValue, clause)
		if err != nil {
			return "", err
		}
		andConditions = append(andConditions, sql)
	}
	combinedSQL := strings.Join(andConditions, " AND ")
	return " HAVING " + combinedSQL, nil
}

// PrepareQuery sets the e.Scope.SQL by generating the whole sql query isnide
// engine.
func PrepareQuery(e *engine.Engine, modelValue interface{}) error {
	sql, err := PrepareQuerySQL(e, modelValue)
	if err != nil {
		return err
	}
	e.Scope.SQL = sql
	return nil
}

func PrepareQuerySQL(e *engine.Engine, modelValue interface{}) (string, error) {
	if e.Search.Raw {
		c, err := CombinedCondition(e, modelValue)
		if err != nil {
			return "", err
		}
		return strings.Replace(c, "$$", "?", -1), nil
	}
	c, err := CombinedCondition(e, modelValue)
	if err != nil {
		return "", err
	}
	return strings.Replace(
		fmt.Sprintf("SELECT %v FROM %v %v",
			SelectSQL(e, modelValue),
			scope.QuotedTableName(e, modelValue),
			c),
		"$$", "?", -1), nil
}

func CombinedCondition(e *engine.Engine, modelValue interface{}) (string, error) {
	joinSql, err := JoinSQL(e, modelValue)
	if err != nil {
		return "", err
	}
	whereSql, err := WhereSQL(e, modelValue)
	if err != nil {
		return "", err
	}
	if e.Search.Raw {
		whereSql = strings.TrimSuffix(strings.TrimPrefix(whereSql, "WHERE ("), ")")
	}
	having, err := HavingSQL(e, modelValue)
	if err != nil {
		return "", err
	}
	return joinSql + whereSql + GroupSQL(e) + having +
		OrderSQL(e, modelValue) + LimitAndOffsetSQL(e), nil
}
