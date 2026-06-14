package repository_test

import (
	"database/sql"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// newMock creates a sqlmock-backed *sql.DB for use in unit tests.
func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// qe escapes a SQL literal so it can be used as an exact regexp in sqlmock.
func qe(sql string) string {
	return regexp.QuoteMeta(sql)
}
