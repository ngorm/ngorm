package ngorm

import (
	"fmt"
	"log"
	"sort"

	"github.com/gernest/ngorm/model"
)

func ExampleOpen() {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		fmt.Println(err)
	} else {
		defer func() { _ = db.Close() }()
		fmt.Println(db.Dialect().GetName())
	}

	//Output:ql-mem
}
func ExampleDB_CreateSQL() {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		fmt.Println(err)
	} else {
		defer func() { _ = db.Close() }()
		type Bar struct {
			ID  int64
			Say string
		}

		b := Bar{Say: "hello"}
		sql, err := db.CreateSQL(&b)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(sql.Q)
			fmt.Printf("$1=%v", sql.Args[0])
		}
	}

	//Output:
	//BEGIN TRANSACTION;
	//	INSERT INTO bars (say) VALUES ($1);
	//COMMIT;
	//$1=hello
}

func ExampleDB_AutomigrateSQL() {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		fmt.Println(err)
	} else {
		defer func() { _ = db.Close() }()

		type Bar struct {
			ID  int64
			Say string
		}

		type Bun struct {
			ID   int64
			Dead bool
		}

		sql, err := db.AutomigrateSQL(&Bar{}, &Bun{})
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(sql.Q)
		}
	}

	//Output:
	//BEGIN TRANSACTION;
	//	CREATE TABLE bars (id int64,say string ) ;
	//	CREATE TABLE buns (id int64,dead bool ) ;
	//COMMIT;
}

func ExampleDB_Automigrate() {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		fmt.Println(err)
	} else {
		defer func() { _ = db.Close() }()

		type Bar struct {
			ID  int64
			Say string
		}

		type Bun struct {
			ID   int64
			Dead bool
		}

		_, err = db.Automigrate(&Bar{}, &Bun{})
		if err != nil {
			fmt.Println(err)
		} else {
			var names []string
			qdb := db.SQLCommon()
			rows, err := qdb.Query("select Name  from __Table ")
			if err != nil {
				fmt.Println(err)
			} else {
				defer func() { _ = rows.Close() }()
				for rows.Next() {
					var n string
					err = rows.Scan(&n)
					if err != nil {
						fmt.Println(err)
					} else {
						names = append(names, n)
					}
				}
				if err := rows.Err(); err != nil {
					log.Fatal(err)
				}
			}
			sort.Strings(names)
			for _, v := range names {
				fmt.Println(v)
			}
		}
	}

	//Output:
	//bars
	//buns

}

func Example_belongsTo() {
	//One to many relationship
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		fmt.Println(err)
	} else {
		defer func() { _ = db.Close() }()

		// Here one user can only have one profile. But one profile can belong
		// to multiple users.
		type Profile struct {
			model.Model
			Name string
		}
		type User struct {
			model.Model
			Profile   Profile
			ProfileID int
		}
		_, err = db.Automigrate(&User{}, &Profile{})
		if err != nil {
			fmt.Println(err)
		} else {
			u := User{
				Profile: Profile{Name: "gernest"},
			}

			// Creating the user with the Profile. The relation will
			// automatically be created and the user.ProfileID will be set to
			// the ID of hte created profile.
			err = db.Create(&u)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(u.ProfileID == int(u.Profile.ID) && u.ProfileID != 0)
			}
		}
	}

	//Output:
	//true

}
