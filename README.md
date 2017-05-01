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

__IMPORTANT__: This is not meant to replace gorm. For advanced users you might find this library lacking, I advice you use ngorm.

## Overview

* Full-Featured ORM (almost)
* Associations (Has One, Has Many, Belongs To, Many To Many, Polymorphism)
* Preloading (eager loading)
* Transactions
* Composite Primary Key
* SQL Builder
* Auto Migrations

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
  - [AddForeignKey](#addforeignkey)
  - [AddIndex](#addforeignkey)
  - [AddUniqueIndex](#adduniqueindex)
  - [Assign](#assign)
  - [Association](#association)
  - [Attrs](#attrs)
  - [Automigrate](#automigrate)
  - [Count](#Count)
  - [CreateTable](#createtable)
  - [Delete](#delete)
  - [Dialect](#dialect)
  - [DropColumn](#dropcolumn)
  - [DropTable](#droptable)
  - [DropTableIfExests](#droptableifexests)
  - [Find](#find)
  - [First](#first)
  - [FirstOrCreate](#firstorcreate)
  - [FirstOrInit](#firstorinit)
  - [Group](#group)
  - [HasTable](#hastable)
  - [Having](#having)
  - [Set](#set)
  - [Joins](#joins)
  - [Last](#last)
  - [Limit](#limit)
  - [Model](#model)
  - [ModifyColumn](#modifycolumn)
  - [Not](#not)
  - [Offset](#offset)
  - [Omit](#Omit)
  - [Or](#or)
  - [Order](#order)
  - [Pluck](#pluck)
  - [Preload](#preload)
  - [Related](#related)
  - [RemoveIndex](#removeindex)
  - [Save](#save)
  - [Select](#select)
  - [SingulatTable](#singulattable)
  - [Table](#table)
  - [Update](#update)
  - [UpdateColumn](#updatecolumn)
  - [UpdateColumns](#updatecolumns)
  - [Updates](#updates)
  - [Where](#where)


# Synopsis

Welcome, I have been looking for ways to work with ql database. Since I was
familiar with gorm I tried to add ql dialect. A task that proved too hard due to
limitation of gorm.

I had to rework internals on gorm to reach my end goal. Along the way I had a
vision on how gorm should have looked like if I were to build it today.

The new codebase is in a good shape. One of the benifits is now, you can inspect
the expected queries without excuting anything (for some methods), eg
`db.FIndSQL` will return the query for finding an item/items without hitting the
database.

With the new code base, it is easy to improve as the building blocks are all
visible and well documented. There is also proper error handling. The error
handling is consistent with other Go libraries, no exceptions are raised but
errors are returned so the application developers can handle them.

## Installation

	go get -u github.com/ngorm/ngorm

## Connecting to a database

NGORM uses a similar API as the one used by `database/sql` package to connect
to a database.

> connections to the databases requires importing of the respective driver

```go package main

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

	// The first argument is the dialect or the name of the database driver that
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
ngorm support automatic migrations of models. ngorm reuses the gorm logic for
loading models so all the valid gorm models are also valid ngorm model.

```go
	type Profile struct {
		model.Model
		Name string
	}
	type User struct {
		model.Model
		Profile   Profile
		ProfileID int64
	}

	db, err := Open("ql-mem", "test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// you can inspect expected generated query
	s, err := db.AutomigrateSQL(&User{}, &Profile{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s.Q)

	// Or you can execute migrations like so
	_, err = db.Begin().Automigrate(&User{}, &Profile{})
	if err != nil {
		log.Fatal(err)
	}
	//Output:
	// BEGIN TRANSACTION;
	// 	CREATE TABLE users (id int64,created_at time,updated_at time,deleted_at time,profile_id int64 ) ;
	// 	CREATE INDEX idx_users_deleted_at ON users(deleted_at);
	// 	CREATE TABLE profiles (id int64,created_at time,updated_at time,deleted_at time,name string ) ;
	// 	CREATE INDEX idx_profiles_deleted_at ON profiles(deleted_at);
	// COMMIT;
  ```


# API

ngorm api borrows heavily from gorm. 

##  AddForeignKey

##  AddIndex

##  AddUniqueIndex

##  Assign

##  Association

##  Attrs

##  Automigrate


##  Count
Returns the number of matched rows for a given query.

You can count the number of all users like this.
```go
var count int64
db.Model(&user).Count(&count)
```

Which will execute

```sql
SELECT count(*) FROM users  
```
You can build a normal query by chaining methods and call `Count` at the end,
that way the query will be executed and the matched rows will be counted.

##  CreateTable

Creates a new database table if the table doesn't exist yet. This is useful for
doing database migrations

e.g

You have the following model

```go
	type User struct {
		ID       int64
		Name     string
		Password string
		Email    string
	}
```

```go
db.CreateTable(&User{})
```

Will execute the following query

```sql
BEGIN TRANSACTION; 
	CREATE TABLE users (id int64,name string,password string,email string ) ;
COMMIT;
```

Checking if the table exists already is handled separately by the dialects.

##  Delete
Executes `DELETE` query, which is used to delete rows from a database table.

```go
db.Begin().Delete(&Users{ID: 10})
```

Will execute 
```sql
BEGIN TRANSACTION;
	DELETE FROM users  WHERE id = $1;
COMMIT;
```

Where `$1=10`

##  Dialect
Gives you the instance of dialect which is registered in the `DB`. 

##  DropColumn

Removes columns from database tables by issuing `ALTER TABLE`.

For instance,

```go
db.Model(&USer{}).DropColumn("password")
//ALTER TABLE users DROP COLUMN password
```

##  DropTable

Executes `DROP Table` query. Use this to get rid of database tables. This is the
opposite of `CreateTable` whatever `CreateTable` does to the database this
undoes it.

##  DropTableIfExests

This will check if the table exist in the database before dropping it by calling `DropTable`.

##  Find

Find is used for looking up things in the database. You can look for one item or
a list of items. This works well will the other query building API calls.
Something to no note is this is the last call after chaining other API calls.
So, you can have something similar to `db.Where(...).Find()` etc.

This is an example of looking up for all users.

```go
	type User struct {
		ID   int64
		Name string
	}

	db, err := Open("ql-mem", "test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&User{})
	if err != nil {
		log.Fatal(err)
	}
	v := []string{"gernest", "kemi", "helen"}
	for _, n := range v {
		err = db.Begin().Save(&User{Name: n})
		if err != nil {
			log.Fatal(err)
		}
	}

	users := []User{}
	err = db.Begin().Find(&users)
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		fmt.Println(u.Name)
	}

	//Output:
	// helen
	// kemi
	// gernest
```


##  First

First  fetches the first record and order by primary key.

For instance,

```go
db.Begin().First(&user)
```

Will execute,
```sql
SELECT * FROM users   ORDER BY id ASC LIMIT 1
```

First user by primary key

```go
db.Begin().First(&user,10)
```

will execute

```sql
SELECT * FROM users  WHERE (id = $1) ORDER BY id ASC LIMIT 1
```
Whereby `$1=10`

You can chain other methods as well to build complex queries.

##  FirstOrCreate

This will first try to find the first record that matches, when there is no
match a new record is created.

##  FirstOrInit

##  Group

##  HasTable

Returns true if there is a table for the given value, the value can
either be a string representing a table name or a ngorm model.

##  Having

Builds `HAVING` SQL

##  Set

Store temporary values that will be available across  db chains. The values are
visible at scope leve.

##  Joins

Add `JOIN` SQL

##  Last

Returns the Last row to match the query.

You can gen the last user by
```go
var user User
db.Last(&user)
```

Which will execute the following query

```sql
SELECT * FROM users   ORDER BY id DESC LIMIT 1
```

##  Limit

Add `LIMIT` SQL clause

##  Model

Sets the value as the scope value for the db instance. value must be a valid
ngorm model.

This paves way for chainable query building, since most methods operate on the
scoped model value.

By calling `db.Model(&user)` we are stating that out primary model we want to perate on is `&user`, now from there we can chain further methods to get what we want. like `db.Model(*user).Limit(2).Offset(4).Find(&users)`

##  ModifyColumn

##  Not

##  Offset

Add `OFFSET` SQL clause

##  Omit

Use this to setup fields from the model to be skipped.


##  Or

Add `OR` SQL clause

##  Order

Add `ORDER BY` SQL clause

##  Pluck


##  Preload

##  Related

##  RemoveIndex

##  Save

##  Select

Use this to compose `SELECT` queries. The first argument is the Query and you
can  pass any positional arguments after it.

eg
```go
db.Select("count(*)")
```

This will build `SELECT count(*)`


##  SingulatTable

SingularTable enables or disables singular tables name. By default this is
disabled, meaning table names are in plural.
```
	Model	| Plural table name
	----------------------------
	Session	| sessions
	User	| users

	Model	| Singular table name
	----------------------------
	Session	| session
	User	| user
```

To enable singular tables do,
```go
db.SingularTable(true)
```

To disable singular tables do,
```go
db.SingularTable(false)
```
##  Table
This specify manually the database table you want to run operations on. Most
operations are built automatically from models.

For instance, to find all users you can do `db.Find(&users)` which might
generate `SELECT * FROM users;`. 

You can  select from `scary_users` instead by,

```go
db.Begin().Table("scary_users").Find(&users)
// SELECT * FROM scary_users
```

##  Update

##  UpdateColumn

##  UpdateColumns

##  Updates

##  Where

This generates `WHERE` SQL clause.

Using Where with plain SQL

```go
db.Where("name","gernest")

// WHERE (name=$1)
//$1="gernest"
```

Using Where `IN`

```go
db.Where(e, "name in (?)", []string{"gernest", "gernest 2"})
// WHERE (name in ($1,$2))
// $1="gernest", $2="gernest 2"
```

Using Where with `LIKE`

```go
db.Where(e, "name LIKE ?", "%jin%")
// WHERE (name LIKE $1)
//$1="%jin%"
```
