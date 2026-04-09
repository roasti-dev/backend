package testutil

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/beans"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/posts"
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

func CreateTestPost(t *testing.T, repo *posts.Repository, postID, authorID string) models.Post {
	t.Helper()
	p := models.Post{
		Id:        postID,
		Title:     "Test Post",
		Blocks:    []models.PostBlock{},
		Author:    models.UserPreview{Id: authorID},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(t.Context(), p))
	return p
}

func CreateTestComment(t *testing.T, db *sql.DB, commentID, postID, authorID, text string) {
	t.Helper()
	now := time.Now().UTC()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO comments (id, target_id, target_type, author_id, text, created_at, updated_at) VALUES (?, ?, 'post', ?, ?, ?, ?)`,
		commentID, postID, authorID, text, now, now,
	)
	require.NoError(t, err)
}

func CreateTestCommentReply(t *testing.T, db *sql.DB, commentID, postID, authorID, text, parentID string) {
	t.Helper()
	now := time.Now().UTC()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO comments (id, target_id, target_type, author_id, text, parent_id, created_at, updated_at) VALUES (?, ?, 'post', ?, ?, ?, ?, ?)`,
		commentID, postID, authorID, text, parentID, now, now,
	)
	require.NoError(t, err)
}

func CreateTestBean(t *testing.T, repo *beans.Repository, beanID, authorID string) models.Bean {
	t.Helper()
	require.NoError(t, repo.Create(t.Context(), beanID, authorID, models.BeanPayload{
		Name:      beanID,
		RoastType: models.BeanRoastTypeEspresso,
		Roaster:   "Test Roaster",
	}))
	bean, err := repo.GetByID(t.Context(), beanID)
	require.NoError(t, err)
	return bean
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
