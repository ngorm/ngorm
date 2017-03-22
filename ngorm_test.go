package ngorm

import (
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/cznic/ql/driver"
	"github.com/ngorm/ngorm/fixture"
)

type Foo struct {
	ID    int
	Stuff string
}

func TestDB_CreateTable(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBCreateTable, &Foo{}, &fixture.User{})
	}
}

func testDBCreateTable(t *testing.T, db *DB) {
	sql, err := db.CreateTableSQL(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.CreateTable1)
	expect = strings.TrimSpace(expect)
	sql.Q = strings.TrimSpace(sql.Q)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
	_, err = db.CreateTable(&Foo{})
	if err != nil {
		t.Error(err)
	}

	_, err = db.CreateTable(&fixture.User{})
	if err != nil {
		t.Error(err)
	}
}

func TestDB_DropTable(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBDropTable)
	}
}

func testDBDropTable(t *testing.T, db *DB) {
	_, err := db.DropTable(&Foo{})
	if err == nil {
		t.Error("expected error")
	}
	_, err = db.CreateTable(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.DropTable(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	if db.dialect.HasTable("foos") {
		t.Error("expected the table to disappear")
	}

	sql, err := db.DropTableSQL(&Foo{}, &fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	sql.Q = strings.TrimSpace(sql.Q)
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.DropTable)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_Automigrate(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBAutomigrate,
			&fixture.User{},
			&fixture.Email{},
			&fixture.Language{},
			&fixture.Company{},
			&fixture.CreditCard{},
			&fixture.Address{},
		)
	}
}

func testDBAutomigrate(t *testing.T, db *DB) {
	sql, err := db.AutomigrateSQL(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}

	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.AutoMigrate)
	sql.Q = strings.TrimSpace(sql.Q)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
	_, err = db.Automigrate(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDB_Create(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBCreate,
			&fixture.User{},
			&fixture.Email{},
			&fixture.Language{},
			&fixture.Company{},
			&fixture.CreditCard{},
			&fixture.Address{},
		)
	}
}
func testDBCreate(t *testing.T, db *DB) {
	_, err := db.Automigrate(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}
	n := time.Now()
	user := fixture.User{Name: "gernest", Age: 29, Birthday: &n}
	_, err = db.CreateSQL(&user)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&user)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDB_SaveSQL(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBSaveSQL)
	}
}

func testDBSaveSQL(t *testing.T, db *DB) {
	sql, err := db.SaveSQL(&Foo{ID: 10, Stuff: "twenty"})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.SaveSQL)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_UpdateSQL(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBUpdateSQL)
	}
}

func testDBUpdateSQL(t *testing.T, db *DB) {
	foo := Foo{ID: 10, Stuff: "twenty"}
	sql, err := db.Model(&foo).UpdateSQL("stuff", "hello")
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.UpdateSQL)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_SingularTable(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBSingularTable)
	}
}

func testDBSingularTable(t *testing.T, db *DB) {
	db.SingularTable(true)
	sql, err := db.CreateSQL(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.SingularTable)
	sql.Q = strings.TrimSpace(sql.Q)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_HasTable(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBHasTable, &Foo{})
	}
}

func testDBHasTable(t *testing.T, db *DB) {
	if db.HasTable("foos") {
		t.Error("expected false")
	}
	if db.HasTable(&Foo{}) {
		t.Error("expected false")
	}
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	if !db.HasTable("foos") {
		t.Error("expected true")
	}
	if !db.HasTable(&Foo{}) {
		t.Error("expected true")
	}
}

func TestDB_FirstSQL(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBFirstSQL)
	}
}

func testDBFirstSQL(t *testing.T, db *DB) {
	// First record order by primary key
	sql, err := db.FirstSQL(&fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.FirstSQL1)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}

	// First record with primary key
	sql, err = db.Begin().FirstSQL(&fixture.User{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	expect = fixture.GetSQL(db.Dialect().GetName(), fixture.FirstSQL2)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_First(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBFirst, &Foo{})
	}
}

func testDBFirst(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.First(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if fu.Stuff != "a" {
		t.Errorf("expected a got  %s", fu.Stuff)
	}
}

func TestDB_LastSQL(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBLastSQL)
	}
}
func testDBLastSQL(t *testing.T, db *DB) {
	// First record order by primary key
	sql, err := db.LastSQL(&fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.LastSQL1)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}

	// First record with primary key
	sql, err = db.clone().LastSQL(&fixture.User{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	expect = fixture.GetSQL(db.Dialect().GetName(), fixture.LastSQL2)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_Last(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBLast, &Foo{})
	}
}

