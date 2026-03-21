package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func TestRevokedTokenRepository_Add(t *testing.T) {
	repo := auth.NewRevokedTokenRepository(testutil.SetupTestDB(t))

	err := repo.Add(t.Context(), "some-token")
	require.NoError(t, err)

	revoked, err := repo.IsRevoked(t.Context(), "some-token")
	require.NoError(t, err)
	assert.True(t, revoked)
}

func TestRevokedTokenRepository_IsRevoked_NotFound(t *testing.T) {
	repo := auth.NewRevokedTokenRepository(testutil.SetupTestDB(t))

	revoked, err := repo.IsRevoked(t.Context(), "unknown-token")
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestRevokedTokenRepository_DeleteExpired(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := auth.NewRevokedTokenRepository(db)

	_, err := db.Exec(`INSERT INTO revoked_tokens (token_hash, revoked_at) VALUES (?, datetime('now', '-91 days'))`, "oldhash")
	require.NoError(t, err)

	err = repo.DeleteExpired(t.Context())
	require.NoError(t, err)

	revoked, err := repo.IsRevoked(t.Context(), "oldhash")
	require.NoError(t, err)
	assert.False(t, revoked)
}
