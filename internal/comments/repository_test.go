package comments_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/comments"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func setupCommentRepo(t *testing.T) (*comments.Repo, *sql.DB) {
	t.Helper()
	database := testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, database, "user-1")
	testutil.CreateTestUser(t, database, "user-2")
	return comments.NewRepository(database), database
}

func TestCommentRepository_Create(t *testing.T) {
	t.Run("inserts comment and returns it with author info", func(t *testing.T) {
		repo, _ := setupCommentRepo(t)

		author := models.UserPreview{Id: "user-2"}
		comment := models.PostComment{
			Id:     "comment-1",
			Author: &author,
			Text:   "Great post!",
		}
		got, err := repo.Create(t.Context(), comment, "post-1", "post")
		require.NoError(t, err)
		assert.Equal(t, "comment-1", got.Id)
		assert.Equal(t, "Great post!", got.Text)
		assert.Equal(t, "user-2", got.Author.Id)
		assert.Equal(t, "user-2", got.Author.Username)
	})

	t.Run("reply has parent_id set", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-1", "root")

		parentID := "comment-1"
		author := models.UserPreview{Id: "user-2"}
		reply := models.PostComment{
			Id:       "comment-2",
			Author:   &author,
			Text:     "reply!",
			ParentId: &parentID,
		}
		got, err := repo.Create(t.Context(), reply, "post-1", "post")
		require.NoError(t, err)
		require.NotNil(t, got.ParentId)
		assert.Equal(t, "comment-1", *got.ParentId)
	})
}

func TestCommentRepository_Delete(t *testing.T) {
	t.Run("soft deletes existing comment", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-2", "hello")

		require.NoError(t, repo.Delete(t.Context(), "comment-1"))

		_, err := repo.GetAuthorID(t.Context(), "comment-1")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("already deleted returns ErrNotFound", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-2", "hello")

		require.NoError(t, repo.Delete(t.Context(), "comment-1"))
		err := repo.Delete(t.Context(), "comment-1")
		assert.ErrorIs(t, err, comments.ErrNotFound)
	})

	t.Run("non-existent returns ErrNotFound", func(t *testing.T) {
		repo, _ := setupCommentRepo(t)

		err := repo.Delete(t.Context(), "non-existent")
		assert.ErrorIs(t, err, comments.ErrNotFound)
	})
}

func TestCommentRepository_GetAuthorID(t *testing.T) {
	t.Run("returns author id", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-2", "hello")

		authorID, err := repo.GetAuthorID(t.Context(), "comment-1")
		require.NoError(t, err)
		assert.Equal(t, "user-2", authorID)
	})

	t.Run("non-existent returns sql.ErrNoRows", func(t *testing.T) {
		repo, _ := setupCommentRepo(t)

		_, err := repo.GetAuthorID(t.Context(), "non-existent")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("soft-deleted returns sql.ErrNoRows", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-2", "hello")
		require.NoError(t, repo.Delete(t.Context(), "comment-1"))

		_, err := repo.GetAuthorID(t.Context(), "comment-1")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestCommentRepository_ExistsInTarget(t *testing.T) {
	t.Run("returns true for existing comment in target", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-1", "hello")

		exists, err := repo.ExistsInTarget(t.Context(), "comment-1", "post-1")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for comment in different target", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-1", "hello")

		exists, err := repo.ExistsInTarget(t.Context(), "comment-1", "post-2")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns false for soft-deleted comment", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-1", "hello")
		require.NoError(t, repo.Delete(t.Context(), "comment-1"))

		exists, err := repo.ExistsInTarget(t.Context(), "comment-1", "post-1")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
