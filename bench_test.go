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

func BenchmarkCreateOneToManySQL(b *testing.B) {
	for _, d := range allTestDB() {
		runWrapBenchDB(b, d, benchCreateOneToManySQL, &Person{}, &Pet{})
	}
}

func benchCreateOneToManySQL(b *testing.B, db *DB) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := db.CreateSQL(newperson()); err != nil {
			b.Fatalf("error creating: %s", err)
		}
	}
}

func BenchmarkCreateOneToMany(b *testing.B) {
	for _, d := range allTestDB() {
		runWrapBenchDB(b, d, benchCreateOneToMany, &Person{}, &Pet{})
	}
}

func benchCreateOneToMany(b *testing.B, db *DB) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := db.Create(newperson()); err != nil {
			b.Fatalf("error creating: %s", err)
		}
	}
}

func BenchmarkCreate(b *testing.B) {
	for _, d := range allTestDB() {
		runWrapBenchDB(b, d, benchCreate, &Person{}, &Pet{})
	}
}

func benchCreate(b *testing.B, db *DB) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := db.Create(&Person{Name: "foo"}); err != nil {
			b.Fatalf("error creating: %s", err)
		}
	}
}

func BenchmarkFindOneToMany(b *testing.B) {
	for _, d := range allTestDB() {
		db, err := d.Open()
		if err != nil {
			b.Fatal(err)
		}
		_, err = db.Automigrate(&Person{}, &Pet{})
		if err != nil {
			b.Fatal(err)
		}
		for i := 0; i < 300; i++ {
			if err := db.Create(newperson()); err != nil {
				b.Fatalf("error creating: %s", err)
			}
		}
		b.Run(db.Dialect().GetName(), func(ts *testing.B) {
			benchFindOneToMany(ts, db)
		})
		err = d.Clear(&Person{}, &Pet{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchFindOneToMany(b *testing.B, db *DB) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		persons := []*Person{}
		if err := db.Preload("Pets").Limit(100).Find(&persons); err != nil {
			b.Fatalf("error finding: %s", err)
		}
	}
}

func BenchmarkFind(b *testing.B) {
	for _, d := range allTestDB() {
		db, err := d.Open()
		if err != nil {
			b.Fatal(err)
		}
		_, err = db.Automigrate(&Person{}, &Pet{})
		if err != nil {
			b.Fatal(err)
		}
		for i := 0; i < 300; i++ {
			if err := db.Create(newperson()); err != nil {
				b.Fatalf("error creating: %s", err)
			}
		}
		b.Run(db.Dialect().GetName(), func(ts *testing.B) {
			benchFind(ts, db)
		})
		err = d.Clear(&Person{}, &Pet{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchFind(b *testing.B, db *DB) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		persons := []*Person{}
		if err := db.Find(&persons); err != nil {
			b.Fatalf("error finding: %s", err)
		}
	}
}

func BenchmarkFindSQL(b *testing.B) {
	for _, d := range allTestDB() {
		db, err := d.Open()
		if err != nil {
			b.Fatal(err)
		}
		_, err = db.Automigrate(&Person{}, &Pet{})
		if err != nil {
			b.Fatal(err)
		}
		for i := 0; i < 300; i++ {
			if err := db.Create(newperson()); err != nil {
				b.Fatalf("error creating: %s", err)
			}
		}
		b.Run(db.Dialect().GetName(), func(ts *testing.B) {
			benchFindSQL(ts, db)
		})
		err = d.Clear(&Person{}, &Pet{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchFindSQL(b *testing.B, db *DB) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		persons := []*Person{}
		if _, err := db.FindSQL(&persons); err != nil {
			b.Fatalf("error finding: %s", err)
		}
	}
}
