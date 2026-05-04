package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

var defaultArticlePayload = models.CreateArticleRequest{
	Title:  "Test article",
	Blocks: []models.ArticleBlockPayload{},
}

func createArticle(t *testing.T, c *authenticatedClient, payload models.CreateArticleRequest) *models.Article {
	t.Helper()
	resp, err := c.CreateArticleWithResponse(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode())
	return resp.JSON201
}

func TestArticleRepository_UpdateArticle(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path - no blocks", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.CreateArticleWithResponse(t.Context(), defaultArticlePayload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, defaultArticlePayload.Title, resp.JSON201.Title)
		assert.NotEmpty(t, resp.JSON201.Id)
		assert.Equal(t, c.Username, resp.JSON201.Author.Username)
		assert.Empty(t, resp.JSON201.Blocks)
		assert.Empty(t, resp.JSON201.Comments)
		assert.False(t, resp.JSON201.IsLiked)
		assert.Zero(t, resp.JSON201.LikesCount)
	})

	t.Run("happy path - with text block", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		text := "Some text content"
		payload := models.CreateArticleRequest{
			Title: "Article with text",
			Blocks: []models.ArticleBlockPayload{
				{Type: models.ArticleBlockTypeText, Text: &text},
			},
		}
		resp, err := c.CreateArticleWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		require.Len(t, resp.JSON201.Blocks, 1)
		assert.Equal(t, models.ArticleBlockTypeText, resp.JSON201.Blocks[0].Type)
		require.NotNil(t, resp.JSON201.Blocks[0].Text)
		assert.Equal(t, text, *resp.JSON201.Blocks[0].Text)
	})

	t.Run("happy path - with images block", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		images := []string{"img-1", "img-2"}
		payload := models.CreateArticleRequest{
			Title: "Article with images",
			Blocks: []models.ArticleBlockPayload{
				{Type: models.ArticleBlockTypeImages, Images: &images},
			},
		}
		resp, err := c.CreateArticleWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		require.Len(t, resp.JSON201.Blocks, 1)
		assert.Equal(t, models.ArticleBlockTypeImages, resp.JSON201.Blocks[0].Type)
		require.NotNil(t, resp.JSON201.Blocks[0].Images)
		assert.Equal(t, images, *resp.JSON201.Blocks[0].Images)
	})

	t.Run("unauthorized", func(t *testing.T) {
		c := newTestClient(t, srv)
		resp, err := c.CreateArticleWithResponse(t.Context(), defaultArticlePayload)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestGetArticle(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns Article by id", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createArticle(t, c, defaultArticlePayload)

		resp, err := c.GetArticleWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, created.Id, resp.JSON200.Id)
		assert.Equal(t, defaultArticlePayload.Title, resp.JSON200.Title)
		assert.Equal(t, c.Username, resp.JSON200.Author.Username)
	})

	t.Run("populates is_liked", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createArticle(t, c, defaultArticlePayload)

		resp, err := c.GetArticleWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.False(t, resp.JSON200.IsLiked)
		assert.Zero(t, resp.JSON200.LikesCount)
	})

	t.Run("non-existent Article returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.GetArticleWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createArticle(t, c, defaultArticlePayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.GetArticleWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestToggleArticleLike(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("like Article", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		resp, err := c.ToggleArticleLikeWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.True(t, resp.JSON200.Liked)
		assert.Equal(t, int32(1), resp.JSON200.LikesCount)
	})

	t.Run("unlike Article", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		_, err := c.ToggleArticleLikeWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)

		resp, err := c.ToggleArticleLikeWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.False(t, resp.JSON200.Liked)
		assert.Equal(t, int32(0), resp.JSON200.LikesCount)
	})

	t.Run("is_liked reflects in get Article", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		_, err := c.ToggleArticleLikeWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)

		resp, err := c.GetArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		assert.True(t, resp.JSON200.IsLiked)
		assert.Equal(t, int32(1), resp.JSON200.LikesCount)
	})

	t.Run("non-existent Article returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.ToggleArticleLikeWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.ToggleArticleLikeWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestUpdateArticle(t *testing.T) {
	srv := setupTestServer(t)

	updatedPayload := models.UpdateArticleRequest{
		Title:  "Updated Title",
		Blocks: []models.ArticleBlockPayload{},
	}

	t.Run("author can update own Article", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createArticle(t, c, defaultArticlePayload)

		resp, err := c.UpdateArticleWithResponse(t.Context(), created.Id, updatedPayload)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, updatedPayload.Title, resp.JSON200.Title)
		assert.Equal(t, created.Id, resp.JSON200.Id)
	})

	t.Run("updates blocks", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createArticle(t, c, defaultArticlePayload)

		text := "new block"
		payload := models.UpdateArticleRequest{
			Title: "Updated",
			Blocks: []models.ArticleBlockPayload{
				{Type: models.ArticleBlockTypeText, Text: &text},
			},
		}

		resp, err := c.UpdateArticleWithResponse(t.Context(), created.Id, payload)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Blocks, 1)
		assert.Equal(t, text, *resp.JSON200.Blocks[0].Text)
	})

	t.Run("non-author cannot update Article", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		created := createArticle(t, c1, defaultArticlePayload)

		resp, err := c2.UpdateArticleWithResponse(t.Context(), created.Id, updatedPayload)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("non-existent Article returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.UpdateArticleWithResponse(t.Context(), "non-existent-id", updatedPayload)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createArticle(t, c, defaultArticlePayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.UpdateArticleWithResponse(t.Context(), created.Id, updatedPayload)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestDeleteArticle(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("author can delete own Article", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		resp, err := c.DeleteArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("deleted Article no longer appears in feed", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		_, err := c.DeleteArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)

		resp, err := c.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{})
		require.NoError(t, err)
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("non-author delete returns 403", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c1, defaultArticlePayload)

		resp, err := c2.DeleteArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("non-existent Article returns 204", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.DeleteArticleWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("deleting already deleted Article returns 204", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		_, err := c.DeleteArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)

		resp, err := c.DeleteArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.DeleteArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestCreateArticleComment(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("author can comment on own Article", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		resp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "nice Article"})
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, "nice Article", resp.JSON201.Text)
		assert.Equal(t, c.Username, resp.JSON201.Author.Username)
		assert.NotEmpty(t, resp.JSON201.Id)
	})

	t.Run("another user can comment", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c1, defaultArticlePayload)

		resp, err := c2.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "great!"})
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, c2.Username, resp.JSON201.Author.Username)
	})

	t.Run("comment appears in get Article", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		_, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "hello"})
		require.NoError(t, err)

		resp, err := c.GetArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		require.Len(t, resp.JSON200.Comments, 1)
		assert.Equal(t, "hello", resp.JSON200.Comments[0].Text)
	})

	t.Run("reply has parent_id set", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		parentResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "root comment"})
		require.NoError(t, err)
		require.Equal(t, 201, parentResp.StatusCode())

		parentID := parentResp.JSON201.Id
		replyResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{
			Text:     "reply",
			ParentId: &parentID,
		})
		require.NoError(t, err)
		require.Equal(t, 201, replyResp.StatusCode())
		require.NotNil(t, replyResp.JSON201.ParentId)
		assert.Equal(t, parentID, *replyResp.JSON201.ParentId)
	})

	t.Run("reply to non-existent comment returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		nonExistent := "non-existent-comment"
		resp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{
			Text:     "reply",
			ParentId: &nonExistent,
		})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("non-existent Article returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.CreateArticleCommentWithResponse(t.Context(), "non-existent-id", models.CreateCommentRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestDeleteComment(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("author can delete own comment", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		commentResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "hello"})
		require.NoError(t, err)
		require.Equal(t, 201, commentResp.StatusCode())

		resp, err := c.DeleteCommentWithResponse(t.Context(), commentResp.JSON201.Id)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("deleted comment appears as deleted placeholder", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		commentResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "hello"})
		require.NoError(t, err)

		_, err = c.DeleteCommentWithResponse(t.Context(), commentResp.JSON201.Id)
		require.NoError(t, err)

		getResp, err := c.GetArticleWithResponse(t.Context(), Article.Id)
		require.NoError(t, err)
		require.Len(t, getResp.JSON200.Comments, 1)
		assert.True(t, getResp.JSON200.Comments[0].IsDeleted)
		assert.Nil(t, getResp.JSON200.Comments[0].Author)
		assert.Empty(t, getResp.JSON200.Comments[0].Text)
	})

	t.Run("non-author cannot delete comment", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c1, defaultArticlePayload)

		commentResp, err := c1.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "hello"})
		require.NoError(t, err)

		resp, err := c2.DeleteCommentWithResponse(t.Context(), commentResp.JSON201.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("non-existent comment returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.DeleteCommentWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		commentResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "hello"})
		require.NoError(t, err)

		unauth := newTestClient(t, srv)
		resp, err := unauth.DeleteCommentWithResponse(t.Context(), commentResp.JSON201.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestListArticleComments(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns empty page when no comments", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		resp, err := c.ListArticleCommentsWithResponse(t.Context(), Article.Id, &client.ListArticleCommentsParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
		assert.Equal(t, int32(0), resp.JSON200.Pagination.ItemsCount)
	})

	t.Run("returns root comments with replies", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		rootResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "root"})
		require.NoError(t, err)
		require.Equal(t, 201, rootResp.StatusCode())

		_, err = c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{
			Text:     "reply",
			ParentId: &rootResp.JSON201.Id,
		})
		require.NoError(t, err)

		resp, err := c.ListArticleCommentsWithResponse(t.Context(), Article.Id, &client.ListArticleCommentsParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Items, 1)
		assert.Equal(t, "root", resp.JSON200.Items[0].Text)
		require.Len(t, resp.JSON200.Items[0].Replies, 1)
		assert.Equal(t, "reply", resp.JSON200.Items[0].Replies[0].Text)
	})

	t.Run("replies are not returned as root comments", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		rootResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "root"})
		require.NoError(t, err)

		_, err = c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{
			Text:     "reply",
			ParentId: &rootResp.JSON201.Id,
		})
		require.NoError(t, err)

		resp, err := c.ListArticleCommentsWithResponse(t.Context(), Article.Id, &client.ListArticleCommentsParams{})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.JSON200.Pagination.ItemsCount)
	})

	t.Run("respects pagination", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		for i := range 3 {
			_, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{
				Text: fmt.Sprintf("comment %d", i),
			})
			require.NoError(t, err)
		}

		limit := int32(2)
		resp, err := c.ListArticleCommentsWithResponse(t.Context(), Article.Id, &client.ListArticleCommentsParams{Limit: &limit})
		require.NoError(t, err)
		assert.Len(t, resp.JSON200.Items, 2)
		assert.Equal(t, int32(2), resp.JSON200.Pagination.ItemsCount)
		assert.Equal(t, int32(2), resp.JSON200.Pagination.LastPage)
	})

	t.Run("Article not found returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.ListArticleCommentsWithResponse(t.Context(), "non-existent", &client.ListArticleCommentsParams{})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}

func TestUpdateComment(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("author can update own comment", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		commentResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "original"})
		require.NoError(t, err)
		require.Equal(t, 201, commentResp.StatusCode())

		resp, err := c.UpdateCommentWithResponse(t.Context(), commentResp.JSON201.Id, models.UpdateCommentRequest{Text: "updated"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, "updated", resp.JSON200.Text)
		assert.Equal(t, commentResp.JSON201.Id, resp.JSON200.Id)
	})

	t.Run("updated text appears in list", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		commentResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "original"})
		require.NoError(t, err)

		_, err = c.UpdateCommentWithResponse(t.Context(), commentResp.JSON201.Id, models.UpdateCommentRequest{Text: "updated"})
		require.NoError(t, err)

		listResp, err := c.ListArticleCommentsWithResponse(t.Context(), Article.Id, &client.ListArticleCommentsParams{})
		require.NoError(t, err)
		require.Len(t, listResp.JSON200.Items, 1)
		assert.Equal(t, "updated", listResp.JSON200.Items[0].Text)
	})

	t.Run("non-author cannot update comment", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c1, defaultArticlePayload)

		commentResp, err := c1.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "hello"})
		require.NoError(t, err)

		resp, err := c2.UpdateCommentWithResponse(t.Context(), commentResp.JSON201.Id, models.UpdateCommentRequest{Text: "hijack"})
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("non-existent comment returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.UpdateCommentWithResponse(t.Context(), "non-existent-id", models.UpdateCommentRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		commentResp, err := c.CreateArticleCommentWithResponse(t.Context(), Article.Id, models.CreateCommentRequest{Text: "hello"})
		require.NoError(t, err)

		unauth := newTestClient(t, srv)
		resp, err := unauth.UpdateCommentWithResponse(t.Context(), commentResp.JSON201.Id, models.UpdateCommentRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestListArticles(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns empty list", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("returns created Articles", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createArticle(t, c, defaultArticlePayload)

		resp, err := c.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Items, 1)
		assert.Equal(t, defaultArticlePayload.Title, resp.JSON200.Items[0].Title)
		assert.Equal(t, c.Username, resp.JSON200.Items[0].Author.Username)
	})

	t.Run("populates is_liked for current user", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createArticle(t, c, defaultArticlePayload)

		resp, err := c.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.False(t, resp.JSON200.Items[0].IsLiked)
		assert.Zero(t, resp.JSON200.Items[0].LikesCount)
	})

	t.Run("respects pagination", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createArticle(t, c, defaultArticlePayload)
		createArticle(t, c, defaultArticlePayload)
		createArticle(t, c, defaultArticlePayload)

		limit := int32(2)
		resp, err := c.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{Limit: &limit})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
		assert.Equal(t, int32(3), resp.JSON200.Pagination.LastPage)
	})
}

func TestArticleWithImages(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("images in block are confirmed after create", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		imageID := uploadImage(t, c, generateTestImage(t))

		images := []string{imageID}
		payload := models.CreateArticleRequest{
			Title: "Article with images",
			Blocks: []models.ArticleBlockPayload{
				{Type: models.ArticleBlockTypeImages, Images: &images},
			},
		}
		createArticle(t, c, payload)

		resp, err := c.GetImageWithResponse(t.Context(), imageID)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
	})

	t.Run("images in block are confirmed after update", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		Article := createArticle(t, c, defaultArticlePayload)

		imageID := uploadImage(t, c, generateTestImage(t))
		images := []string{imageID}
		updatePayload := models.UpdateArticleRequest{
			Title: Article.Title,
			Blocks: []models.ArticleBlockPayload{
				{Type: models.ArticleBlockTypeImages, Images: &images},
			},
		}
		resp, err := c.UpdateArticleWithResponse(t.Context(), Article.Id, updatePayload)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())

		imgResp, err := c.GetImageWithResponse(t.Context(), imageID)
		require.NoError(t, err)
		assert.Equal(t, 200, imgResp.StatusCode())
	})
}
