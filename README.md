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
  - [CommonDB](#commondb)
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

##  CommonDB

##  Count

##  CreateTable

##  Delete

##  Dialect

##  DropColumn

##  DropTable

##  DropTableIfExests

##  Find

Find is uded for looking up things in the database. You can look for one item or a list of items. This works well will the other query building API calls. Something to no note is this is the last call after chaining other API calls. So, you can have something similar to `db.Where(...).Find()` etc.

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

##  FirstOrCreate

##  FirstOrInit

##  Group

##  HasTable

##  Having

##  Set

##  Joins

##  Last

##  Limit

##  Model

##  ModifyColumn

##  Not

##  Offset

##  Omit

##  Or

##  Order

##  Pluck

##  Preload

##  Related

##  RemoveIndex

##  Save

##  Select

Use this to compose `SELECT` queries. The first argument is the Query and you can  pass any positional arguments after it.

eg
```go
db.Select("count(*)")
```

This will build `SELECT count(*)`


##  SingulatTable

##  Table

##  Update

##  UpdateColumn

##  UpdateColumns

##  Updates

##  Where
