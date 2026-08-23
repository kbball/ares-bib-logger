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
