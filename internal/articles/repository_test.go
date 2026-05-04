package articles_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/articles"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func setupArticleRepo(t *testing.T) (*articles.Repository, *sql.DB) {
	t.Helper()
	database := testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, database, "user-1")
	testutil.CreateTestUser(t, database, "user-2")
	return articles.NewRepository(database, database), database
}

func TestArticleRepository_Create(t *testing.T) {
	repo, _ := setupArticleRepo(t)

	p := testutil.CreateTestArticle(t, repo, "article-1", "user-1")

	got, err := repo.GetArticleByID(t.Context(), p.Id)
	require.NoError(t, err)
	assert.Equal(t, p.Id, got.Id)
	assert.Equal(t, p.Title, got.Title)
	assert.Equal(t, "user-1", got.Author.Id)
	assert.Equal(t, "user-1", got.Author.Username)
	assert.Empty(t, got.Blocks)
	assert.Empty(t, got.Comments)
}

func TestArticleRepository_Create_WithBlocks(t *testing.T) {
	t.Run("text block", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		text := "Hello world"
		p := models.Article{
			Id:     "article-1",
			Title:  "Article with text",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.ArticleBlock{
				{Type: models.ArticleBlockTypeText, Text: &text},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Blocks, 1)
		assert.Equal(t, models.ArticleBlockTypeText, got.Blocks[0].Type)
		require.NotNil(t, got.Blocks[0].Text)
		assert.Equal(t, text, *got.Blocks[0].Text)
		assert.Nil(t, got.Blocks[0].Images)
		assert.Nil(t, got.Blocks[0].Recipe)
	})

	t.Run("images block", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		images := []string{"img-1", "img-2"}
		p := models.Article{
			Id:     "article-1",
			Title:  "Article with images",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.ArticleBlock{
				{Type: models.ArticleBlockTypeImages, Images: &images},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Blocks, 1)
		assert.Equal(t, models.ArticleBlockTypeImages, got.Blocks[0].Type)
		require.NotNil(t, got.Blocks[0].Images)
		assert.Equal(t, images, *got.Blocks[0].Images)
	})

	t.Run("recipe block - unavailable when recipe does not exist", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		recipeID := "recipe-1"
		p := models.Article{
			Id:     "article-1",
			Title:  "Article with recipe",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.ArticleBlock{
				{Type: models.ArticleBlockTypeRecipe, Recipe: &models.ArticleRecipeRef{Id: recipeID}},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Blocks, 1)
		assert.Equal(t, models.ArticleBlockTypeRecipe, got.Blocks[0].Type)
		require.NotNil(t, got.Blocks[0].Recipe)
		assert.Equal(t, recipeID, got.Blocks[0].Recipe.Id)
		assert.Equal(t, models.ArticleRecipeRefStatusUnavailable, got.Blocks[0].Recipe.Status)
	})

	t.Run("recipe block - unavailable when recipe is private", func(t *testing.T) {
		repo, db := setupArticleRepo(t)
		recipeRepo := recipes.NewRepository(db, db)
		recipe := testutil.CreateTestRecipe(t, recipeRepo, "recipe-1", "user-1")
		recipe.Public = false
		require.NoError(t, recipeRepo.UpsertRecipe(t.Context(), recipe))
		p := models.Article{
			Id:     "article-1",
			Title:  "Article with recipe",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.ArticleBlock{
				{Type: models.ArticleBlockTypeRecipe, Recipe: &models.ArticleRecipeRef{Id: recipe.Id}},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.NotNil(t, got.Blocks[0].Recipe)
		assert.Equal(t, models.ArticleRecipeRefStatusUnavailable, got.Blocks[0].Recipe.Status)
	})

	t.Run("recipe block - available when recipe is public", func(t *testing.T) {
		repo, db := setupArticleRepo(t)
		recipeRepo := recipes.NewRepository(db, db)
		recipe := testutil.CreateTestRecipe(t, recipeRepo, "recipe-1", "user-1")
		p := models.Article{
			Id:     "article-1",
			Title:  "Article with recipe",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.ArticleBlock{
				{Type: models.ArticleBlockTypeRecipe, Recipe: &models.ArticleRecipeRef{Id: recipe.Id}},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.NotNil(t, got.Blocks[0].Recipe)
		assert.Equal(t, models.ArticleRecipeRefStatusAvailable, got.Blocks[0].Recipe.Status)
	})

	t.Run("preserves block order", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		text1, text2 := "first", "second"
		p := models.Article{
			Id:     "article-1",
			Title:  "Article with ordered blocks",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.ArticleBlock{
				{Type: models.ArticleBlockTypeText, Text: &text1},
				{Type: models.ArticleBlockTypeText, Text: &text2},
			},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		require.Len(t, got.Blocks, 2)
		assert.Equal(t, text1, *got.Blocks[0].Text)
		assert.Equal(t, text2, *got.Blocks[1].Text)
	})
}

func TestArticleRepository_GetArticleByID(t *testing.T) {
	t.Run("returns article with author info", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		p := testutil.CreateTestArticle(t, repo, "article-1", "user-1")

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		assert.Equal(t, p.Id, got.Id)
		assert.Equal(t, p.Title, got.Title)
		assert.Equal(t, "user-1", got.Author.Id)
		assert.Equal(t, "user-1", got.Author.Username)
	})

	t.Run("not found returns sql.ErrNoRows", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)

		_, err := repo.GetArticleByID(t.Context(), "non-existent")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestArticleRepository_UpdateArticle(t *testing.T) {
	t.Run("updates title and replaces blocks", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		p := testutil.CreateTestArticle(t, repo, "article-1", "user-1")

		text := "new text"
		err := repo.UpdateArticle(t.Context(), p.Id, "New Title", []models.ArticleBlock{
			{Type: models.ArticleBlockTypeText, Text: &text},
		})
		require.NoError(t, err)

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		assert.Equal(t, "New Title", got.Title)
		require.Len(t, got.Blocks, 1)
		assert.Equal(t, text, *got.Blocks[0].Text)
	})

	t.Run("clears blocks when updated with empty list", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		text := "original"
		p := models.Article{
			Id:     "article-1",
			Title:  "Original",
			Author: models.UserPreview{Id: "user-1"},
			Blocks: []models.ArticleBlock{{Type: models.ArticleBlockTypeText, Text: &text}},
		}
		require.NoError(t, repo.Create(t.Context(), p))

		err := repo.UpdateArticle(t.Context(), p.Id, "Updated", []models.ArticleBlock{})
		require.NoError(t, err)

		got, err := repo.GetArticleByID(t.Context(), p.Id)
		require.NoError(t, err)
		assert.Empty(t, got.Blocks)
	})
}

func TestArticleRepository_DeleteArticle(t *testing.T) {
	t.Run("deletes existing article", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		p := testutil.CreateTestArticle(t, repo, "article-1", "user-1")

		require.NoError(t, repo.DeleteArticle(t.Context(), p.Id))

		_, err := repo.GetArticleByID(t.Context(), p.Id)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("no error on non-existent article", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)

		err := repo.DeleteArticle(t.Context(), "non-existent")
		assert.NoError(t, err)
	})
}

func TestArticleRepository_ListArticles(t *testing.T) {
	t.Run("returns articles with author info", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		testutil.CreateTestArticle(t, repo, "article-1", "user-1")

		articles, total, err := repo.ListArticles(t.Context(), articles.ListArticlesParams{Page: new(int32(1)), Limit: new(int32(20))})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, articles, 1)
		assert.Equal(t, "user-1", articles[0].Author.Id)
		assert.Equal(t, "user-1", articles[0].Author.Username)
	})

	t.Run("returns empty slices for blocks and comments", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		testutil.CreateTestArticle(t, repo, "article-1", "user-1")

		result, _, err := repo.ListArticles(t.Context(), articles.ListArticlesParams{Page: new(int32(1)), Limit: new(int32(20))})
		require.NoError(t, err)
		assert.NotNil(t, result[0].Blocks)
		assert.NotNil(t, result[0].Comments)
	})

	t.Run("returns articles in descending order by created_at", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		testutil.CreateTestArticle(t, repo, "article-1", "user-1")
		testutil.CreateTestArticle(t, repo, "article-2", "user-1")

		result, _, err := repo.ListArticles(t.Context(), articles.ListArticlesParams{Page: new(int32(1)), Limit: new(int32(20))})
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "article-2", result[0].Id)
		assert.Equal(t, "article-1", result[1].Id)
	})

	t.Run("respects pagination", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)
		testutil.CreateTestArticle(t, repo, "article-1", "user-1")
		testutil.CreateTestArticle(t, repo, "article-2", "user-1")
		testutil.CreateTestArticle(t, repo, "article-3", "user-1")

		result, total, err := repo.ListArticles(t.Context(), articles.ListArticlesParams{Page: new(int32(1)), Limit: new(int32(2))})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, result, 2)

		page2, _, err := repo.ListArticles(t.Context(), articles.ListArticlesParams{Page: new(int32(2)), Limit: new(int32(2))})
		require.NoError(t, err)
		assert.Len(t, page2, 1)
	})

	t.Run("returns empty when no articles", func(t *testing.T) {
		repo, _ := setupArticleRepo(t)

		result, total, err := repo.ListArticles(t.Context(), articles.ListArticlesParams{Page: new(int32(1)), Limit: new(int32(20))})
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, result)
	})
}

func TestArticleRepository_ListArticles_WithComments(t *testing.T) {
	repo, db := setupArticleRepo(t)
	testutil.CreateTestArticle(t, repo, "article-1", "user-1")
	testutil.CreateTestComment(t, db, "comment-1", "article-1", "user-2", "Nice article!")
	testutil.CreateTestComment(t, db, "comment-2", "article-1", "user-1", "Thanks!")

	result, _, err := repo.ListArticles(t.Context(), articles.ListArticlesParams{Page: new(int32(1)), Limit: new(int32(20))})
	require.NoError(t, err)
	require.Len(t, result, 1)

	comments := result[0].Comments
	require.Len(t, comments, 2)
	assert.Equal(t, "comment-1", comments[0].Id)
	assert.Equal(t, "user-2", comments[0].Author.Id)
	assert.Equal(t, "user-2", comments[0].Author.Username)
	assert.Equal(t, "Nice article!", comments[0].Text)
}
