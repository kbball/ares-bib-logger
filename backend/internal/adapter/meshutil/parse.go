package meshutil

import (
	"log/slog"
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
