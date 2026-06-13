package repository

import (
	"database/sql"
	"errors"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
)

// mapNotFound translates sql.ErrNoRows to domain.ErrNotFound.
func mapNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
