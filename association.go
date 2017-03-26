package ngorm

import "github.com/ngorm/ngorm/model"

// Association provides utility functions for dealing with association queries
type Association struct {
	db     *DB
	column string
	field  *model.Field
}
