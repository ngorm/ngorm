package ngorm

import (
	"testing"
)

type Person struct {
	ID   int64 `gorm:"primary_key"`
	Name string
	Pets []Pet `gorm:"ForeignKey:PersonID"`
}

type Pet struct {
	ID       int64 `gorm:"primary_key"`
	PersonID int64
	Name     string
	Kind     string
}

func newperson() *Person {
	return &Person{
		Name: "Dolan",
		Pets: []Pet{
			{Name: "Garfield", Kind: "cat"},
			{Name: "Oddie", Kind: "dog"},
			{Name: "Reptar", Kind: "fish"},
		},
	}
}
func BenchmarkCreateSQL(b *testing.B) {
	for _, d := range allTestDB() {
		runWrapBenchDB(b, d, benchCreate, &Person{}, &Pet{})
	}
}

func benchCreate(b *testing.B, db *DB) {
	b.ReportAllocs()
	_, err := db.Automigrate(&Person{}, &Pet{})
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		if err := db.Create(newperson()); err != nil {
			b.Fatalf("error creating: %s", err)
		}
	}
}
