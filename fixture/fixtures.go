//Package fixture contains all stuctures neesessary for consinstent testing on a
//wide range of SQL database dialects.
package fixture

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"time"

	"github.com/ngorm/ngorm/engine"
	"github.com/ngorm/ngorm/model"
)

//CalculateField fixture
type CalculateField struct {
	model.Model
	Name     string
	Children []CalculateFieldChild
	Category CalculateFieldCategory
	EmbeddedField
}

//EmbeddedField fixture
type EmbeddedField struct {
	EmbeddedName string `sql:"NOT NULL;DEFAULT:'hello'"`
}

//CalculateFieldChild fixture
type CalculateFieldChild struct {
	model.Model
	CalculateFieldID uint
	Name             string
}

//CalculateFieldCategory fixture
type CalculateFieldCategory struct {
	model.Model
	CalculateFieldID uint
	Name             string
}

//CustomizeColumn fixture
type CustomizeColumn struct {
	ID   int64      `gorm:"column:mapped_id; primary_key:yes"`
	Name string     `gorm:"column:mapped_name"`
	Date *time.Time `gorm:"column:mapped_time"`
}

//CustomColumnAndIgnoredFieldClash fixture
// Make sure an ignored field does not interfere with another field's custom
// column name that matches the ignored field.
type CustomColumnAndIgnoredFieldClash struct {
	Body    string `sql:"-"`
	RawBody string `gorm:"column:body"`
}

//Cat fixture
type Cat struct {
	ID   int
	Name string
	Toy  Toy `gorm:"polymorphic:Owner;"`
}

//Dog fixture
type Dog struct {
	ID   int
	Name string
	Toys []Toy `gorm:"polymorphic:Owner;"`
}

//Hamster fixture
type Hamster struct {
	ID           int
	Name         string
	PreferredToy Toy `gorm:"polymorphic:Owner;polymorphic_value:hamster_preferred"`
	OtherToy     Toy `gorm:"polymorphic:Owner;polymorphic_value:hamster_other"`
}

//Toy fixture
type Toy struct {
	ID        int
	Name      string
	OwnerID   int
	OwnerType string
}

//User fixture
type User struct {
	ID                int64
	Age               int64
	UserNum           Num
	Name              string `sql:"size:255"`
	Email             string
	Birthday          *time.Time    // Time
	CreatedAt         time.Time     // CreatedAt: Time of record is created, will be insert automatically
	UpdatedAt         time.Time     // UpdatedAt: Time of record is updated, will be updated automatically
	Emails            []Email       // Embedded structs
	BillingAddress    Address       // Embedded struct
	BillingAddressID  sql.NullInt64 // Embedded struct's foreign key
	ShippingAddress   Address       // Embedded struct
	ShippingAddressID int64         // Embedded struct's foreign key
	CreditCard        CreditCard
	Latitude          float64
	Languages         []Language `gorm:"many2many:user_languages;"`
	CompanyID         *int
	Company           Company
	Role
	PasswordHash      []byte
	Sequence          uint                  `gorm:"AUTO_INCREMENT"`
	IgnoreMe          int64                 `sql:"-"`
	IgnoreStringSlice []string              `sql:"-"`
	Ignored           struct{ Name string } `sql:"-"`
	IgnoredPointer    *User                 `sql:"-"`
}

//NotSoLongTableName fixture
type NotSoLongTableName struct {
	ID                int64
	ReallyLongThingID int64
	ReallyLongThing   ReallyLongTableNameToTestMySQLNameLengthLimit
}

//ReallyLongTableNameToTestMySQLNameLengthLimit fixture
type ReallyLongTableNameToTestMySQLNameLengthLimit struct {
	ID int64
}

//ReallyLongThingThatReferencesShort fixture
type ReallyLongThingThatReferencesShort struct {
	ID      int64
	ShortID int64
	Short   Short
}

//Short fixture
type Short struct {
	ID int64
}

//CreditCard fixture
type CreditCard struct {
	ID        int8
	Number    string
	UserID    sql.NullInt64
	CreatedAt time.Time `sql:"not null"`
	UpdatedAt time.Time
	DeletedAt *time.Time
}

//Email fixture
type Email struct {
	ID        int16
	UserID    int
	Email     string `sql:"type:varchar(100);"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

//Address fixture
type Address struct {
	ID        int
	Address1  string
	Address2  string
	Post      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

//Language fixture
type Language struct {
	model.Model
	Name  string
	Users []User `gorm:"many2many:user_languages;"`
}

//Product fixture
type Product struct {
	ID                    int64
	Code                  string
	Price                 int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
	AfterFindCallTimes    int64
	BeforeCreateCallTimes int64
	AfterCreateCallTimes  int64
	BeforeUpdateCallTimes int64
	AfterUpdateCallTimes  int64
	BeforeSaveCallTimes   int64
	AfterSaveCallTimes    int64
	BeforeDeleteCallTimes int64
	AfterDeleteCallTimes  int64
}

//Company fixture
type Company struct {
	ID    int64
	Name  string
	Owner *User `sql:"-"`
}

//Role fixture
type Role struct {
	Name string `gorm:"size:256"`
}

//Scan implements sql.Scanner
func (role *Role) Scan(value interface{}) error {
	if b, ok := value.([]uint8); ok {
		role.Name = string(b)
	} else {
		role.Name = value.(string)
	}
	return nil
}

//Value implements sql driver.Valuer
func (role Role) Value() (driver.Value, error) {
	return role.Name, nil
}

//IsAdmin return true if the role is admin
func (role Role) IsAdmin() bool {
	return role.Name == "admin"
}

//Num custom int64 type
type Num int64

//Scan implements sql.Scanner
func (i *Num) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
	case int64:
		*i = Num(s)
	default:
		return errors.New("Cannot scan NamedInt from " + reflect.ValueOf(src).String())
	}
	return nil
}

//Animal fixture
type Animal struct {
	Counter    uint64    `gorm:"primary_key:yes"`
	Name       string    `sql:"DEFAULT:'galeone'"`
	From       string    //test reserved sql keyword as field name
	Age        time.Time `sql:"DEFAULT:current_timestamp"`
	unexported string    // unexported value
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

//JoinTable fixture
type JoinTable struct {
	From uint64
	To   uint64
	Time time.Time `sql:"default: null"`
}

//Post fixture
type Post struct {
	ID             int64
	CategoryID     sql.NullInt64
	MainCategoryID int64
	Title          string
	Body           string
	Comments       []*Comment
	Category       Category
	MainCategory   Category
}

//Category fixture
type Category struct {
	model.Model
	Name string

	Categories []Category
	CategoryID *uint
}

//Comment fixture
type Comment struct {
	model.Model
	PostID  int64
	Content string
	Post    Post
}

//NullValue fixture
type NullValue struct {
	ID      int64
	Name    sql.NullString  `sql:"not null"`
	Gender  *sql.NullString `sql:"not null"`
	Age     sql.NullInt64
	Male    sql.NullBool
	Height  sql.NullFloat64
	AddedAt NullTime
}

//NullTime fixture
type NullTime struct {
	Time  time.Time
	Valid bool
}

//Scan implents sql.Scanner
func (nt *NullTime) Scan(value interface{}) error {
	if value == nil {
		nt.Valid = false
		return nil
	}
	nt.Time, nt.Valid = value.(time.Time), true
	return nil
}

//Value implements driver.valuer
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

//TestEngine returns an *engine.Engine instance suitable for testing
func TestEngine() *engine.Engine {
	return &engine.Engine{
		Search:    &model.Search{},
		Scope:     &model.Scope{},
		StructMap: model.NewStructsMap(),
	}
}