func testDBLast(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.Last(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if fu.Stuff != "d" {
		t.Errorf("expected d got  %s", fu.Stuff)
	}
}

func TestDB_FindSQL(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBFindSQL)
	}
}

func testDBFindSQL(t *testing.T, db *DB) {
	// First record order by primary key
	users := []*fixture.User{}
	sql, err := db.FindSQL(&users)
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.FindSQL1)
	sql.Q = strings.TrimSpace(sql.Q)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}

	sql, err = db.clone().Limit(2).FindSQL(&users)
	if err != nil {
		t.Fatal(err)
	}
	expect = fixture.GetSQL(db.Dialect().GetName(), fixture.FindSQL2)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_Find(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBFind, &Foo{})
	}
}

func testDBFind(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := []Foo{}
	err = db.Find(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if len(fu) != 4 {
		t.Errorf("expected 4 got  %d", len(fu))
	}
}

func TestDB_FirstOrInit(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBFirstOrInit, &Foo{})
	}
}

func testDBFirstOrInit(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	fu := Foo{}
	err = db.Where(Foo{Stuff: "nah"}).Attrs(Foo{ID: 20}).FirstOrInit(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if fu.ID != 20 {
		t.Errorf("expected 20 got %d", fu.ID)
	}
}

func TestDB_Save(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBSave, &Foo{})
	}
}
func testDBSave(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.Begin().First(&fu)
	if err != nil {
		t.Fatal(err)
	}
	fu.Stuff = "updates"
	err = db.Save(&fu)
	if err != nil {
		t.Fatal(err)
	}
	first := Foo{}
	err = db.Begin().First(&first)
	if err != nil {
		t.Fatal(err)
	}
	if first.Stuff != fu.Stuff {
		t.Errorf("expected %s got %s", fu.Stuff, first.Stuff)
	}
}

func TestDB_Update(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBUpdate, &Foo{})
	}
}
func testDBUpdate(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.Begin().First(&fu)
	if err != nil {
		t.Fatal(err)
	}
	up := "stuff"
	err = db.Begin().Model(&fu).Update("stuff", up)
	if err != nil {
		t.Fatal(err)
	}
	first := Foo{}
	err = db.Begin().First(&first)
	if err != nil {
		t.Fatal(err)
	}
	if first.Stuff != up {
		t.Errorf("expected %s got %s", up, first.Stuff)
	}
}

func TestDB_Assign(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBAssign, &fixture.User{})
	}
}
func testDBAssign(t *testing.T, db *DB) {
	_, err := db.Automigrate(&fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	user := fixture.User{}
	err = db.Where(
		fixture.User{Name: "non_existing"}).Assign(
		fixture.User{Age: 20}).FirstOrInit(&user)
	if err != nil {
		t.Fatal(err)
	}
	if user.Age != 20 {
		t.Errorf("expected 40 got %d", user.Age)
	}

}

func TestDB_Pluck(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBPluck, &Foo{})
	}
}
func testDBPluck(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	var stuffs []string
	err = db.Begin().Model(&Foo{}).Pluck("stuff", &stuffs)
	if err != nil {
		t.Fatal(err)
	}
	if len(stuffs) != 4 {
		t.Errorf("expected %d got %d", 4, len(stuffs))
	}
}

func TestDB_Count(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBCount, &Foo{})
	}
}
func testDBCount(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	var stuffs int64
	err = db.Begin().Model(&Foo{}).Count(&stuffs)
	if err != nil {
		t.Fatal(err)
	}
	if stuffs != 4 {
		t.Errorf("expected %d got %d", 4, stuffs)
	}
}

func TestDB_AddIndexSQL(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBAddIndexSQL, &Foo{})
	}
}

func testDBAddIndexSQL(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.AddIndexSQL("_idx_foo_stuff", "stuff")
	if err == nil {
		t.Fatal("expected an error")
	}

	sql, err := db.Model(&Foo{}).AddIndexSQL("_idx_foo_stuff", "stuff")
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.AddIndexSQL)
	sql.Q = strings.TrimSpace(sql.Q)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_AddIndex(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBAddIndex, &Foo{})
	}
}

