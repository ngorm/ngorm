// Package regexes exposes pre compiled reqular expressions that are used by
// ngorm.
package regexes

import "regexp"

var (
	//DistinctSQL matches distict sql query
	DistinctSQL = regexp.MustCompile(`(?i)distinct[^a-z]+[a-z]+`)

	//Column matches database column
	// only match string like `name`, `users.name`
	Column = regexp.MustCompile("^[a-zA-Z]+(\\.[a-zA-Z]+)*$")

	//IsNumber matches if the string is a number.
	IsNumber = regexp.MustCompile("^\\s*\\d+\\s*$")

	//Comparison matches comparison in sql query
	Comparison = regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS|IN) ")

	//CountingQuery matches cound query.
	CountingQuery = regexp.MustCompile("(?i)^count(.+)$")
)
