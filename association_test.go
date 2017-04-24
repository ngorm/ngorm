package ngorm

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ngorm/ngorm/fixture"
	"github.com/ngorm/ngorm/model"
	"github.com/ngorm/ngorm/scope"
)

type Cat struct {
	ID   int64
	Name string
	Toy  Toy `gorm:"polymorphic:Owner;"`
}

type Dog struct {
	ID   int64
	Name string
	Toys []Toy `gorm:"polymorphic:Owner;"`
}

type Hamster struct {
	ID           int
	Name         string
	PreferredToy Toy `gorm:"polymorphic:Owner;polymorphic_value:hamster_preferred"`
	OtherToy     Toy `gorm:"polymorphic:Owner;polymorphic_value:hamster_other"`
}

type Toy struct {
	ID        int
	Name      string
	OwnerID   int64
	OwnerType string
}

var compareToys = func(toys []Toy, contents []string) bool {
	var toyContents []string
	for _, toy := range toys {
		toyContents = append(toyContents, toy.Name)
	}
	sort.Strings(toyContents)
	sort.Strings(contents)
	return reflect.DeepEqual(toyContents, contents)
}

func TestPolymorphic(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testPolymorphic,
			&Cat{}, &Dog{}, &Hamster{}, &Toy{})

	}
}

func testPolymorphic(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Cat{}, &Dog{}, &Hamster{}, &Toy{})
	if err != nil {
		t.Fatal(err)
	}

	cat := Cat{Name: "Mr. Bigglesworth", Toy: Toy{Name: "cat toy"}}
	dog := Dog{Name: "Pluto", Toys: []Toy{{Name: "dog toy 1"}, {Name: "dog toy 2"}}}
	db.Begin().Save(&cat)
	db.Begin().Save(&dog)

	a, err := db.Begin().Model(&cat).Association("Toy")
	if err != nil {
		t.Fatal(err)
	}
	count, err := a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 got %d", count)
	}

	a, err = db.Begin().Model(&dog).Association("Toys")
	if err != nil {
		t.Fatal(err)
	}
	count, err = a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 got %d", count)
	}

	// Query
	var catToys []Toy
	err = db.Begin().Model(&cat).Related(&catToys, "Toy")
	if err != nil {
		t.Error(err)
	}
	if len(catToys) != 1 {
		t.Errorf("expected 1 got %d", len(catToys))
	}
	if catToys[0].Name != cat.Toy.Name {
		t.Errorf("expected %s got %s", cat.Toy.Name, catToys[0].Name)
	}

	var dogToys []Toy
	err = db.Begin().Model(&dog).Related(&dogToys, "Toys")
	if err != nil {
		t.Error(err)
	}
	if len(dogToys) != len(dog.Toys) {
		t.Errorf("expected %d got %d", len(dog.Toys), len(dogToys))
	}

	var catToy Toy
	a, err = db.Begin().Model(&cat).Association("Toy")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&catToy)
	if err != nil {
		t.Error(err)
	}
	if catToy.Name != cat.Toy.Name {
		t.Errorf("expected %s got %s", cat.Toy.Name, catToy.Name)
	}

	a, err = db.Begin().Model(&cat).Association("Toy")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Append(&Toy{
		Name: "dog toy 3",
	})
	if err != nil {
		t.Error(err)
	}
	a, err = db.Begin().Model(&dog).Association("Toys")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Append(&Toy{
		Name: "dog toy 3",
	})
	if err != nil {
		t.Fatal(err)
	}
	count, err = a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("expected 3 got %d", count)
	}
}

func TestNamedPolymorphic(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testNamedPolymorphic,
			&Cat{}, &Dog{}, &Hamster{}, &Toy{})

	}
}

