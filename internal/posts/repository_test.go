package posts_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/posts"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func setupPostRepo(t *testing.T) (*posts.Repository, *sql.DB) {
	t.Helper()
	database := testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, database, "user-1")
	testutil.CreateTestUser(t, database, "user-2")
	return posts.NewRepository(database, database), database
}

func TestPostRepository_Create(t *testing.T) {
	repo, _ := setupPostRepo(t)

	p := testutil.CreateTestPost(t, repo, "post-1", "user-1")

	got, err := repo.GetPostByID(t.Context(), p.Id)
	require.NoError(t, err)
	assert.Equal(t, p.Id, got.Id)
	assert.Equal(t, p.Title, got.Title)
	assert.Equal(t, "user-1", got.Author.Id)
	assert.Equal(t, "user-1", got.Author.Username)
	assert.Empty(t, got.Blocks)
	assert.Empty(t, got.Comments)
}

func TestPostRepository_Create_WithBlocks(t *testing.T) {
	t.Run("text block", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		text := "Hello world"
		p := models.Post{
			Id:     "post-1",
			Title:  "Post with text",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.PostBlock{
				{Type: models.PostBlockTypeText, Text: &text},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Blocks, 1)
		assert.Equal(t, models.PostBlockTypeText, got.Blocks[0].Type)
		require.NotNil(t, got.Blocks[0].Text)
		assert.Equal(t, text, *got.Blocks[0].Text)
		assert.Nil(t, got.Blocks[0].Images)
		assert.Nil(t, got.Blocks[0].Recipe)
	})

	t.Run("images block", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		images := []string{"img-1", "img-2"}
		p := models.Post{
			Id:     "post-1",
			Title:  "Post with images",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.PostBlock{
				{Type: models.PostBlockTypeImages, Images: &images},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Blocks, 1)
		assert.Equal(t, models.PostBlockTypeImages, got.Blocks[0].Type)
		require.NotNil(t, got.Blocks[0].Images)
		assert.Equal(t, images, *got.Blocks[0].Images)
	})

	t.Run("recipe block - unavailable when recipe does not exist", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		recipeID := "recipe-1"
		p := models.Post{
			Id:     "post-1",
			Title:  "Post with recipe",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.PostBlock{
				{Type: models.PostBlockTypeRecipe, Recipe: &models.PostRecipeRef{Id: recipeID}},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Blocks, 1)
		assert.Equal(t, models.PostBlockTypeRecipe, got.Blocks[0].Type)
		require.NotNil(t, got.Blocks[0].Recipe)
		assert.Equal(t, recipeID, got.Blocks[0].Recipe.Id)
		assert.Equal(t, models.Unavailable, got.Blocks[0].Recipe.Status)
	})

	t.Run("recipe block - unavailable when recipe is private", func(t *testing.T) {
		repo, db := setupPostRepo(t)
		recipeRepo := recipes.NewRepository(db, db)
		recipe := testutil.CreateTestRecipe(t, recipeRepo, "recipe-1", "user-1")
		recipe.Public = false
		require.NoError(t, recipeRepo.UpsertRecipe(t.Context(), recipe))
		p := models.Post{
			Id:     "post-1",
			Title:  "Post with recipe",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.PostBlock{
				{Type: models.PostBlockTypeRecipe, Recipe: &models.PostRecipeRef{Id: recipe.Id}},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.NotNil(t, got.Blocks[0].Recipe)
		assert.Equal(t, models.Unavailable, got.Blocks[0].Recipe.Status)
	})

	t.Run("recipe block - available when recipe is public", func(t *testing.T) {
		repo, db := setupPostRepo(t)
		recipeRepo := recipes.NewRepository(db, db)
		recipe := testutil.CreateTestRecipe(t, recipeRepo, "recipe-1", "user-1")
		p := models.Post{
			Id:     "post-1",
			Title:  "Post with recipe",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.PostBlock{
				{Type: models.PostBlockTypeRecipe, Recipe: &models.PostRecipeRef{Id: recipe.Id}},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.NotNil(t, got.Blocks[0].Recipe)
		assert.Equal(t, models.Available, got.Blocks[0].Recipe.Status)
	})

	t.Run("preserves block order", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		text1, text2 := "first", "second"
		p := models.Post{
			Id:     "post-1",
			Title:  "Post with ordered blocks",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.PostBlock{
				{Type: models.PostBlockTypeText, Text: &text1},
				{Type: models.PostBlockTypeText, Text: &text2},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Blocks, 2)
		assert.Equal(t, text1, *got.Blocks[0].Text)
		assert.Equal(t, text2, *got.Blocks[1].Text)
	})
}

func TestPostRepository_GetPostByID(t *testing.T) {
	t.Run("returns post with author info", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		p := testutil.CreateTestPost(t, repo, "post-1", "user-1")

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		assert.Equal(t, p.Id, got.Id)
		assert.Equal(t, p.Title, got.Title)
		assert.Equal(t, "user-1", got.Author.Id)
		assert.Equal(t, "user-1", got.Author.Username)
	})

	t.Run("not found returns sql.ErrNoRows", func(t *testing.T) {
		repo, _ := setupPostRepo(t)

		_, err := repo.GetPostByID(t.Context(), "non-existent")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestPostRepository_UpdatePost(t *testing.T) {
	t.Run("updates title and replaces blocks", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		p := testutil.CreateTestPost(t, repo, "post-1", "user-1")

		text := "new text"
		err := repo.UpdatePost(t.Context(), p.Id, "New Title", []models.PostBlock{
			{Type: models.PostBlockTypeText, Text: &text},
		})
		require.NoError(t, err)

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		assert.Equal(t, "New Title", got.Title)
		require.Len(t, got.Blocks, 1)
		assert.Equal(t, text, *got.Blocks[0].Text)
	})

	t.Run("clears blocks when updated with empty list", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		text := "original"
		p := models.Post{
			Id:     "post-1",
			Title:  "Original",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.PostBlock{{Type: models.PostBlockTypeText, Text: &text}},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		err := repo.UpdatePost(t.Context(), p.Id, "Updated", []models.PostBlock{})
		require.NoError(t, err)

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		assert.Empty(t, got.Blocks)
	})
}

func TestPostRepository_DeletePost(t *testing.T) {
	t.Run("deletes existing post", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		p := testutil.CreateTestPost(t, repo, "post-1", "user-1")

		require.NoError(t, repo.DeletePost(t.Context(), p.Id))

		_, err := repo.GetPostByID(t.Context(), p.Id)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("no error on non-existent post", func(t *testing.T) {
		repo, _ := setupPostRepo(t)

		err := repo.DeletePost(t.Context(), "non-existent")
		assert.NoError(t, err)
	})
}

func TestPostRepository_ListPosts(t *testing.T) {
	t.Run("returns posts with author info", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		testutil.CreateTestPost(t, repo, "post-1", "user-1")

		posts, total, err := repo.ListPosts(t.Context(), models.NewPaginationParams(1, 20))
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, posts, 1)
		assert.Equal(t, "user-1", posts[0].Author.Id)
		assert.Equal(t, "user-1", posts[0].Author.Username)
	})

	t.Run("returns empty slices for blocks and comments", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		testutil.CreateTestPost(t, repo, "post-1", "user-1")

		result, _, err := repo.ListPosts(t.Context(), models.NewPaginationParams(1, 20))
		require.NoError(t, err)
		assert.NotNil(t, result[0].Blocks)
		assert.NotNil(t, result[0].Comments)
	})

	t.Run("returns posts in descending order by created_at", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		testutil.CreateTestPost(t, repo, "post-1", "user-1")
		testutil.CreateTestPost(t, repo, "post-2", "user-1")

		result, _, err := repo.ListPosts(t.Context(), models.NewPaginationParams(1, 20))
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "post-2", result[0].Id)
		assert.Equal(t, "post-1", result[1].Id)
	})

	t.Run("respects pagination", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		testutil.CreateTestPost(t, repo, "post-1", "user-1")
		testutil.CreateTestPost(t, repo, "post-2", "user-1")
		testutil.CreateTestPost(t, repo, "post-3", "user-1")

		result, total, err := repo.ListPosts(t.Context(), models.NewPaginationParams(1, 2))
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, result, 2)

		page2, _, err := repo.ListPosts(t.Context(), models.NewPaginationParams(2, 2))
		require.NoError(t, err)
		assert.Len(t, page2, 1)
	})

	t.Run("returns empty when no posts", func(t *testing.T) {
		repo, _ := setupPostRepo(t)

		result, total, err := repo.ListPosts(t.Context(), models.NewPaginationParams(1, 20))
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, result)
	})
}

