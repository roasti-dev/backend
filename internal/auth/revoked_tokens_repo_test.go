package auth_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	database, err := db.NewSQLite(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.InitSchema(database))
	t.Cleanup(func() { database.Close() }) //nolint: errcheck
	return database
}

func TestRevokedTokenRepository_Add(t *testing.T) {
	repo := auth.NewRevokedTokenRepository(setupTestDB(t))

	err := repo.Add(t.Context(), "some-token")
	require.NoError(t, err)

	revoked, err := repo.IsRevoked(t.Context(), "some-token")
	require.NoError(t, err)
	assert.True(t, revoked)
}

func TestRevokedTokenRepository_IsRevoked_NotFound(t *testing.T) {
	repo := auth.NewRevokedTokenRepository(setupTestDB(t))

	revoked, err := repo.IsRevoked(t.Context(), "unknown-token")
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestRevokedTokenRepository_DeleteExpired(t *testing.T) {
	db := setupTestDB(t)
	repo := auth.NewRevokedTokenRepository(db)

	_, err := db.Exec(`INSERT INTO revoked_tokens (token_hash, revoked_at) VALUES (?, datetime('now', '-91 days'))`, "oldhash")
	require.NoError(t, err)

	err = repo.DeleteExpired(t.Context())
	require.NoError(t, err)

	revoked, err := repo.IsRevoked(t.Context(), "oldhash")
	require.NoError(t, err)
	assert.False(t, revoked)
}
