package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

// waitForUnreadCount polls until the unread notification count reaches the expected value.
// Needed because notifications are created asynchronously via the event bus.
func waitForUnreadCount(t *testing.T, c *authenticatedClient, expected int) {
	t.Helper()
	require.Eventually(t, func() bool {
		resp, err := c.GetNotificationUnreadCountWithResponse(t.Context())
		return err == nil && resp.JSON200 != nil && int(resp.JSON200.UnreadCount) == expected
	}, 500*time.Millisecond, 10*time.Millisecond, "unread count did not reach %d", expected)
}

func TestListNotifications(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns empty list when no notifications", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
		assert.Equal(t, int32(0), resp.JSON200.Pagination.ItemsCount)
	})

	t.Run("recipe like creates notification for author", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		liker := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, author, defaultPayload)

		_, err := liker.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)

		waitForUnreadCount(t, author, 1)

		resp, err := author.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Items, 1)

		n := resp.JSON200.Items[0]
		assert.Equal(t, models.LikeRecipe, n.Type)
		assert.Equal(t, recipe.Id, n.EntityId)
		assert.Equal(t, liker.Username, n.Actor.Username)
		assert.Nil(t, n.ReadAt)
	})

	t.Run("post like creates notification for author", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		liker := newAuthenticatedTestClient(t, srv)
		post := createPost(t, author, defaultPostPayload)

		_, err := liker.TogglePostLikeWithResponse(t.Context(), post.Id)
		require.NoError(t, err)

		waitForUnreadCount(t, author, 1)

		resp, err := author.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{})
		require.NoError(t, err)
		require.Len(t, resp.JSON200.Items, 1)

		n := resp.JSON200.Items[0]
		assert.Equal(t, models.LikePost, n.Type)
		assert.Equal(t, post.Id, n.EntityId)
		assert.Equal(t, liker.Username, n.Actor.Username)
	})

	t.Run("recipe comment creates notification for author", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		commenter := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, author, defaultPayload)

		_, err := commenter.CreateRecipeCommentWithResponse(t.Context(), recipe.Id, models.CreatePostCommentRequest{Text: "great recipe!"})
		require.NoError(t, err)

		waitForUnreadCount(t, author, 1)

		resp, err := author.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{})
		require.NoError(t, err)
		require.Len(t, resp.JSON200.Items, 1)

		n := resp.JSON200.Items[0]
		assert.Equal(t, models.CommentRecipe, n.Type)
		assert.Equal(t, recipe.Id, n.EntityId)
		assert.Equal(t, commenter.Username, n.Actor.Username)
	})

	t.Run("post comment creates notification for author", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		commenter := newAuthenticatedTestClient(t, srv)
		post := createPost(t, author, defaultPostPayload)

		_, err := commenter.CreatePostCommentWithResponse(t.Context(), post.Id, models.CreatePostCommentRequest{Text: "nice post!"})
		require.NoError(t, err)

		waitForUnreadCount(t, author, 1)

		resp, err := author.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{})
		require.NoError(t, err)
		require.Len(t, resp.JSON200.Items, 1)

		n := resp.JSON200.Items[0]
		assert.Equal(t, models.CommentPost, n.Type)
		assert.Equal(t, post.Id, n.EntityId)
		assert.Equal(t, commenter.Username, n.Actor.Username)
	})

	t.Run("self-like does not create notification", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		_, err := c.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)

		// Wait briefly and assert count stays at 0.
		time.Sleep(100 * time.Millisecond)
		resp, err := c.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int32(0), resp.JSON200.UnreadCount)
	})

	t.Run("self-comment does not create notification", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		_, err := c.CreateRecipeCommentWithResponse(t.Context(), recipe.Id, models.CreatePostCommentRequest{Text: "my own comment"})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		resp, err := c.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int32(0), resp.JSON200.UnreadCount)
	})

	t.Run("unlike does not create notification", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		liker := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, author, defaultPayload)

		// Like then unlike.
		_, err := liker.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		waitForUnreadCount(t, author, 1)

		_, err = liker.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)

		// Count should stay at 1 (the like notification), not increase.
		time.Sleep(100 * time.Millisecond)
		resp, err := author.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.JSON200.UnreadCount)
	})

	t.Run("notifications sorted newest first", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		u1 := newAuthenticatedTestClient(t, srv)
		u2 := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, author, defaultPayload)

		_, err := u1.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		waitForUnreadCount(t, author, 1)

		_, err = u2.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		waitForUnreadCount(t, author, 2)

		resp, err := author.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{})
		require.NoError(t, err)
		require.Len(t, resp.JSON200.Items, 2)
		assert.True(t, resp.JSON200.Items[0].CreatedAt.After(resp.JSON200.Items[1].CreatedAt) ||
			resp.JSON200.Items[0].CreatedAt.Equal(resp.JSON200.Items[1].CreatedAt))
	})

	t.Run("respects pagination", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, author, defaultPayload)

		for range 3 {
			liker := newAuthenticatedTestClient(t, srv)
			_, err := liker.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
			require.NoError(t, err)
		}
		waitForUnreadCount(t, author, 3)

		limit := int32(2)
		resp, err := author.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{Limit: &limit})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
		assert.Equal(t, int32(2), resp.JSON200.Pagination.ItemsCount)
		assert.Equal(t, int32(2), resp.JSON200.Pagination.LastPage)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newTestClient(t, srv)
		resp, err := c.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestGetNotificationUnreadCount(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns 0 when no notifications", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, int32(0), resp.JSON200.UnreadCount)
	})

	t.Run("returns correct count after activity", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, author, defaultPayload)

		u1 := newAuthenticatedTestClient(t, srv)
		u2 := newAuthenticatedTestClient(t, srv)
		_, err := u1.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		_, err = u2.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)

		waitForUnreadCount(t, author, 2)

		resp, err := author.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.JSON200.UnreadCount)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newTestClient(t, srv)
		resp, err := c.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestMarkAllNotificationsRead(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("marks all notifications as read", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, author, defaultPayload)

		liker := newAuthenticatedTestClient(t, srv)
		_, err := liker.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		waitForUnreadCount(t, author, 1)

		markResp, err := author.MarkAllNotificationsReadWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 204, markResp.StatusCode())

		countResp, err := author.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int32(0), countResp.JSON200.UnreadCount)
	})

	t.Run("read_at is set after marking as read", func(t *testing.T) {
		author := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, author, defaultPayload)

		liker := newAuthenticatedTestClient(t, srv)
		_, err := liker.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		waitForUnreadCount(t, author, 1)

		_, err = author.MarkAllNotificationsReadWithResponse(t.Context())
		require.NoError(t, err)

		listResp, err := author.ListNotificationsWithResponse(t.Context(), &client.ListNotificationsParams{})
		require.NoError(t, err)
		require.Len(t, listResp.JSON200.Items, 1)
		assert.NotNil(t, listResp.JSON200.Items[0].ReadAt)
	})

	t.Run("only affects current user notifications", func(t *testing.T) {
		author1 := newAuthenticatedTestClient(t, srv)
		author2 := newAuthenticatedTestClient(t, srv)
		liker := newAuthenticatedTestClient(t, srv)

		recipe1 := createRecipe(t, author1, defaultPayload)
		recipe2 := createRecipe(t, author2, defaultPayload)

		_, err := liker.ToggleRecipeLikeWithResponse(t.Context(), recipe1.Id)
		require.NoError(t, err)
		_, err = liker.ToggleRecipeLikeWithResponse(t.Context(), recipe2.Id)
		require.NoError(t, err)

		waitForUnreadCount(t, author1, 1)
		waitForUnreadCount(t, author2, 1)

		_, err = author1.MarkAllNotificationsReadWithResponse(t.Context())
		require.NoError(t, err)

		count1, err := author1.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int32(0), count1.JSON200.UnreadCount)

		count2, err := author2.GetNotificationUnreadCountWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int32(1), count2.JSON200.UnreadCount)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newTestClient(t, srv)
		resp, err := c.MarkAllNotificationsReadWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}
