package regexes

import "regexp"

var (
	DistinctSQL    = regexp.MustCompile(`(?i)distinct[^a-z]+[a-z]+`)
	Column         = regexp.MustCompile("^[a-zA-Z]+(\\.[a-zA-Z]+)*$") // only match string like `name`, `users.name`
	IsNumber       = regexp.MustCompile("^\\s*\\d+\\s*$")             // match if string is number
	Comparison     = regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS|IN) ")
	CcountingQuery = regexp.MustCompile("(?i)^count(.+)$")
)
