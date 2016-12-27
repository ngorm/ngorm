package fixture

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"time"

	"github.com/gernest/ngorm/model"
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

type Cat struct {
	Id   int
	Name string
	Toy  Toy `gorm:"polymorphic:Owner;"`
}

type Dog struct {
	Id   int
	Name string
	Toys []Toy `gorm:"polymorphic:Owner;"`
}

type Hamster struct {
	Id           int
	Name         string
	PreferredToy Toy `gorm:"polymorphic:Owner;polymorphic_value:hamster_preferred"`
	OtherToy     Toy `gorm:"polymorphic:Owner;polymorphic_value:hamster_other"`
}

type Toy struct {
	Id        int
	Name      string
	OwnerId   int
	OwnerType string
}

type User struct {
	Id                int64
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
	ShippingAddressId int64         // Embedded struct's foreign key
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

type NotSoLongTableName struct {
	Id                int64
	ReallyLongThingID int64
	ReallyLongThing   ReallyLongTableNameToTestMySQLNameLengthLimit
}

type ReallyLongTableNameToTestMySQLNameLengthLimit struct {
	Id int64
}

type ReallyLongThingThatReferencesShort struct {
	Id      int64
	ShortID int64
	Short   Short
}

type Short struct {
	Id int64
}

type CreditCard struct {
	ID        int8
	Number    string
	UserId    sql.NullInt64
	CreatedAt time.Time `sql:"not null"`
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type Email struct {
	Id        int16
	UserId    int
	Email     string `sql:"type:varchar(100);"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Address struct {
	ID        int
	Address1  string
	Address2  string
	Post      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type Language struct {
	model.Model
	Name  string
	Users []User `gorm:"many2many:user_languages;"`
}

type Product struct {
	Id                    int64
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

type Company struct {
	Id    int64
	Name  string
	Owner *User `sql:"-"`
}

type Role struct {
	Name string `gorm:"size:256"`
}

func (role *Role) Scan(value interface{}) error {
	if b, ok := value.([]uint8); ok {
		role.Name = string(b)
	} else {
		role.Name = value.(string)
	}
	return nil
}

func (role Role) Value() (driver.Value, error) {
	return role.Name, nil
}

func (role Role) IsAdmin() bool {
	return role.Name == "admin"
}

type Num int64

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

type Animal struct {
	Counter    uint64    `gorm:"primary_key:yes"`
	Name       string    `sql:"DEFAULT:'galeone'"`
	From       string    //test reserved sql keyword as field name
	Age        time.Time `sql:"DEFAULT:current_timestamp"`
	unexported string    // unexported value
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type JoinTable struct {
	From uint64
	To   uint64
	Time time.Time `sql:"default: null"`
}

type Post struct {
	Id             int64
	CategoryId     sql.NullInt64
	MainCategoryId int64
	Title          string
	Body           string
	Comments       []*Comment
	Category       Category
	MainCategory   Category
}

type Category struct {
	model.Model
	Name string

	Categories []Category
	CategoryID *uint
}

type Comment struct {
	model.Model
	PostId  int64
	Content string
	Post    Post
}

// Scanner
type NullValue struct {
	Id      int64
	Name    sql.NullString  `sql:"not null"`
	Gender  *sql.NullString `sql:"not null"`
	Age     sql.NullInt64
	Male    sql.NullBool
	Height  sql.NullFloat64
	AddedAt NullTime
}

type NullTime struct {
	Time  time.Time
	Valid bool
}

func (nt *NullTime) Scan(value interface{}) error {
	if value == nil {
		nt.Valid = false
		return nil
	}
	nt.Time, nt.Valid = value.(time.Time), true
	return nil
}

func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}
