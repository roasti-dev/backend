package testutil

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

func CreateTestUser(t *testing.T, db *sql.DB, userID string) {
	t.Helper()
	repo := users.NewUserRepository(db)
	require.NoError(t, repo.Create(t.Context(), users.User{
		ID:        userID,
		Email:     userID + "@test.com",
		Username:  userID,
		CreatedAt: time.Now().UTC(),
	}))
}

func CreateTestRecipe(t *testing.T, repo *recipes.Repository, recipeID, authorID string) models.Recipe {
	t.Helper()
	r := models.Recipe{
		Id:          recipeID,
		AuthorId:    authorID,
		Title:       "Test Recipe",
		Description: "Test",
		BrewMethod:  models.V60,
		Difficulty:  models.DifficultyEasy,
		Public:      true,
		Steps:       []models.BrewStep{},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))
	return r
}

func CreateTestLike(t *testing.T, repo *likes.Repository, userID, targetID string, targetType models.LikeTargetType) likes.Like {
	t.Helper()
	l := likes.Like{
		ID:         userID + "-" + targetID,
		UserID:     userID,
		TargetID:   targetID,
		TargetType: targetType,
		CreatedAt:  time.Now().UTC(),
	}
	require.NoError(t, repo.Create(t.Context(), l))
	return l
}
