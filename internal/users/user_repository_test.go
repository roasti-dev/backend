package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

func ptr[T any](v T) *T { return &v }

func TestUserRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := users.NewUserRepository(db)

	testutil.CreateTestUser(t, db, "user-1")

	t.Run("updates username", func(t *testing.T) {
		err := repo.Update(t.Context(), "user-1", users.UpdateUserFields{
			Username: ptr("new_username"),
		})
		require.NoError(t, err)

		user, err := repo.GetByID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, "new_username", user.Username)
	})

	t.Run("updates bio", func(t *testing.T) {
		err := repo.Update(t.Context(), "user-1", users.UpdateUserFields{
			Bio: ptr("my bio"),
		})
		require.NoError(t, err)

		user, err := repo.GetByID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, ptr("my bio"), user.Bio)
	})

	t.Run("updates avatar_id", func(t *testing.T) {
		err := repo.Update(t.Context(), "user-1", users.UpdateUserFields{
			AvatarID: ptr("avatar-123"),
		})
		require.NoError(t, err)

		user, err := repo.GetByID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, ptr("avatar-123"), user.AvatarID)
	})

	t.Run("does not overwrite unset fields", func(t *testing.T) {
		testutil.CreateTestUser(t, db, "user-2")
		// Set username first
		require.NoError(t, repo.Update(t.Context(), "user-2", users.UpdateUserFields{
			Username: ptr("original_name"),
			Bio:      ptr("original bio"),
		}))

		// Update only bio — username must stay the same
		require.NoError(t, repo.Update(t.Context(), "user-2", users.UpdateUserFields{
			Bio: ptr("updated bio"),
		}))

		user, err := repo.GetByID(t.Context(), "user-2")
		require.NoError(t, err)
		assert.Equal(t, "original_name", user.Username)
		assert.Equal(t, ptr("updated bio"), user.Bio)
	})

	t.Run("empty fields make no changes", func(t *testing.T) {
		testutil.CreateTestUser(t, db, "user-3")
		require.NoError(t, repo.Update(t.Context(), "user-3", users.UpdateUserFields{
			Username: ptr("unchanged"),
		}))

		require.NoError(t, repo.Update(t.Context(), "user-3", users.UpdateUserFields{}))

		user, err := repo.GetByID(t.Context(), "user-3")
		require.NoError(t, err)
		assert.Equal(t, "unchanged", user.Username)
	})
}

func TestLikeRepository_ListByUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	likesRepo := likes.NewRepository(db)
	recipesRepo := recipes.NewRepository(db, db)

	testutil.CreateTestRecipe(t, recipesRepo, "recipe-1", "user-1")
	testutil.CreateTestRecipe(t, recipesRepo, "recipe-2", "user-1")
	testutil.CreateTestLike(t, likesRepo, "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	testutil.CreateTestLike(t, likesRepo, "user-1", "recipe-2", models.LikeTargetTypeRecipe)

	t.Run("returns liked recipes", func(t *testing.T) {
		result, err := likesRepo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 10, 0)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("respects limit", func(t *testing.T) {
		result, err := likesRepo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 1, 0)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("respects offset", func(t *testing.T) {
		result, err := likesRepo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 10, 1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		result, err := likesRepo.ListByUser(t.Context(), "unknown", models.LikeTargetTypeRecipe, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
