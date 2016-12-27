package fixture

import (
	"time"

	"github.com/gernest/gorm/model"
)

type CalculateField struct {
	model.Model
	Name     string
	Children []CalculateFieldChild
	Category CalculateFieldCategory
	EmbeddedField
}

type EmbeddedField struct {
	EmbeddedName string `sql:"NOT NULL;DEFAULT:'hello'"`
}

type CalculateFieldChild struct {
	model.Model
	CalculateFieldID uint
	Name             string
}

type CalculateFieldCategory struct {
	model.Model
	CalculateFieldID uint
	Name             string
}

type CustomizeColumn struct {
	ID   int64      `gorm:"column:mapped_id; primary_key:yes"`
	Name string     `gorm:"column:mapped_name"`
	Date *time.Time `gorm:"column:mapped_time"`
}

// Make sure an ignored field does not interfere with another field's custom
// column name that matches the ignored field.
type CustomColumnAndIgnoredFieldClash struct {
	Body    string `sql:"-"`
	RawBody string `gorm:"column:body"`
}
