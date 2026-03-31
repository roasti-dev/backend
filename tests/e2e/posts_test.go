package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

var defaultPostPayload = models.CreatePostRequest{
	Title:  "Test Post",
	Blocks: []models.PostBlockPayload{},
}

func createPost(t *testing.T, c *authenticatedClient, payload models.CreatePostRequest) *models.Post {
	t.Helper()
	resp, err := c.CreatePostWithResponse(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode())
	return resp.JSON201
}

func TestCreatePost(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path - no blocks", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.CreatePostWithResponse(t.Context(), defaultPostPayload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, defaultPostPayload.Title, resp.JSON201.Title)
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
		payload := models.CreatePostRequest{
			Title: "Post with text",
			Blocks: []models.PostBlockPayload{
				{Type: models.PostBlockTypeText, Text: &text},
			},
		}
		resp, err := c.CreatePostWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		require.Len(t, resp.JSON201.Blocks, 1)
		assert.Equal(t, models.PostBlockTypeText, resp.JSON201.Blocks[0].Type)
		require.NotNil(t, resp.JSON201.Blocks[0].Text)
		assert.Equal(t, text, *resp.JSON201.Blocks[0].Text)
	})

	t.Run("happy path - with images block", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		images := []string{"img-1", "img-2"}
		payload := models.CreatePostRequest{
			Title: "Post with images",
			Blocks: []models.PostBlockPayload{
				{Type: models.PostBlockTypeImages, Images: &images},
			},
		}
		resp, err := c.CreatePostWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		require.Len(t, resp.JSON201.Blocks, 1)
		assert.Equal(t, models.PostBlockTypeImages, resp.JSON201.Blocks[0].Type)
		require.NotNil(t, resp.JSON201.Blocks[0].Images)
		assert.Equal(t, images, *resp.JSON201.Blocks[0].Images)
	})

	t.Run("unauthorized", func(t *testing.T) {
		c := newTestClient(t, srv)
		resp, err := c.CreatePostWithResponse(t.Context(), defaultPostPayload)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestGetPost(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns post by id", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createPost(t, c, defaultPostPayload)

		resp, err := c.GetPostWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, created.Id, resp.JSON200.Id)
		assert.Equal(t, defaultPostPayload.Title, resp.JSON200.Title)
		assert.Equal(t, c.Username, resp.JSON200.Author.Username)
	})

	t.Run("populates is_liked", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createPost(t, c, defaultPostPayload)

		resp, err := c.GetPostWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.False(t, resp.JSON200.IsLiked)
		assert.Zero(t, resp.JSON200.LikesCount)
	})

	t.Run("non-existent post returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.GetPostWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createPost(t, c, defaultPostPayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.GetPostWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestTogglePostLike(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("like post", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		resp, err := c.TogglePostLikeWithResponse(t.Context(), post.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.True(t, resp.JSON200.Liked)
		assert.Equal(t, int32(1), resp.JSON200.LikesCount)
	})

	t.Run("unlike post", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		_, err := c.TogglePostLikeWithResponse(t.Context(), post.Id)
		require.NoError(t, err)

		resp, err := c.TogglePostLikeWithResponse(t.Context(), post.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.False(t, resp.JSON200.Liked)
		assert.Equal(t, int32(0), resp.JSON200.LikesCount)
	})

	t.Run("is_liked reflects in get post", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		_, err := c.TogglePostLikeWithResponse(t.Context(), post.Id)
		require.NoError(t, err)

		resp, err := c.GetPostWithResponse(t.Context(), post.Id)
		require.NoError(t, err)
		assert.True(t, resp.JSON200.IsLiked)
		assert.Equal(t, int32(1), resp.JSON200.LikesCount)
	})

	t.Run("non-existent post returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.TogglePostLikeWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.TogglePostLikeWithResponse(t.Context(), post.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestUpdatePost(t *testing.T) {
	srv := setupTestServer(t)

	updatedPayload := models.UpdatePostRequest{
		Title:  "Updated Title",
		Blocks: []models.PostBlockPayload{},
	}

	t.Run("author can update own post", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createPost(t, c, defaultPostPayload)

		resp, err := c.UpdatePostWithResponse(t.Context(), created.Id, updatedPayload)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, updatedPayload.Title, resp.JSON200.Title)
		assert.Equal(t, created.Id, resp.JSON200.Id)
	})

	t.Run("updates blocks", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createPost(t, c, defaultPostPayload)

		text := "new block"
		payload := models.UpdatePostRequest{
			Title: "Updated",
			Blocks: []models.PostBlockPayload{
				{Type: models.PostBlockTypeText, Text: &text},
			},
		}

		resp, err := c.UpdatePostWithResponse(t.Context(), created.Id, payload)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Blocks, 1)
		assert.Equal(t, text, *resp.JSON200.Blocks[0].Text)
	})

	t.Run("non-author cannot update post", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		created := createPost(t, c1, defaultPostPayload)

		resp, err := c2.UpdatePostWithResponse(t.Context(), created.Id, updatedPayload)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("non-existent post returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.UpdatePostWithResponse(t.Context(), "non-existent-id", updatedPayload)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createPost(t, c, defaultPostPayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.UpdatePostWithResponse(t.Context(), created.Id, updatedPayload)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestDeletePost(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("author can delete own post", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		resp, err := c.DeletePostWithResponse(t.Context(), post.Id)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("deleted post no longer appears in feed", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		_, err := c.DeletePostWithResponse(t.Context(), post.Id)
		require.NoError(t, err)

		resp, err := c.ListPostsWithResponse(t.Context(), &client.ListPostsParams{})
		require.NoError(t, err)
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("non-author cannot delete post", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c1, defaultPostPayload)

		resp, err := c2.DeletePostWithResponse(t.Context(), post.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("delete non-existent post returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.DeletePostWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.DeletePostWithResponse(t.Context(), post.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestCreatePostComment(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("author can comment on own post", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		resp, err := c.CreatePostCommentWithResponse(t.Context(), post.Id, models.CreatePostCommentRequest{Text: "nice post"})
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, "nice post", resp.JSON201.Text)
		assert.Equal(t, c.Username, resp.JSON201.Author.Username)
		assert.NotEmpty(t, resp.JSON201.Id)
	})

	t.Run("another user can comment", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c1, defaultPostPayload)

		resp, err := c2.CreatePostCommentWithResponse(t.Context(), post.Id, models.CreatePostCommentRequest{Text: "great!"})
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, c2.Username, resp.JSON201.Author.Username)
	})

	t.Run("comment appears in get post", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		_, err := c.CreatePostCommentWithResponse(t.Context(), post.Id, models.CreatePostCommentRequest{Text: "hello"})
		require.NoError(t, err)

		resp, err := c.GetPostWithResponse(t.Context(), post.Id)
		require.NoError(t, err)
		require.Len(t, resp.JSON200.Comments, 1)
		assert.Equal(t, "hello", resp.JSON200.Comments[0].Text)
	})

	t.Run("non-existent post returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.CreatePostCommentWithResponse(t.Context(), "non-existent-id", models.CreatePostCommentRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		post := createPost(t, c, defaultPostPayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.CreatePostCommentWithResponse(t.Context(), post.Id, models.CreatePostCommentRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestListPosts(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns empty list", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.ListPostsWithResponse(t.Context(), &client.ListPostsParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("returns created posts", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createPost(t, c, defaultPostPayload)

		resp, err := c.ListPostsWithResponse(t.Context(), &client.ListPostsParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Items, 1)
		assert.Equal(t, defaultPostPayload.Title, resp.JSON200.Items[0].Title)
		assert.Equal(t, c.Username, resp.JSON200.Items[0].Author.Username)
	})

	t.Run("populates is_liked for current user", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createPost(t, c, defaultPostPayload)

		resp, err := c.ListPostsWithResponse(t.Context(), &client.ListPostsParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.False(t, resp.JSON200.Items[0].IsLiked)
		assert.Zero(t, resp.JSON200.Items[0].LikesCount)
	})

	t.Run("respects pagination", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createPost(t, c, defaultPostPayload)
		createPost(t, c, defaultPostPayload)
		createPost(t, c, defaultPostPayload)

		limit := int32(2)
		resp, err := c.ListPostsWithResponse(t.Context(), &client.ListPostsParams{Limit: &limit})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
		assert.Equal(t, int32(3), resp.JSON200.Pagination.LastPage)
	})
}
