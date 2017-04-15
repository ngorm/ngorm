# NGORM

 The fork of gorm,The fantastic ORM( Object Relational Mapping) library for Golang, that focus on

* Performance
* Maintainability
* Modularity
* Battle testing
* Extensibility
* Safety
* Developer friendly for real

[![GoDoc](https://godoc.org/github.com/ngorm/ngorm?status.svg)](https://godoc.org/github.com/ngorm/ngorm)[![Coverage Status](https://coveralls.io/repos/github/ngorm/ngorm/badge.svg?branch=master)](https://coveralls.io/github/ngorm/ngorm?branch=master)[![Build Status](https://travis-ci.org/ngorm/ngorm.svg?branch=master)](https://travis-ci.org/ngorm/ngorm)

## Overview

* Full-Featured ORM (almost)
* Associations (Has One, Has Many, Belongs To, Many To Many, Polymorphism)
* Callbacks (Before/After Create/Save/Update/Delete/Find)
* Preloading (eager loading)
* Transactions
* Composite Primary Key
* SQL Builder
* Auto Migrations
* Logger
* Extendible, write Plugins based on NGORM hooks
* Every feature comes with tests
* Developer Friendly

Documentation https://godoc.org/github.com/ngorm/ngorm

Database support

- [x] [ql](https://godoc.org/github.com/cznic/ql)
- [x] postgresql
- [ ] mysql
- [ ] mssql
- [ ] sqlite


# Table of contents
- Introduction
  - [Synopsis](#synopsis)
  - [Installation](#installation)
  - [Connecting to a database](#connecting-to-a-database)
  - [Migration](#migrations)

- API
  - AddForeignKey
  - AddIndex
  - AddUniqueIndex
  - Assign
  - Association
  - Attrs
  - Automigrate
  - CommonDB
  - Count
  - CreateTable
  - Delete
  - Dialect
  - DropColumn
  - DropTable
  - DropTableIfExests
  - Find
  - First
  - FirstOrCreate
  - FirstOrInit
  - Group
  - HasTable
  - Having
  - Set
  - Joins
  - Last
  - Limit
  - Model
  - ModifyColumn
  - Not
  - Offset
  - Omit
  - Or
  - Order
  - Pluck
  - Preload
  - Related
  - RemoveIndex
  - Save
  - Select
  - SingulatTable
  - Table
  - Update
  - UpdateColumn
  - UpdateColumns
  - Updates
  - Where


# Synopsis

NGORM is a fork of gorm. I initially forked gorm for the purpose of improving
performance, after navigating through the whole code base it dawned to me that
there was no straight way to benchmark and any efforts won't be conveying the
truth about what is happening.

Queries are executed using `database/sql` package, majority of the time is spent
generating the queries from models. So there can be two places for
benchmarking.

First is the code that is responsible to take models and generate SQl. Second is the
code that uses `database/sql` to run the queries i.e measure how fast/efficient
are the generated queries.

The second part is too broad and vague, and might have different outcomes based
on the nature of the database. So the focus of ngorm is to make sure all the
cases are addressed, in a way that  the library generates the best possible
queries for the supported databases.

## Installation

	go get -u github.com/ngorm/ngorm

## Connecting to a database

NGORM uses a similar API as the one used by `database/sql` package to connect
to a database.

> connections to the databases requires importing of the respective driver

```go
package main

import (
	"log"

	// You must import the driver for the database you wish to connect to. In
	// this example I am using the ql and postgresql driver, this should work similar for the
	// other supported databases.

    // driver for postgresql database
	_ "github.com/ngorm/ngorm/postgres"
    // driver for ql database
	_ "github.com/ngorm/ngorm/ql"
	"github.com/ngorm/ngorm"
)

func main() {

	// The frist argument is the dialect or the name of the database driver that
	// you wish to to connect to, the second argument is connection information
	// please check the appropriate driver for more information on the arguments
	// that are passed to database/sql Open.
	db, err := ngorm.Open("ql-mem", "est.db")
	if err != nil {
		log.Fatal(err)
	}

	// Do something with db
}
```

The returned `ngorm.DB` instance is safe. It is a good idea to have only one
instance of this object throughout your application life cycle. Make it a global
or pass it in context.

## Migrations
ngorm support automatic migrations of models. ngorm reuses the gorm logic for loading models so all the valid gorm models are also valid ngorm model.

```go
	type Profile struct {
		model.Model
		Name string
	}
	type User struct {
		model.Model
		Profile   Profile
		ProfileID int
	}

	db, err := Open("ql-mem", "test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	s, err := db.AutomigrateSQL(&User{}, &Profile{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s.Q)
	//Output:
	// BEGIN TRANSACTION;
	// 	CREATE TABLE users (id int64,created_at time,updated_at time,deleted_at time,profile_id int ) ;
	// 	CREATE INDEX idx_users_deleted_at ON users(deleted_at);
	// 	CREATE TABLE profiles (id int64,created_at time,updated_at time,deleted_at time,name string ) ;
	// 	CREATE INDEX idx_profiles_deleted_at ON profiles(deleted_at);
	// COMMIT;
  ```


# API

ngorm api borrows heavily from gorm. 

## `DB.AddForeignKey`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```


## `DB.AddIndex`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.AddUniqueIndex`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```
## `DB.Assign`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Association`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Attrs`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Automigrate`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.CommonDB`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Count`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.CreateTable`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Delete`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Dialect`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.DropColumn`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.DropTable`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.DropTableIfExests`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Find`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.First`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.FirstOrCreate`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.FirstOrInit`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Group`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.HasTable`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Having`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Set`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Joins`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Last`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Limit`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Model`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

##  `DB.ModifyColumn`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Not`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Offset`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Omit`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Or`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Order`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```


## `DB.Pluck`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Preload`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

##`DB.Related`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.RemoveIndex`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Save`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Select`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.SingulatTable`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Table`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Update`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.UpdateColumn`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.UpdateColumns`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Updates`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```

## `DB.Where`

```go
 db.Model(User{}).AddForeignKeySQL("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

> Generates 

```sql
-- ql has no support for foreign keys
```

```sql
ALTER TABLE "users" ADD CONSTRAINT "users_city_id_cities_id_foreign" FOREIGN KEY ("city_id") REFERENCES cities(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
```
