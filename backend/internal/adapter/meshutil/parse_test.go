package meshutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBibs_Newlines(t *testing.T) {
	assert.Equal(t, []int{101, 202, 303}, ParseBibs("101\n\nabc\n202\n  303  \n"))
}

func TestParseBibs_Commas(t *testing.T) {
	assert.Equal(t, []int{101, 202, 303}, ParseBibs("101,202,303"))
}

func TestParseBibs_Spaces(t *testing.T) {
	assert.Equal(t, []int{101, 202, 303}, ParseBibs("101 202 303"))
}

func TestParseBibs_Mixed(t *testing.T) {
	assert.Equal(t, []int{101, 202, 303}, ParseBibs("101, 202,303"))
}

func TestParseBibs_Empty(t *testing.T) {
	assert.Empty(t, ParseBibs(""))
	assert.Empty(t, ParseBibs("\n\n\n"))
}

func TestParseBibs_SingleBib(t *testing.T) {
	assert.Equal(t, []int{42}, ParseBibs("42"))
}

func TestParseBibs_AllNonNumeric(t *testing.T) {
	assert.Empty(t, ParseBibs("abc def ghi"))
}

func TestParseQuery(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		wantBib int
		wantOK  bool
	}{
		{"lowercase", "query 101", 101, true},
		{"uppercase", "QUERY 101", 101, true},
		{"mixed case", "Query 101", 101, true},
		{"leading/trailing whitespace", "  query 101  ", 101, true},
		{"extra internal spaces", "query   101", 101, true},
		{"not a query", "101", 0, false},
		{"query without bib", "query", 0, false},
		{"query with non-numeric arg", "query abc", 0, false},
		{"query with trailing junk", "query 101 now", 0, false},
		{"empty", "", 0, false},
		{"query is a substring not the whole command", "please query 101", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bib, ok := ParseQuery(tc.text)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantBib, bib)
		})
	}
}

func TestParseCheckpoint(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"lowercase checkpoint", "checkpoint", true},
		{"uppercase checkpoint", "CHECKPOINT", true},
		{"cp alias", "cp", true},
		{"cp alias uppercase", "CP", true},
		{"leading/trailing whitespace", "  checkpoint  ", true},
		{"not a checkpoint command", "101", false},
		{"checkpoint with trailing junk", "checkpoint now", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseCheckpoint(tc.text))
		})
	}
}

func TestParseCount(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"lowercase count", "count", true},
		{"uppercase count", "COUNT", true},
		{"leading/trailing whitespace", "  count  ", true},
		{"not a count command", "101", false},
		{"count with trailing junk", "count now", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseCount(tc.text))
		})
	}
}

func TestParseSearch(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		wantName string
		wantOK   bool
	}{
		{"lowercase", "search smith", "smith", true},
		{"uppercase command", "SEARCH smith", "smith", true},
		{"leading/trailing whitespace", "  search smith  ", "smith", true},
		{"multi-word name", "search van der berg", "van der berg", true},
		{"extra internal spaces before name", "search   smith", "smith", true},
		{"not a search command", "101", "", false},
		{"search without name", "search", "", false},
		{"search is a substring not the whole command", "please search smith", "", false},
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, ok := ParseSearch(tc.text)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantName, name)
		})
	}
}

func TestParseDup(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		wantBib int
		wantOK  bool
	}{
		{"lowercase", "dup 101", 101, true},
		{"uppercase", "DUP 101", 101, true},
		{"leading/trailing whitespace", "  dup 101  ", 101, true},
		{"extra internal spaces", "dup   101", 101, true},
		{"not a dup command", "101", 0, false},
		{"dup without bib", "dup", 0, false},
		{"dup with non-numeric arg", "dup abc", 0, false},
		{"dup with trailing junk", "dup 101 now", 0, false},
		{"empty", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bib, ok := ParseDup(tc.text)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantBib, bib)
		})
	}
}

func TestParseHelp(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"lowercase help", "help", true},
		{"uppercase help", "HELP", true},
		{"question mark", "?", true},
		{"leading/trailing whitespace", "  help  ", true},
		{"not a help command", "101", false},
		{"help with trailing junk", "help me", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseHelp(tc.text))
		})
	}
}
