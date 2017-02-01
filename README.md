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
- [ ] postgresql
- [ ] mysql
- [ ] mssql
- [ ] sqlite


##  Motivation

I wanted to contribute to gorm by improving performance , I ended up being
enlightened.

## Installation

	go get -u github.com/gernest/ngorm



## Usage

See [GUIDE](https://ngorm.github.io/) for getting started, examples and so much more.

##  FAQ

### Why ngorm?

Seriously? Why not?

###  Can I use my gorm models?

Yep

### Why there is no support for database x?

This is still under development, if the database is not on the checklist above,
please open an issue for feature request.

### What is the difference between gorm and ngorm?

As the name implies, ngorm is based on gorm. An attempt to modernise gorm.These
are some of the things that differentiate the two.

Oh! Wait, there is no much difference from the user point of view. I
restructured the gorm  source files. Removed the global state, reworked
callbacks/hooks to be more straight forward and added support for ql database.
The API is almost the same.

The bonus here is, it is easy to measure performance, and hence improve. The
code base can easily be groked hence easy to contribute  , also this comes with
an extensive test suite to make sure that regressions can not be introduced
without being detected.

# Todo

- [ ] Preload
