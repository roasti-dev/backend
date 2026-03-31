package likes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func setupLikeRepo(t *testing.T) *likes.Repository {
	t.Helper()
	database := testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, database, "user-1")
	testutil.CreateTestUser(t, database, "user-2")
	return likes.NewRepository(database)
}

func TestLikeRepository_Delete(t *testing.T) {
	repo := setupLikeRepo(t)

	t.Run("deletes existing like", func(t *testing.T) {
		testutil.CreateTestLike(t, repo, "user-1", "recipe-1", models.LikeTargetTypeRecipe)

		err := repo.Delete(t.Context(), "user-1", "recipe-1", models.LikeTargetTypeRecipe)
		require.NoError(t, err)

		exists, err := repo.Exists(t.Context(), "user-1", "recipe-1", models.LikeTargetTypeRecipe)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("deleting non-existent like does not error", func(t *testing.T) {
		err := repo.Delete(t.Context(), "user-1", "unknown", models.LikeTargetTypeRecipe)
		require.NoError(t, err)
	})
}

func TestLikeRepository_Exists(t *testing.T) {
	repo := setupLikeRepo(t)

	t.Run("returns false when not found", func(t *testing.T) {
		exists, err := repo.Exists(t.Context(), "user-1", "recipe-1", models.LikeTargetTypeRecipe)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestLikeRepository_GetLikedIDs(t *testing.T) {
	repo := setupLikeRepo(t)

	testutil.CreateTestLike(t, repo, "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	testutil.CreateTestLike(t, repo, "user-1", "recipe-2", models.LikeTargetTypeRecipe)
	testutil.CreateTestLike(t, repo, "user-2", "recipe-1", models.LikeTargetTypeRecipe)

	t.Run("returns liked ids for user", func(t *testing.T) {
		result, err := repo.GetLikedIDs(t.Context(), "user-1", models.LikeTargetTypeRecipe, []string{"recipe-1", "recipe-2", "recipe-3"})
		require.NoError(t, err)
		assert.True(t, result["recipe-1"])
		assert.True(t, result["recipe-2"])
		assert.False(t, result["recipe-3"])
	})

	t.Run("does not return likes of other users", func(t *testing.T) {
		result, err := repo.GetLikedIDs(t.Context(), "user-2", models.LikeTargetTypeRecipe, []string{"recipe-1", "recipe-2"})
		require.NoError(t, err)
		assert.True(t, result["recipe-1"])
		assert.False(t, result["recipe-2"])
	})
}

func TestLikesRepository_ListByUser(t *testing.T) {
	repo := setupLikeRepo(t)

	l1 := testutil.CreateTestLike(t, repo, "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	l2 := testutil.CreateTestLike(t, repo, "user-1", "recipe-2", models.LikeTargetTypeRecipe)
	testutil.CreateTestLike(t, repo, "user-2", "recipe-3", models.LikeTargetTypeRecipe)

	t.Run("returns likes for user", func(t *testing.T) {
		result, err := repo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 10, 0)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("does not return other user likes", func(t *testing.T) {
		result, err := repo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 10, 0)
		require.NoError(t, err)
		for _, l := range result {
			assert.NotEqual(t, "user-2", l.UserID)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		result, err := repo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 1, 0)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("respects offset", func(t *testing.T) {
		result, err := repo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 10, 1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("returns in descending order by created_at", func(t *testing.T) {
		result, err := repo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, l2.ID, result[0].ID)
		assert.Equal(t, l1.ID, result[1].ID)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		result, err := repo.ListByUser(t.Context(), "unknown", models.LikeTargetTypeRecipe, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestLikesRepository_CountByUser(t *testing.T) {
	repo := setupLikeRepo(t)

	testutil.CreateTestLike(t, repo, "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	testutil.CreateTestLike(t, repo, "user-1", "recipe-2", models.LikeTargetTypeRecipe)
	testutil.CreateTestLike(t, repo, "user-2", "recipe-3", models.LikeTargetTypeRecipe)

	t.Run("returns correct count for user", func(t *testing.T) {
		count, err := repo.CountByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("does not count other user likes", func(t *testing.T) {
		count, err := repo.CountByUser(t.Context(), "user-2", models.LikeTargetTypeRecipe)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("returns zero for unknown user", func(t *testing.T) {
		count, err := repo.CountByUser(t.Context(), "unknown", models.LikeTargetTypeRecipe)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
