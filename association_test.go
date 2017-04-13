package ngorm

import (
	"reflect"
	"sort"
	"testing"
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

	// // Append
	// db.Model(&hamster).Association("PreferredToy").Append(&Toy{
	// 	Name: "bike 2",
	// })
	// db.Model(&hamster).Association("OtherToy").Append(&Toy{
	// 	Name: "treadmill 2",
	// })

	// hamsterToy = Toy{}
	// db.Model(&hamster).Association("PreferredToy").Find(&hamsterToy)
	// if hamsterToy.Name != "bike 2" {
	// 	t.Errorf("Should update has one polymorphic association with Append")
	// }

	// hamsterToy = Toy{}
	// db.Model(&hamster).Association("OtherToy").Find(&hamsterToy)
	// if hamsterToy.Name != "treadmill 2" {
	// 	t.Errorf("Should update has one polymorphic association with Append")
	// }

	// if db.Model(&hamster2).Association("PreferredToy").Count() != 1 {
	// 	t.Errorf("Hamster's toys count should be 1 after Append")
	// }

	// if db.Model(&hamster2).Association("OtherToy").Count() != 1 {
	// 	t.Errorf("Hamster's toys count should be 1 after Append")
	// }

	// // Replace
	// db.Model(&hamster).Association("PreferredToy").Replace(&Toy{
	// 	Name: "bike 3",
	// })
	// db.Model(&hamster).Association("OtherToy").Replace(&Toy{
	// 	Name: "treadmill 3",
	// })

	// hamsterToy = Toy{}
	// db.Model(&hamster).Association("PreferredToy").Find(&hamsterToy)
	// if hamsterToy.Name != "bike 3" {
	// 	t.Errorf("Should update has one polymorphic association with Replace")
	// }

	// hamsterToy = Toy{}
	// db.Model(&hamster).Association("OtherToy").Find(&hamsterToy)
	// if hamsterToy.Name != "treadmill 3" {
	// 	t.Errorf("Should update has one polymorphic association with Replace")
	// }

	// if db.Model(&hamster2).Association("PreferredToy").Count() != 1 {
	// 	t.Errorf("hamster's toys count should be 1 after Replace")
	// }

	// if db.Model(&hamster2).Association("OtherToy").Count() != 1 {
	// 	t.Errorf("hamster's toys count should be 1 after Replace")
	// }

	// // Clear
	// db.Model(&hamster).Association("PreferredToy").Append(&Toy{
	// 	Name: "bike 2",
	// })
	// db.Model(&hamster).Association("OtherToy").Append(&Toy{
	// 	Name: "treadmill 2",
	// })

	// if db.Model(&hamster).Association("PreferredToy").Count() != 1 {
	// 	t.Errorf("Hamster's toys should be added with Append")
	// }
	// if db.Model(&hamster).Association("OtherToy").Count() != 1 {
	// 	t.Errorf("Hamster's toys should be added with Append")
	// }

	// db.Model(&hamster).Association("PreferredToy").Clear()

	// if db.Model(&hamster2).Association("PreferredToy").Count() != 0 {
	// 	t.Errorf("Hamster's preferred toy should be cleared with Clear")
	// }
	// if db.Model(&hamster2).Association("OtherToy").Count() != 1 {
	// 	t.Errorf("Hamster's other toy should be still available")
	// }

	// db.Model(&hamster).Association("OtherToy").Clear()
	// if db.Model(&hamster).Association("OtherToy").Count() != 0 {
	// 	t.Errorf("Hamster's other toy should be cleared with Clear")
	// }
}
