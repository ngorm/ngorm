# NGORM

 The fork of gorm,The fantastic ORM( Object Relational Mapping) library for Golang, that focus on

* Performance
* Maintainability
* Modularity
* Battle testing
* Extensibility
* Safety
* Developer friendly for real

[![Build Status](https://travis-ci.org/gernest/ngorm.svg?branch=master)](https://travis-ci.org/gernest/ngorm) [![Coverage Status](https://coveralls.io/repos/github/gernest/ngorm/badge.svg?branch=master)](https://coveralls.io/github/gernest/ngorm?branch=master) [![GoDoc](https://godoc.org/github.com/gernest/ngorm?status.svg)](https://godoc.org/github.com/gernest/ngorm) [![Go Report Card](https://goreportcard.com/badge/github.com/gernest/ngorm)](https://goreportcard.com/report/github.com/gernest/ngorm)

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

Documentation https://godoc.org/github.com/gernest/ngorm

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

See [GUIDE](GUIDE.md) for getting started, examples and so much more.

##  FAQ

### Why ngorm?

Seriously? Why not?

###  Can I use my gorm models?

Yep

### Why there is no support for database x?

There are check boxes on this README. If you find the database is unchecked then
it might be in a queue you can come back in the future and hopefully it should be
there!

Theres is a [board tracking database support](https://github.com/gernest/ngorm/projects/2)
I am a sole maintainer, I'm good in ql so that is why I'm maintaining support
for it. If you are good in any of the databases, I can help you getting started
so you can add support and help maintaining.

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
