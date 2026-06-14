package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors_NotNil(t *testing.T) {
	assert.Error(t, domain.ErrNotFound)
	assert.Error(t, domain.ErrAlreadyExists)
	assert.Error(t, domain.ErrLocked)
	assert.Error(t, domain.ErrNoSession)
}

func TestSentinelErrors_Distinct(t *testing.T) {
	errs := []error{
		domain.ErrNotFound,
		domain.ErrAlreadyExists,
		domain.ErrLocked,
		domain.ErrNoSession,
	}
	for i, a := range errs {
		for j, b := range errs {
			if i != j {
				assert.False(t, errors.Is(a, b),
					"expected %v and %v to be distinct sentinel errors", a, b)
			}
		}
	}
}

func TestSentinelErrors_Wrappable(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", domain.ErrNotFound)
	assert.True(t, errors.Is(wrapped, domain.ErrNotFound))

	wrapped = fmt.Errorf("outer: %w", domain.ErrAlreadyExists)
	assert.True(t, errors.Is(wrapped, domain.ErrAlreadyExists))

	wrapped = fmt.Errorf("outer: %w", domain.ErrLocked)
	assert.True(t, errors.Is(wrapped, domain.ErrLocked))

	wrapped = fmt.Errorf("outer: %w", domain.ErrNoSession)
	assert.True(t, errors.Is(wrapped, domain.ErrNoSession))
}

func TestSentinelErrors_Messages(t *testing.T) {
	assert.Equal(t, "not found", domain.ErrNotFound.Error())
	assert.Equal(t, "already exists", domain.ErrAlreadyExists.Error())
	assert.Equal(t, "locked", domain.ErrLocked.Error())
	assert.Equal(t, "no active session", domain.ErrNoSession.Error())
}
