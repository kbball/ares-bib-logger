package meshutil

import (
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

// ParseBibs splits text on newlines, commas, and spaces, returning all integer values found.
// Non-numeric and empty tokens are silently skipped.
func ParseBibs(text string) []int {
	var bibs []int
	for _, tok := range strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == ',' || r == ' '
	}) {
		n, err := strconv.Atoi(tok)
		if err != nil {
			slog.Debug("mesh: skipping non-numeric token", "token", tok)
			continue
		}
		bibs = append(bibs, n)
	}
	return bibs
}

var queryPattern = regexp.MustCompile(`(?i)^\s*query\s+(\d+)\s*$`)

// ParseQuery reports whether text is a "query <bib>" command (case-insensitive,
// surrounding whitespace tolerated) and, if so, returns the bib number.
func ParseQuery(text string) (bib int, ok bool) {
	m := queryPattern.FindStringSubmatch(text)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}
