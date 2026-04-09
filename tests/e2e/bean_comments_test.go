package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func TestCreateBeanComment(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("can comment on a bean", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		resp, err := c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{Text: "great bean!"})
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, "great bean!", resp.JSON201.Text)
		assert.Equal(t, c.Username, resp.JSON201.Author.Username)
		assert.NotEmpty(t, resp.JSON201.Id)
	})

	t.Run("another user can comment", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c1, defaultBeanPayload)

		resp, err := c2.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{Text: "nice!"})
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, c2.Username, resp.JSON201.Author.Username)
	})

	t.Run("reply has parent_id set", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		parentResp, err := c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{Text: "root comment"})
		require.NoError(t, err)
		require.Equal(t, 201, parentResp.StatusCode())

		parentID := parentResp.JSON201.Id
		replyResp, err := c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{
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
		bean := createBean(t, c, defaultBeanPayload)

		nonExistent := "non-existent-comment"
		resp, err := c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{
			Text:     "reply",
			ParentId: &nonExistent,
		})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("non-existent bean returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.CreateBeanCommentWithResponse(t.Context(), "non-existent-id", models.CreatePostCommentRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestListBeanComments(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns empty page when no comments", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		resp, err := c.ListBeanCommentsWithResponse(t.Context(), bean.Id, &client.ListBeanCommentsParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
		assert.Equal(t, int32(0), resp.JSON200.Pagination.ItemsCount)
	})

	t.Run("returns root comments with replies", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		rootResp, err := c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{Text: "root"})
		require.NoError(t, err)
		require.Equal(t, 201, rootResp.StatusCode())

		_, err = c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{
			Text:     "reply",
			ParentId: &rootResp.JSON201.Id,
		})
		require.NoError(t, err)

		resp, err := c.ListBeanCommentsWithResponse(t.Context(), bean.Id, &client.ListBeanCommentsParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Items, 1)
		assert.Equal(t, "root", resp.JSON200.Items[0].Text)
		require.Len(t, resp.JSON200.Items[0].Replies, 1)
		assert.Equal(t, "reply", resp.JSON200.Items[0].Replies[0].Text)
	})

	t.Run("replies are not counted as root comments", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		rootResp, err := c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{Text: "root"})
		require.NoError(t, err)

		_, err = c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{
			Text:     "reply",
			ParentId: &rootResp.JSON201.Id,
		})
		require.NoError(t, err)

		resp, err := c.ListBeanCommentsWithResponse(t.Context(), bean.Id, &client.ListBeanCommentsParams{})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.JSON200.Pagination.ItemsCount)
	})

	t.Run("respects pagination", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		for i := range 3 {
			_, err := c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{
				Text: fmt.Sprintf("comment %d", i),
			})
			require.NoError(t, err)
		}

		limit := int32(2)
		resp, err := c.ListBeanCommentsWithResponse(t.Context(), bean.Id, &client.ListBeanCommentsParams{Limit: &limit})
		require.NoError(t, err)
		assert.Len(t, resp.JSON200.Items, 2)
		assert.Equal(t, int32(2), resp.JSON200.Pagination.ItemsCount)
		assert.Equal(t, int32(2), resp.JSON200.Pagination.LastPage)
	})

	t.Run("deleted comment appears as placeholder", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		commentResp, err := c.CreateBeanCommentWithResponse(t.Context(), bean.Id, models.CreatePostCommentRequest{Text: "hello"})
		require.NoError(t, err)

		_, err = c.DeleteCommentWithResponse(t.Context(), commentResp.JSON201.Id)
		require.NoError(t, err)

		listResp, err := c.ListBeanCommentsWithResponse(t.Context(), bean.Id, &client.ListBeanCommentsParams{})
		require.NoError(t, err)
		require.Len(t, listResp.JSON200.Items, 1)
		assert.True(t, listResp.JSON200.Items[0].IsDeleted)
		assert.Nil(t, listResp.JSON200.Items[0].Author)
		assert.Empty(t, listResp.JSON200.Items[0].Text)
	})

	t.Run("bean not found returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.ListBeanCommentsWithResponse(t.Context(), "non-existent", &client.ListBeanCommentsParams{})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}