func testNamedPolymorphic(t *testing.T, db *DB) {
	_, err := db.Automigrate(&Cat{}, &Dog{}, &Hamster{}, &Toy{})
	if err != nil {
		t.Fatal(err)
	}

	hamster := Hamster{Name: "Mr. Hammond", PreferredToy: Toy{Name: "bike"}, OtherToy: Toy{Name: "treadmill"}}
	err = db.Save(&hamster)
	if err != nil {
		t.Fatal(err)
	}
	hamster2 := Hamster{}
	db.Begin().Preload("PreferredToy").Preload("OtherToy").Find(&hamster2, hamster.ID)
	if hamster2.PreferredToy.ID != hamster.PreferredToy.ID || hamster2.PreferredToy.Name != hamster.PreferredToy.Name {
		t.Errorf("Hamster's preferred toy couldn't be preloaded")
	}
	if hamster2.OtherToy.ID != hamster.OtherToy.ID || hamster2.OtherToy.Name != hamster.OtherToy.Name {
		t.Errorf("Hamster's other toy couldn't be preloaded")
	}

	// clear to omit Toy.Id in count
	hamster2.PreferredToy = Toy{}
	hamster2.OtherToy = Toy{}

	a, err := db.Begin().Model(&hamster2).Association("PreferredToy")
	if err != nil {
		t.Fatal(err)
	}
	count, err := a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected %d got %d", 1, count)
	}

	a, err = db.Begin().Model(&hamster2).Association("OtherToy")
	if err != nil {
		t.Fatal(err)
	}
	count, err = a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected %d got %d", 1, count)
	}

	// Query
	var hamsterToys []Toy

	err = db.Begin().Model(&hamster).Related(&hamsterToys, "PreferredToy")
	if err != nil {
		t.Fatal(err)
	}
	if len(hamsterToys) != 1 {
		t.Errorf("expected %d got %d", 1, len(hamsterToys))
	}
	if hamsterToys[0].Name != hamster.PreferredToy.Name {
		t.Errorf("expected %s got %s", hamster.PreferredToy.Name, hamsterToys[0].Name)
	}

	err = db.Begin().Model(&hamster).Related(&hamsterToys, "OtherToy")
	if err != nil {
		t.Fatal(err)
	}
	if len(hamsterToys) != 1 {
		t.Errorf("expected %d got %d", 1, len(hamsterToys))
	}
	if hamsterToys[0].Name != hamster.OtherToy.Name {
		t.Errorf("expected %s got %s", hamster.OtherToy.Name, hamsterToys[0].Name)
	}

	hamsterToy := Toy{}
	a, err = db.Begin().Model(&hamster).Association("PreferredToy")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&hamsterToy)
	if err != nil {
		t.Fatal(err)
	}

	if hamsterToy.Name != hamster.PreferredToy.Name {
		t.Errorf("Should find has one polymorphic association")
	}

	hamsterToy = Toy{}
	a, err = db.Begin().Model(&hamster).Association("OtherToy")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&hamsterToy)
	if err != nil {
		t.Fatal(err)
	}

	if hamsterToy.Name != hamster.OtherToy.Name {
		t.Errorf("Should find has one polymorphic association")
	}

	// Append
	a, err = db.Begin().Model(&hamster).Association("PreferredToy")
	if err != nil {
		t.Fatal(err)
	}

	err = a.Append(&Toy{
		Name: "bike 2",
	})
	if err != nil {
		t.Fatal(err)
	}

	a, err = db.Begin().Model(&hamster).Association("OtherToy")
	if err != nil {
		t.Fatal(err)
	}

	err = a.Append(&Toy{
		Name: "treadmill 2",
	})
	if err != nil {
		t.Fatal(err)
	}

	hamsterToy = Toy{}
	a, err = db.Begin().Model(&hamster).Association("PreferredToy")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&hamsterToy)
	if err != nil {
		t.Fatal(err)
	}
	if hamsterToy.Name != "bike 2" {
		t.Errorf("Should update has one polymorphic association with Append")
	}

	hamsterToy = Toy{}
	a, err = db.Begin().Model(&hamster).Association("OtherToy")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&hamsterToy)
	if err != nil {
		t.Fatal(err)
	}

	if hamsterToy.Name != "treadmill 2" {
		t.Errorf("Should update has one polymorphic association with Append")
	}

	a, err = db.Begin().Model(&hamster).Association("OtherToy")
	if err != nil {
		t.Fatal(err)
	}
	count, err = a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected %d got %d", 1, count)
	}

	a, err = db.Begin().Model(&hamster).Association("PreferredToy")
	if err != nil {
		t.Fatal(err)
	}
	count, err = a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected %d got %d", 1, count)
	}
}

