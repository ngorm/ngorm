package scope

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ngorm/ngorm/engine"
	"github.com/ngorm/ngorm/model"
	"github.com/ngorm/ngorm/search"
	"github.com/ngorm/ngorm/util"
)

// SetupJoinTable  initialize a default join table handler
func SetupJoinTable(s *model.JoinTableHandler, relationship *model.Relationship,
	tableName string, source reflect.Type, destination reflect.Type) {
	s.TableName = tableName
	s.Source = model.JoinTableSource{ModelType: source}
	for idx, dbName := range relationship.ForeignFieldNames {
		s.Source.ForeignKeys = append(s.Source.ForeignKeys, model.JoinTableForeignKey{
			DBName:            relationship.ForeignDBNames[idx],
			AssociationDBName: dbName,
		})
	}

	s.Destination = model.JoinTableSource{ModelType: destination}
	for idx, dbName := range relationship.AssociationForeignFieldNames {
		s.Destination.ForeignKeys = append(s.Destination.ForeignKeys, model.JoinTableForeignKey{
			DBName:            relationship.AssociationForeignDBNames[idx],
			AssociationDBName: dbName,
		})
	}
}

//GetSearchMap return a map of  fields that are related  as in foreign keys
//between the source model and destination model.
func GetSearchMap(e *engine.Engine, s *model.JoinTableHandler, sources ...interface{}) map[string]interface{} {
	values := map[string]interface{}{}

	for _, source := range sources {
		m, err := GetModelStruct(e, source)
		if err != nil {
			//TODO return?
		}
		modelType := m.ModelType

		if s.Source.ModelType == modelType {
			for _, foreignKey := range s.Source.ForeignKeys {
				field, err := FieldByName(e, source, foreignKey.AssociationDBName)
				if err != nil {
					//TODO return?
				} else {
					values[foreignKey.DBName] = field.Field.Interface()
				}
			}
		} else if s.Destination.ModelType == modelType {
			for _, foreignKey := range s.Destination.ForeignKeys {
				field, err := FieldByName(e, source, foreignKey.AssociationDBName)
				if err != nil {
					//TODO return?
				} else {
					values[foreignKey.DBName] = field.Field.Interface()
				}
			}
		}
	}
	return values
}

// AddJoinRelation  create relationship in join table for source and destination
func AddJoinRelation(table string, s *model.JoinTableHandler,
	e *engine.Engine, source interface{},
	destination interface{}) (*model.Expr, error) {
	searchMap := GetSearchMap(e, s, source, destination)

	var assignColumns, binVars, conditions []string
	var values []interface{}
	for key, value := range searchMap {
		assignColumns = append(assignColumns, Quote(e, key))
		values = append(values, value)
		bind := e.Dialect.BindVar(len(values))
		binVars = append(binVars, bind)
		conditions = append(conditions, fmt.Sprintf("%v = %s", Quote(e, key), bind))
	}

	quotedTable := Quote(e, table)
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) SELECT %v %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v)",
		quotedTable,
		strings.Join(assignColumns, ","),
		strings.Join(binVars, ","),
		e.Dialect.SelectFromDummyTable(),
		quotedTable,
		strings.Join(conditions, " AND "),
	)
	return &model.Expr{Q: sql, Args: values}, nil
}

// JoinWith query with `Join` conditions
func JoinWith(handler *model.JoinTableHandler, ne *engine.Engine, source interface{}) error {
	ne.Scope.Value = source
	tableName := handler.TableName
	quotedTableName := Quote(ne, tableName)
	var joinConditions []string
	var values []interface{}
	m, err := GetModelStruct(ne, source)
	if err != nil {
		return err
	}
	if handler.Source.ModelType == m.ModelType {
		d := reflect.New(handler.Destination.ModelType).Interface()
		destinationTableName := QuotedTableName(ne, d)
		for _, foreignKey := range handler.Destination.ForeignKeys {
			joinConditions = append(joinConditions, fmt.Sprintf("%v.%v = %v.%v",
				quotedTableName,
				Quote(ne, foreignKey.DBName),
				destinationTableName,
				Quote(ne, foreignKey.AssociationDBName)))
		}

		var foreignDBNames []string
		var foreignFieldNames []string

		for _, foreignKey := range handler.Source.ForeignKeys {
			foreignDBNames = append(foreignDBNames, foreignKey.DBName)
			if field, ok := FieldByName(ne, source, foreignKey.AssociationDBName); ok != nil {
				foreignFieldNames = append(foreignFieldNames, field.Name)
			}
		}

		foreignFieldValues := util.ColumnAsArray(foreignFieldNames, ne.Scope.Value)

		var condString string
		if len(foreignFieldValues) > 0 {
			var quotedForeignDBNames []string
			for _, dbName := range foreignDBNames {
				quotedForeignDBNames = append(quotedForeignDBNames, tableName+"."+dbName)
			}

			condString = fmt.Sprintf("%v IN (%v)",
				ToQueryCondition(ne, quotedForeignDBNames),
				util.ToQueryMarks(foreignFieldValues))

			keys := util.ColumnAsArray(foreignFieldNames, ne.Scope.Value)
			values = append(values, util.ToQueryValues(keys))
		} else {
			condString = fmt.Sprintf("1 <> 1")
		}

		search.Join(ne,
			fmt.Sprintf("INNER JOIN %v ON %v",
				quotedTableName,
				strings.Join(joinConditions, " AND ")))
		search.Where(ne, condString, util.ToQueryValues(foreignFieldValues)...)
		return nil
	}
	return errors.New("wrong source type for join table handler")
}
