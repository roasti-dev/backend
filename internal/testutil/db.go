package testutil

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/db"
)

func SetupTestDB(t *testing.T) *sql.DB {
	database, err := db.NewSQLite(t.Context(), ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.InitSchema(database))
	t.Cleanup(func() { database.Close() }) //nolint: errcheck
	return database
}
