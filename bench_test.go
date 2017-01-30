package ngorm

import (
	"testing"

	"github.com/ngorm/ngorm/fixture"
)

func BenchmarkCreateSQL(b *testing.B) {
	for _, d := range AllTestDB() {
		db, err := d.Open()
		if err != nil {
			b.Fatal(err)
		}

		b.Run(db.Dialect().GetName(), func(sb *testing.B) {
			u := fixture.User{}
			n := db.Begin()
			sb.ReportAllocs()
			for i := 0; i < sb.N; i++ {
				u.ID = int64(i)
				_, err = n.CreateTableSQL(&u)
				if err != nil {
					sb.Fatal(err)
				}
			}

		})
	}
}
