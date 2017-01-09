package scope

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/model"
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

//GetSearchMap return a map of  fiels that are related  as in foreign keys
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
		binVars = append(binVars, `?`)
		conditions = append(conditions, fmt.Sprintf("%v = ?", Quote(e, key)))
		values = append(values, value)
	}

	for _, value := range values {
		values = append(values, value)
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
