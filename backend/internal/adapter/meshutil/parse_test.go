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
