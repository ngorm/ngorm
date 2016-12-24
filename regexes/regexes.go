package regexes

import "regexp"

var DistinctSQL = regexp.MustCompile(`(?i)distinct[^a-z]+[a-z]+`)