func TestAssociationBelongsTo(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testAssociationBelongsTo,
			&fixture.Post{}, &fixture.Comment{}, &fixture.Category{},
		)
	}
}

func testAssociationBelongsTo(t *testing.T, db *DB) {
	_, err := db.Automigrate(
		&fixture.Post{}, &fixture.Comment{}, fixture.Category{},
	)
	if err != nil {
		t.Fatal(err)
	}
	post := fixture.Post{
		Title:        "post belongs to",
		Body:         "body belongs to",
		Category:     fixture.Category{Name: "Category 1"},
		MainCategory: fixture.Category{Name: "Main Category 1"},
	}
	err = db.Begin().Save(&post)
	if err != nil {
		t.Fatal(err)
	}
	if post.Category.ID == 0 || post.MainCategory.ID == 0 {
		t.Errorf("Category's primary key should be updated")
	}

	if post.CategoryID.Int64 == 0 || post.MainCategoryID == 0 {
		t.Errorf("post's foreign key should be updated")
	}

	// Query
	var category1 fixture.Category
	a, err := db.Begin().Model(&post).Association("Category")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&category1)
	if err != nil {
		t.Fatal(err)
	}

	if category1.Name != post.Category.Name {
		t.Errorf("expected %s got %s", post.Category.Name, category1.Name)
	}

	var mainCategory1 fixture.Category
	a, err = db.Begin().Model(&post).Association("MainCategory")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&mainCategory1)
	if err != nil {
		t.Fatal(err)
	}
	if mainCategory1.Name != post.MainCategory.Name {
		t.Errorf("expected %s got %s", post.MainCategory.Name, mainCategory1.Name)
	}

	var category11 fixture.Category
	err = db.Begin().Model(&post).Related(&category11)
	if err != nil {
		t.Fatal(err)
	}
	if category11.Name != post.Category.Name {
		t.Errorf("expected %s got %s", post.Category.Name, category11.Name)
	}

	a, err = db.Begin().Model(&post).Association("Category")
	if err != nil {
		t.Fatal(err)
	}
	count, err := a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected %d got %d", 1, count)
	}

	a, err = db.Begin().Model(&post).Association("MainCategory")
	if err != nil {
		t.Fatal(err)
	}
	count, err = a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected %d got %d", 1, count)
	}
}

func TestAssociationBelongsToOverideFK(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testAssociationBelongsToOverideFK)
	}
}

func testAssociationBelongsToOverideFK(t *testing.T, db *DB) {
	type Profile struct {
		model.Model
		Name string
	}
	type User struct {
		model.Model
		Profile      Profile `gorm:"ForeignKey:ProfileRefer"`
		ProfileRefer int64
	}
	e := db.NewEngine()
	f, err := scope.FieldByName(e, &User{}, "Profile")
	if err != nil {
		t.Fatal(err)
	}
	if f.Relationship.Kind != "belongs_to" {
		t.Errorf("expected belongs_to got %s", f.Relationship.Kind)
	}
	fk := []string{"ProfileRefer"}
	if !reflect.DeepEqual(f.Relationship.ForeignFieldNames, fk) {
		t.Errorf("expected %v got %v", fk, f.Relationship.ForeignFieldNames)
	}
	fn := []string{"ID"}
	if !reflect.DeepEqual(f.Relationship.AssociationForeignFieldNames, fn) {
		t.Errorf("expected %v got %v", fn, f.Relationship.AssociationForeignFieldNames)
	}
}
func TestAssociationBelongsToOverideFK2(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testAssociationBelongsToOverideFK2)
	}
}