func testDBAddIndex(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	i := "idx_foo_stuff"
	_, err = db.Model(&Foo{}).AddIndex(i, "stuff")
	if err != nil {
		t.Fatal(err)
	}
	if !db.Dialect().HasIndex("foos", i) {
		t.Error("expected index to be created")
	}
}

func TestDB_DeleteSQL(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBDeleteSQL)
	}
}

func testDBDeleteSQL(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DeleteSQL(&Foo{ID: 10})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.DeleteSQL)
	sql.Q = strings.TrimSpace(sql.Q)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_Delete(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBDelete, &Foo{})
	}
}

func testDBDelete(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	f := Foo{Stuff: "halloween"}
	err = db.Create(&f)
	if err != nil {
		t.Fatal(err)
	}
	if f.ID == 0 {
		t.Fatalf("expected a new record to be created")
	}

	err = db.Delete(&f)
	if err != nil {
		t.Fatal(err)
	}
	fu := Foo{}
	err = db.Model(&Foo{ID: f.ID}).First(&fu)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestDB_AddUniqueIndex(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBAddUniqueIndex, &Foo{})
	}
}

func testDBAddUniqueIndex(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	i := "idx_foo_stuff"
	ndb := db.Model(&Foo{})
	_, err = ndb.AddUniqueIndex(i, "stuff")
	if err != nil {
		t.Fatal(err)
	}
	if !db.Dialect().HasIndex("foos", i) {
		t.Error("expected index to be created")
	}
	q := ndb.e.Scope.SQL
	q = strings.TrimSpace(q)
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.AddUniqueIndex)
	if q != expect {
		t.Errorf("expected %s got %s", expect, q)
	}
}

func TestDB_RemoveIndex(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBRemoveIndex, &Foo{})
	}
}

func testDBRemoveIndex(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	i := "idx_foo_stuff"
	ndb := db.Model(&Foo{})
	_, err = ndb.AddUniqueIndex(i, "stuff")
	if err != nil {
		t.Fatal(err)
	}
	if !db.Dialect().HasIndex("foos", i) {
		t.Error("expected index to be created")
	}
	err = db.Model(&Foo{}).RemoveIndex(i)
	if err != nil {
		t.Fatal(err)
	}
	if db.Dialect().HasIndex("foos", i) {
		t.Error("expected index to be gone")
	}
}

func TestDB_DropColumn(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBDropColumn, &Foo{})
	}
}

func testDBDropColumn(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	col := "stuff"
	_, err = db.Model(&Foo{}).DropColumn(col)
	if err != nil {
		t.Fatal(err)
	}
	if db.Dialect().HasColumn("foos", col) {
		t.Error("expected column to be gone")
	}
}

func TestDB_FirstOrCreate(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBFirstOrCreate, &Foo{})
	}
}

func testDBFirstOrCreate(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	first := Foo{Stuff: "first"}
	err = db.FirstOrCreate(&first)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == 0 {
		t.Error("expected a new record")
	}
	second := Foo{}
	err = db.Begin().Where(Foo{Stuff: first.Stuff}).FirstOrCreate(&second)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Errorf("expected %d got %d", first.ID, second.ID)
	}
}

func TestDB_Preload(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testDBPreload,
			&fixture.User{},
			&fixture.Email{},
			&fixture.Language{},
			&fixture.Company{},
			&fixture.CreditCard{},
			&fixture.Address{},
		)
	}
}

func testDBPreload(t *testing.T, db *DB) {
	_, err := db.Begin().Automigrate(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}
	user1, err := getPreparedUser(db, "user1", "Preload")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Begin().Save(user1)
	if err != nil {
		t.Fatal(err)
	}
	preloadDB := db.Begin().Where("role = ?", "Preload").Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company")
	var user fixture.User
	err = preloadDB.Find(&user)
	if err != nil {
		t.Fatal(err)
	}
	checkUserHasPreloadData(db, user, t)

	user2, err := getPreparedUser(db, "user2", "Preload")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Begin().Save(user2)
	if err != nil {
		t.Fatal(err)
	}

	user3, err := getPreparedUser(db, "user3", "Preload")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Begin().Save(user3)
	if err != nil {
		t.Fatal(err)
	}

	var users []fixture.User
	preloadDB = db.Begin().Where("role = ?", "Preload").Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company")
	err = preloadDB.Find(&users)
	if err != nil {
		t.Fatal(err)
	}

	for _, user := range users {
		checkUserHasPreloadData(db, user, t)
	}

	var users2 []*fixture.User
	preloadDB = db.Begin().Where("role = ?", "Preload").Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company")
	err = preloadDB.Find(&users2)
	if err != nil {
		t.Fatal(err)
	}

	for _, user := range users2 {
		checkUserHasPreloadData(db, *user, t)
	}

	var users3 []*fixture.User
	preloadDB = db.Begin().Where("role = ?", "Preload").Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company")
	err = preloadDB.Preload("Emails", "email = ?", user3.Emails[0].Email).Find(&users3)
	if err != nil {
		t.Fatal(err)
	}

	for _, user := range users3 {
		if user.Name == user3.Name {
			if len(user.Emails) != 1 {
				t.Errorf("should only preload one emails for user3 when with condition")
			}
		} else if len(user.Emails) != 0 {
			t.Errorf("should not preload any emails for other users when with condition")
		} else if user.Emails == nil {
			t.Errorf("should return an empty slice to indicate zero results")
		}
	}
}