func TestPostRepository_CreateComment(t *testing.T) {
	t.Run("inserts comment and returns it with author info", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		p := testutil.CreateTestPost(t, repo, "post-1", "user-1")

		comment := models.PostComment{
			Id:     "comment-1",
			Author: models.UserPreview{Id: "user-2"},
			Text:   "Great post!",
		}
		got, err := repo.CreateComment(t.Context(), comment, p.Id)
		require.NoError(t, err)
		assert.Equal(t, "comment-1", got.Id)
		assert.Equal(t, "Great post!", got.Text)
		assert.Equal(t, "user-2", got.Author.Id)
		assert.Equal(t, "user-2", got.Author.Username)
	})

	t.Run("comment appears in post", func(t *testing.T) {
		repo, _ := setupPostRepo(t)
		p := testutil.CreateTestPost(t, repo, "post-1", "user-1")

		comment := models.PostComment{
			Id:     "comment-1",
			Author: models.UserPreview{Id: "user-2"},
			Text:   "Nice!",
		}
		_, err := repo.CreateComment(t.Context(), comment, p.Id)
		require.NoError(t, err)

		got, err := repo.GetPostByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Comments, 1)
		assert.Equal(t, "comment-1", got.Comments[0].Id)
		assert.Equal(t, "Nice!", got.Comments[0].Text)
	})
}

