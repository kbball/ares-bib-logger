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

var checkpointPattern = regexp.MustCompile(`(?i)^\s*(?:checkpoint|cp)\s*$`)

// ParseCheckpoint reports whether text is a "checkpoint" (or "cp") command —
// asks this station to report its currently active checkpoint(s).
func ParseCheckpoint(text string) bool {
	return checkpointPattern.MatchString(text)
}

var countPattern = regexp.MustCompile(`(?i)^\s*count\s*$`)

// ParseCount reports whether text is a "count" command — asks this station
// how many bibs it has logged at its active checkpoint(s) so far.
func ParseCount(text string) bool {
	return countPattern.MatchString(text)
}

var searchPattern = regexp.MustCompile(`(?i)^\s*search\s+(\S.*?)\s*$`)

// ParseSearch reports whether text is a "search <name>" command (case-insensitive,
// surrounding whitespace tolerated) and, if so, returns the search term.
func ParseSearch(text string) (name string, ok bool) {
	m := searchPattern.FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	return m[1], true
}

var dupPattern = regexp.MustCompile(`(?i)^\s*dup\s+(\d+)\s*$`)

// ParseDup reports whether text is a "dup <bib>" command (case-insensitive,
// surrounding whitespace tolerated) and, if so, returns the bib number.
func ParseDup(text string) (bib int, ok bool) {
	m := dupPattern.FindStringSubmatch(text)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

var helpPattern = regexp.MustCompile(`(?i)^\s*(?:help|\?)\s*$`)

// ParseHelp reports whether text is a "help" (or "?") command.
func ParseHelp(text string) bool {
	return helpPattern.MatchString(text)
}

// HelpText is the reply sent for the "help" command: a compact list of every
// mesh command this app understands.
const HelpText = "Cmds: <bib list> logs bibs, query <bib>, checkpoint, count, search <name>, dup <bib>, help"
