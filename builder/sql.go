package builder

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/model"
	"github.com/gernest/ngorm/regexes"
	"github.com/gernest/ngorm/scope"
)

//Where bilds the Ehere Condition the sql where condition.
func Where(e *engine.Engine, modelValue interface{}, clause map[string]interface{}) (str string) {
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
		str = fmt.Sprintf("(%v.%v IN (?))", scope.QuotedTableName(e, modelValue),
			scope.Quote(e, scope.PrimaryKey(e, modelValue)))
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
		return strings.Join(sqls, " AND ")
	case interface{}:
		var sqls []string
		for _, field := range scope.Fields(e, modelValue) {
			if !field.IsIgnored && !field.IsBlank {
				sqls = append(sqls, fmt.Sprintf("(%v.%v = %v)",
					scope.QuotedTableName(e, modelValue),
					scope.Quote(e, field.DBName),
					scope.AddToVars(e, field.Field.Interface())))
			}
		}
		return strings.Join(sqls, " AND ")
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

func PrimaryCondition(e *engine.Engine, modelValue, value interface{}) string {
	return fmt.Sprintf("(%v.%v = %v)", scope.QuotedTableName(e, modelValue),
		scope.Quote(e, scope.PrimaryKey(e, modelValue)), value)
}

func WhereSQL(e *engine.Engine, modelValue interface{}) (sql string) {
	var (
		quotedTableName                                = scope.QuotedTableName(e, modelValue)
		primaryConditions, andConditions, orConditions []string
	)

	if !e.Search.Unscoped && scope.HasColumn(e, modelValue, "deleted_at") {
		sql := fmt.Sprintf("%v.deleted_at IS NULL", quotedTableName)
		primaryConditions = append(primaryConditions, sql)
	}

	f := scope.PrimaryField(e, modelValue)
	if !(f == nil || f.IsBlank) {
		for _, field := range scope.PrimaryFields(e, modelValue) {
			sql := fmt.Sprintf("%v.%v = %v", quotedTableName,
				scope.Quote(e, field.DBName), scope.AddToVars(e, field.Field.Interface()))
			primaryConditions = append(primaryConditions, sql)
		}
	}

	for _, clause := range e.Search.WhereConditions {
		if sql := Where(e, modelValue, clause); sql != "" {
			andConditions = append(andConditions, sql)
		}
	}

	for _, clause := range e.Search.OrConditions {
		if sql := Where(e, modelValue, clause); sql != "" {
			orConditions = append(orConditions, sql)
		}
	}

	for _, clause := range e.Search.NotConditions {
		if sql := Not(e, modelValue, clause); sql != "" {
			andConditions = append(andConditions, sql)
		}
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

func Not(e *engine.Engine, modelValue interface{}, clause map[string]interface{}) (str string) {
	var notEqualSQL string
	var primaryKey = scope.PrimaryKey(e, modelValue)
	switch value := clause["query"].(type) {
	case string:
		if regexes.IsNumber.MatchString(value) {
			id, _ := strconv.Atoi(value)
			return fmt.Sprintf("(%v <> %v)", scope.Quote(e, primaryKey), id)
		} else if regexes.Comparison.MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			notEqualSQL = fmt.Sprintf("NOT (%v)", value)
		} else {
			str = fmt.Sprintf("(%v.%v NOT IN (?))", scope.QuotedTableName(e, modelValue), scope.Quote(e, value))
			notEqualSQL = fmt.Sprintf("(%v.%v <> ?)", scope.QuotedTableName(e, modelValue), scope.Quote(e, value))
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, sql.NullInt64:
		return fmt.Sprintf("(%v.%v <> %v)", scope.QuotedTableName(e, modelValue), scope.Quote(e, primaryKey), value)
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []string:
		if reflect.ValueOf(value).Len() > 0 {
			str = fmt.Sprintf("(%v.%v NOT IN (?))", scope.QuotedTableName(e, modelValue), scope.Quote(e, primaryKey))
			clause["args"] = []interface{}{value}
		}
		return ""
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
		return strings.Join(sqls, " AND ")
	case interface{}:
		var sqls []string
		//var newScope = scope.New(value)
		for _, field := range scope.Fields(e, value) {
			if !field.IsBlank {
				sqls = append(sqls, fmt.Sprintf("(%v.%v <> %v)",
					scope.QuotedTableName(e, modelValue),
					scope.Quote(e, field.DBName),
					scope.AddToVars(e, field.Field.Interface())))
			}
		}
		return strings.Join(sqls, " AND ")
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

func JoinSQL(e *engine.Engine, modelValue interface{}) string {
	var j []string
	for _, clause := range e.Search.JoinConditions {
		if sql := Where(e, modelValue, clause); sql != "" {
			j = append(j, strings.TrimSuffix(strings.TrimPrefix(sql, "("), ")"))
		}
	}
	return strings.Join(j, " ") + " "
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

func LimitAndOffsetSQL(e *engine.Engine, modelValue interface{}) string {
	return e.Dialect.LimitAndOffsetSQL(e.Search.Limit, e.Search.Offset)
}

func GroupSQL(e *engine.Engine, modelValue interface{}) string {
	if len(e.Search.Group) == 0 {
		return ""
	}
	return " GROUP BY " + e.Search.Group
}

func HavingSQL(e *engine.Engine, modelValue interface{}) string {
	if len(e.Search.HavingConditions) == 0 {
		return ""
	}

	var andConditions []string
	for _, clause := range e.Search.HavingConditions {
		if sql := Where(e, modelValue, clause); sql != "" {
			andConditions = append(andConditions, sql)
		}
	}
	combinedSQL := strings.Join(andConditions, " AND ")
	if len(combinedSQL) == 0 {
		return ""
	}

	return " HAVING " + combinedSQL
}

func PrepareQuerySQL(e *engine.Engine, modelValue interface{}) {
	if e.Search.Raw {
		e.Scope.SQL = strings.Replace(
			CombinedCondition(e, modelValue),
			"$$", "?", -1)
	} else {
		e.Scope.SQL = strings.Replace(
			fmt.Sprintf("SELECT %v FROM %v %v",
				SelectSQL(e, modelValue),
				scope.QuotedTableName(e, modelValue),
				CombinedCondition(e, modelValue)),
			"$$", "?", -1)
	}
}

func CombinedCondition(e *engine.Engine, modelValue interface{}) string {
	joinSql := JoinSQL(e, modelValue)
	whereSql := WhereSQL(e, modelValue)
	if e.Search.Raw {
		whereSql = strings.TrimSuffix(strings.TrimPrefix(whereSql, "WHERE ("), ")")
	}
	return joinSql + whereSql + GroupSQL(e, modelValue) +
		HavingSQL(e, modelValue) +
		OrderSQL(e, modelValue) + LimitAndOffsetSQL(e, modelValue)
}