func TestPostRepository_DeleteComment(t *testing.T) {
	t.Run("deletes existing comment", func(t *testing.T) {
		repo, db := setupPostRepo(t)
		testutil.CreateTestPost(t, repo, "post-1", "user-1")
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-2", "hello")

		require.NoError(t, repo.DeleteComment(t.Context(), "comment-1"))

		got, err := repo.GetPostByID(t.Context(), "post-1")
		require.NoError(t, err)
		assert.Empty(t, got.Comments)
	})

	t.Run("non-existent comment returns ErrCommentNotFound", func(t *testing.T) {
		repo, _ := setupPostRepo(t)

		err := repo.DeleteComment(t.Context(), "non-existent")
		assert.ErrorIs(t, err, posts.ErrCommentNotFound)
	})
}

func TestPostRepository_GetCommentAuthorID(t *testing.T) {
	t.Run("returns author id", func(t *testing.T) {
		repo, db := setupPostRepo(t)
		testutil.CreateTestPost(t, repo, "post-1", "user-1")
		testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-2", "hello")

		authorID, err := repo.GetCommentAuthorID(t.Context(), "comment-1")
		require.NoError(t, err)
		assert.Equal(t, "user-2", authorID)
	})

	t.Run("non-existent comment returns sql.ErrNoRows", func(t *testing.T) {
		repo, _ := setupPostRepo(t)

		_, err := repo.GetCommentAuthorID(t.Context(), "non-existent")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestPostRepository_ListPosts_WithComments(t *testing.T) {
	repo, db := setupPostRepo(t)
	testutil.CreateTestPost(t, repo, "post-1", "user-1")
	testutil.CreateTestComment(t, db, "comment-1", "post-1", "user-2", "Nice post!")
	testutil.CreateTestComment(t, db, "comment-2", "post-1", "user-1", "Thanks!")

	result, _, err := repo.ListPosts(t.Context(), models.NewPaginationParams(1, 20))
	require.NoError(t, err)
	require.Len(t, result, 1)

	comments := result[0].Comments
	require.Len(t, comments, 2)
	assert.Equal(t, "comment-1", comments[0].Id)
	assert.Equal(t, "user-2", comments[0].Author.Id)
	assert.Equal(t, "user-2", comments[0].Author.Username)
	assert.Equal(t, "Nice post!", comments[0].Text)
}