func testAssociationBelongsToOverideFK2(t *testing.T, db *DB) {
	type Profile struct {
		model.Model
		Refer string
		Name  string
	}
	type User struct {
		model.Model
		Profile   Profile `gorm:"ForeignKey:ProfileID;AssociationForeignKey:Refer"`
		ProfileID int64
	}
	e := db.NewEngine()
	f, err := scope.FieldByName(e, &User{}, "Profile")
	if err != nil {
		t.Fatal(err)
	}
	if f.Relationship.Kind != "belongs_to" {
		t.Errorf("expected belongs_to got %s", f.Relationship.Kind)
	}
	fk := []string{"ProfileID"}
	if !reflect.DeepEqual(f.Relationship.ForeignFieldNames, fk) {
		t.Errorf("expected %v got %v", fk, f.Relationship.ForeignFieldNames)
	}
	fn := []string{"Refer"}
	if !reflect.DeepEqual(f.Relationship.AssociationForeignFieldNames, fn) {
		t.Errorf("expected %v got %v", fn, f.Relationship.AssociationForeignFieldNames)
	}
}

func TestAssociationHasOne(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testAssociationHasOne,
			&fixture.User{}, &fixture.CreditCard{},
		)
	}
}

func testAssociationHasOne(t *testing.T, db *DB) {
	_, err := db.Automigrate(
		&fixture.User{}, &fixture.CreditCard{},
	)
	if err != nil {
		t.Fatal(err)
	}
	user := fixture.User{
		Name:       "has one",
		CreditCard: fixture.CreditCard{Number: "411111111111"},
	}
	err = db.Begin().Save(&user)
	if err != nil {
		t.Fatal(err)
	}

	if user.CreditCard.UserID.Int64 == 0 {
		t.Errorf("CreditCard's foreign key should be updated")
	}

	// Query
	var creditCard1 fixture.CreditCard
	err = db.Begin().Model(&user).Related(&creditCard1)
	if err != nil {
		t.Fatal(err)
	}

	if creditCard1.Number != "411111111111" {
		t.Errorf("Query has one relations with Related")
	}

	var creditCard11 fixture.CreditCard
	a, err := db.Begin().Model(&user).Association("CreditCard")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&creditCard11)
	if err != nil {
		t.Fatal(err)
	}
	if creditCard11.Number != "411111111111" {
		t.Errorf("Query has one relations with Related")
	}

	a, err = db.Begin().Model(&user).Association("CreditCard")
	if err != nil {
		t.Fatal(err)
	}
	count, err := a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected %d got %d", 1, count)
	}
	// Append
	var creditcard2 = fixture.CreditCard{
		Number: "411111111112",
	}
	a, err = db.Begin().Model(&user).Association("CreditCard")
	if err != nil {
		t.Fatal(err)
	}

	err = a.Append(&creditcard2)
	if err != nil {
		t.Fatal(err)
	}

	if creditcard2.ID == 0 {
		t.Errorf("Creditcard should has ID when created with Append")
	}

	count, err = a.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected %d got %d", 1, count)
	}

	var creditcard21 fixture.CreditCard
	err = db.Begin().Model(&user).Related(&creditcard21)
	if err != nil {
		t.Fatal(err)
	}
	if creditcard21.Number != "411111111112" {
		t.Errorf("CreditCard should be updated with Append")
	}
}

func TestAssociationHasOneOverideFK(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testAssociationHasOneOverideFK)
	}
}

