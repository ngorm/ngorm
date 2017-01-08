# Definitive Guide to Object Relational Mapping with ngorm

Welcome! Thanks you for taking your time to check out on ngorm. This is a rather
TL:DR doc which will walk you through all things necessary to get started with
ngorm. Please check godoc reference too, there is a lot of good details there.

Enjoy!

## Table of contents

- [Getting started](#getting-started)
 - [Installation](#installation)
 - [Connecting to databases](#connecting-to-database)
 - [Migrations](#migrations)
 - [Create](#create)
 - [Query](#query)
 - [Preload](#preload)
 - [Update](#update)
 - [Delete](#delete)
 - [Associations](#associations)
    - [Belogs To](#belongs-to)
    - [Has one](#has-one)
    - [Has many](#has-many)
    - [Many to many](many-to-many)
    - [Polymorphism](#polymorphism)

- [Advanced](#advanced)
 - [Hooks](#hooks)
 - [Logging](#logging)
 - [SQL building](#sql-building)
 - [SQL execution](#sql-execution)
 - [Transactions](#transactions)

- [Primer on `database/sql` package](#primer-on-database-sql-package)