func checkUserHasPreloadData(db *DB, user fixture.User, t *testing.T) {
	u, err := getPreparedUser(db, user.Name, "Preload")
	if err != nil {
		t.Fatal(err)
	}
	if user.BillingAddress.Address1 != u.BillingAddress.Address1 {
		t.Errorf("BillingAddress: expected %s got %s",
			u.BillingAddress.Address1, user.BillingAddress.Address1)
	}

	if user.ShippingAddress.Address1 != u.ShippingAddress.Address1 {
		t.Errorf("ShippingAddress: expected %s got %s",
			u.ShippingAddress.Address1, user.ShippingAddress.Address1)
	}

	if user.CreditCard.Number != u.CreditCard.Number {
		t.Errorf("CreditCard: expected %s got %s",
			u.CreditCard.Number, user.CreditCard.Number)
	}

	if user.Company.Name != u.Company.Name {
		t.Errorf(" Company: expected %s got %s", u.Company.Name, user.Company.Name)
	}

	if len(user.Emails) != len(u.Emails) {
		t.Errorf("Emails: expected %d emails got %d", len(u.Emails), len(user.Emails))

	} else {
		var found int
		for _, e1 := range u.Emails {
			for _, e2 := range user.Emails {
				if e1.Email == e2.Email {
					found++
					break
				}
			}
		}
		if found != len(u.Emails) {
			t.Errorf("expected %d emails got %d", len(u.Emails), found)
		}
	}
}

func getPreparedUser(db *DB, name string, role string) (*fixture.User, error) {
	var company fixture.Company
	err := db.Begin().Where(fixture.Company{Name: role}).FirstOrCreate(&company)
	if err != nil {
		return nil, err
	}
	return &fixture.User{
		Name:            name,
		Age:             20,
		Role:            fixture.Role{Name: role},
		BillingAddress:  fixture.Address{Address1: fmt.Sprintf("Billing Address %v", name)},
		ShippingAddress: fixture.Address{Address1: fmt.Sprintf("Shipping Address %v", name)},
		CreditCard:      fixture.CreditCard{Number: fmt.Sprintf("123456%v", name)},
		Emails: []fixture.Email{
			{Email: fmt.Sprintf("user_%v@example1.com", name)}, {Email: fmt.Sprintf("user_%v@example2.com", name)},
		},
		Company: company,
		Languages: []fixture.Language{
			{Name: fmt.Sprintf("lang_1_%v", name)},
			{Name: fmt.Sprintf("lang_2_%v", name)},
		},
	}, nil
}

func TestRelationship_Belongs_To(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, belongsTo,
			&bUser{}, &bAddr{},
		)
	}
}

type bUser struct {
	ID     int64
	Addr   bAddr
	AddrID int
	Ship   bAddr
	ShipID int64
}

type bAddr struct {
	ID   int64
	Name string
}

func belongsTo(t *testing.T, db *DB) {
	_, err := db.Begin().Automigrate(
		&bUser{}, &bAddr{},
	)
	if err != nil {
		t.Fatal(err)
	}
	n := "some name"
	u := &bUser{
		Addr: bAddr{Name: n},
		Ship: bAddr{Name: n},
	}
	err = db.Begin().Create(u)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 {
		t.Error("expected user to be created")
	}
	if u.AddrID == 0 {
		t.Error("expected Addr to be created")
	}
	if u.ShipID == 0 {
		t.Error("expected Ship to be created")
	}
}
