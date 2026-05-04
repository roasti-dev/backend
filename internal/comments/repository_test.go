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
		comment := models.Comment{
			Id:     "comment-1",
			Author: &author,
			Text:   "Great article!",
		}
		require.NoError(t, repo.Create(t.Context(), comment, "article-1", "article"))

		got, err := repo.GetByID(t.Context(), "comment-1")
		require.NoError(t, err)
		assert.Equal(t, "comment-1", got.Id)
		assert.Equal(t, "Great article!", got.Text)
		assert.Equal(t, "user-2", got.Author.Id)
		assert.Equal(t, "user-2", got.Author.Username)
	})

	t.Run("reply has parent_id set", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-1", "root")

		parentID := "comment-1"
		author := models.UserPreview{Id: "user-2"}
		reply := models.Comment{
			Id:       "comment-2",
			Author:   &author,
			Text:     "reply!",
			ParentId: &parentID,
		}
		require.NoError(t, repo.Create(t.Context(), reply, "article-1", "article"))

		got, err := repo.GetByID(t.Context(), "comment-2")
		require.NoError(t, err)
		require.NotNil(t, got.ParentId)
		assert.Equal(t, "comment-1", *got.ParentId)
	})
}

func TestCommentRepository_Delete(t *testing.T) {
	t.Run("soft deletes existing comment", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-2", "hello")

		require.NoError(t, repo.Delete(t.Context(), "comment-1"))

		_, err := repo.GetAuthorID(t.Context(), "comment-1")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("already deleted returns ErrNotFound", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-2", "hello")

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
		testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-2", "hello")

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
		testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-2", "hello")
		require.NoError(t, repo.Delete(t.Context(), "comment-1"))

		_, err := repo.GetAuthorID(t.Context(), "comment-1")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func defaultPag() models.PaginationParams {
	return models.NewPaginationParams(1, 20)
}

func TestCommentRepository_ListForTarget(t *testing.T) {
	t.Run("returns root comment with reply", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "c1", "article-1", "user-1", "root text")
		testutil.CreateTestCommentReply(t, db, "c2", "article-1", "user-2", "reply text", "c1")

		threads, total, err := repo.ListForTarget(t.Context(), "article-1", defaultPag())
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, threads, 1)
		assert.Equal(t, "root text", threads[0].Text)
		assert.False(t, threads[0].IsDeleted)
		require.NotNil(t, threads[0].Author)
		require.Len(t, threads[0].Replies, 1)
		assert.Equal(t, "reply text", threads[0].Replies[0].Text)
		assert.False(t, threads[0].Replies[0].IsDeleted)
	})

	t.Run("deleted root comment returned as tombstone", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "c1", "article-1", "user-1", "root text")
		require.NoError(t, repo.Delete(t.Context(), "c1"))

		threads, total, err := repo.ListForTarget(t.Context(), "article-1", defaultPag())
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, threads, 1)
		assert.True(t, threads[0].IsDeleted)
		assert.Empty(t, threads[0].Text)
		assert.Nil(t, threads[0].Author)
	})

	t.Run("deleted root with replies: root is tombstone, replies preserved", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "c1", "article-1", "user-1", "root text")
		testutil.CreateTestCommentReply(t, db, "c2", "article-1", "user-2", "reply text", "c1")
		require.NoError(t, repo.Delete(t.Context(), "c1"))

		threads, total, err := repo.ListForTarget(t.Context(), "article-1", defaultPag())
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, threads, 1)
		assert.True(t, threads[0].IsDeleted)
		assert.Empty(t, threads[0].Text)
		assert.Nil(t, threads[0].Author)
		require.Len(t, threads[0].Replies, 1)
		assert.Equal(t, "reply text", threads[0].Replies[0].Text)
		assert.False(t, threads[0].Replies[0].IsDeleted)
	})

	t.Run("deleted reply returned as tombstone", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "c1", "article-1", "user-1", "root text")
		testutil.CreateTestCommentReply(t, db, "c2", "article-1", "user-2", "reply text", "c1")
		require.NoError(t, repo.Delete(t.Context(), "c2"))

		threads, total, err := repo.ListForTarget(t.Context(), "article-1", defaultPag())
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, threads[0].Replies, 1)
		assert.True(t, threads[0].Replies[0].IsDeleted)
		assert.Empty(t, threads[0].Replies[0].Text)
		assert.Nil(t, threads[0].Replies[0].Author)
	})

	t.Run("reply to reply appears under same root", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "c1", "article-1", "user-1", "root")
		testutil.CreateTestCommentReply(t, db, "c2", "article-1", "user-2", "reply", "c1")
		testutil.CreateTestCommentReply(t, db, "c3", "article-1", "user-1", "reply to reply", "c2")

		threads, total, err := repo.ListForTarget(t.Context(), "article-1", defaultPag())
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, threads, 1)
		require.Len(t, threads[0].Replies, 2)
		assert.Equal(t, "c2", threads[0].Replies[0].Id)
		assert.Equal(t, "c3", threads[0].Replies[1].Id)
		assert.Equal(t, "c2", *threads[0].Replies[1].ParentId)
	})

	t.Run("reply to reply on deleted middle comment preserved", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "c1", "article-1", "user-1", "root")
		testutil.CreateTestCommentReply(t, db, "c2", "article-1", "user-2", "reply", "c1")
		testutil.CreateTestCommentReply(t, db, "c3", "article-1", "user-1", "reply to reply", "c2")
		require.NoError(t, repo.Delete(t.Context(), "c2"))

		threads, _, err := repo.ListForTarget(t.Context(), "article-1", defaultPag())
		require.NoError(t, err)
		require.Len(t, threads[0].Replies, 2)
		assert.True(t, threads[0].Replies[0].IsDeleted)
		assert.False(t, threads[0].Replies[1].IsDeleted)
		assert.Equal(t, "reply to reply", threads[0].Replies[1].Text)
	})

	t.Run("empty target returns empty list", func(t *testing.T) {
		repo, _ := setupCommentRepo(t)

		threads, total, err := repo.ListForTarget(t.Context(), "article-1", defaultPag())
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, threads)
	})
}

func TestCommentRepository_ExistsInTarget(t *testing.T) {
	t.Run("returns true for existing comment in target", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-1", "hello")

		exists, err := repo.ExistsInTarget(t.Context(), "comment-1", "article-1")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for comment in different target", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-1", "hello")

		exists, err := repo.ExistsInTarget(t.Context(), "comment-1", "article-2")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns false for soft-deleted comment", func(t *testing.T) {
		repo, db := setupCommentRepo(t)
		testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-1", "hello")
		require.NoError(t, repo.Delete(t.Context(), "comment-1"))

		exists, err := repo.ExistsInTarget(t.Context(), "comment-1", "article-1")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
