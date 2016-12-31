//Package ngorm is a Go Object relation mapper that foucs on performance,
//maintainability, modularity,	battle testing, extensibility , safety and
//developer frindliness.
//
// To achieve all of the goals, the project is divided into many components. The
// components are desined in a functional style API, whereby objects are
// explicitly passed around as arguments to functions that operate on them.
//
// This tries to avoid defining methods on structs. This comes at a cost of
// limiting chaining, this cost is intential. I intend to work really hard on
// improving perfomance and thus avoiding spaghetti is not an option.
//
// Installation
//
// You can install  with go get
//   go get -u github.com/gernest/ngorm
//
//
// The package is divided into two phases, Query building and Query execution
// phase.
//
// Query Building
//
// The subpackage engine exposes a structure named Engine. This structure has
// everything necessary to build a query. Most of the functions defined in this
// package subpackages operate on this struct by accepting it as the first
// argument.
//
// Having this as a separate layer helps fine tuning the generated querries and
// also it make easy to test and very that the ORM is doing the right thing. So,
// the generated query can be easily optimised without adding a lot of overhead.
//
// Query execution
//
// This is s the phase where the generated sql query is executed. This phase is as generic as
// possible in a way that you can easily implement adoptes for non SQL database
// and still reap all the benefits of this package.
//
// Table of Ccntents
//
// The following are links to packages under this project.
//   [engine] https://godoc.org/github.com/gernest/ngorm/engine
// This is what drives the whole project, helps with query building and provides
// conveinet structure to help with query execution.
//
//   [scope] https://godoc.org/github.com/gernest/ngorm/scope
// Functions to help with model manipulations.
//
//   [search] https://godoc.org/github.com/gernest/ngorm/search
// Functions to help with search  querries building.
//
//   [hooks] https://godoc.org/github.com/gernest/ngorm/hooks
// Callbacks executed byt ngorm. You can easily overide and provide custom ones
// to suit your needs.
//
//   [logger] https://godoc.org/github.com/gernest/ngorm/logger
// The logger used by ngorm for logging. It is an interface, and a reference
// implementation is provided.
//
//   [dialects] https://godoc.org/github.com/gernest/ngorm/dialects
// Adopts to different SQL databases supported by ngorm. For now ngorm support
// mysql, mssql, postgresql, sqlite and ql.
package ngorm

// DB contains information for current db connection
type DB struct {
}