func testAssociationHasOneOverideFK(t *testing.T, db *DB) {
	type Profile struct {
		model.Model
		Name      string
		UserRefer int64
	}

	type User struct {
		model.Model
		Profile Profile `gorm:"ForeignKey:UserRefer"`
	}
	f, err := scope.FieldByName(db.NewEngine(), &User{}, "Profile")
	if err != nil {
		t.Fatal(err)
	}
	rel := f.Relationship
	if rel.Kind != "has_one" {
		t.Errorf("expected has_one got %s", rel.Kind)
	}
	e := []string{"UserRefer"}
	if !reflect.DeepEqual(rel.ForeignFieldNames, e) {
		t.Errorf("expected %v got %v", e, rel.ForeignFieldNames)
	}

	e = []string{"ID"}
	if !reflect.DeepEqual(rel.AssociationForeignFieldNames, e) {
		t.Errorf("expected %v got %v", e, rel.AssociationForeignFieldNames)
	}
}

func TestAssociationHasOneOverideFK2(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testAssociationHasOneOverideFK2)
	}
}

func testAssociationHasOneOverideFK2(t *testing.T, db *DB) {
	type Profile struct {
		model.Model
		Name   string
		UserID int64
	}

	type User struct {
		model.Model
		Refer   string
		Profile Profile `gorm:"ForeignKey:UserID;AssociationForeignKey:Refer"`
	}
	f, err := scope.FieldByName(db.NewEngine(), &User{}, "Profile")
	if err != nil {
		t.Fatal(err)
	}
	rel := f.Relationship
	if rel.Kind != "has_one" {
		t.Errorf("expected has_one got %s", rel.Kind)
	}
	e := []string{"UserID"}
	if !reflect.DeepEqual(rel.ForeignFieldNames, e) {
		t.Errorf("expected %v got %v", e, rel.ForeignFieldNames)
	}

	e = []string{"Refer"}
	if !reflect.DeepEqual(rel.AssociationForeignFieldNames, e) {
		t.Errorf("expected %v got %v", e, rel.AssociationForeignFieldNames)
	}
}

func TestAssociationHasMany(t *testing.T) {
	for _, d := range allTestDB() {
		runWrapDB(t, d, testAssociationHasMany, &fixture.Post{}, &fixture.Comment{})
	}
}

func testAssociationHasMany(t *testing.T, db *DB) {
	_, err := db.Automigrate(&fixture.Post{}, &fixture.Comment{})
	if err != nil {
		t.Fatal(err)
	}
	post := fixture.Post{
		Title:    "post has many",
		Body:     "body has many",
		Comments: []*fixture.Comment{{Content: "Comment 1"}, {Content: "Comment 2"}},
	}
	err = db.Begin().Save(&post)
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range post.Comments {
		if comment.PostID == 0 {
			t.Errorf("comment's PostID should be updated")
		}
	}
	var compareComments = func(comments []fixture.Comment, contents []string) bool {
		var commentContents []string
		for _, comment := range comments {
			commentContents = append(commentContents, comment.Content)
		}
		sort.Strings(commentContents)
		sort.Strings(contents)
		return reflect.DeepEqual(commentContents, contents)
	}

	// Query
	err = db.Begin().First(&fixture.Comment{}, "content = ?", "Comment 1")
	if err != nil {
		t.Fatal(err)
	}

	var comments1 []fixture.Comment

	a, err := db.Begin().Model(&post).Association("Comments")
	if err != nil {
		t.Fatal(err)
	}
	err = a.Find(&comments1)
	if err != nil {
		t.Fatal(err)
	}
	if !compareComments(comments1, []string{"Comment 1", "Comment 2"}) {
		t.Errorf("Query has many relations with Association")
	}

	var comments11 []fixture.Comment
	db.Begin().Model(&post).Related(&comments11)
	if !compareComments(comments11, []string{"Comment 1", "Comment 2"}) {
		t.Errorf("Query has many relations with Related")
	}
}
