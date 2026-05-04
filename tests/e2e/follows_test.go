package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func followUser(t *testing.T, c *authenticatedClient, userID string) {
	t.Helper()
	resp, err := c.FollowUserWithResponse(t.Context(), userID)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())
}

func unfollowUser(t *testing.T, c *authenticatedClient, userID string) {
	t.Helper()
	resp, err := c.UnfollowUserWithResponse(t.Context(), userID)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode())
}

func TestFollowUser(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("follow and unfollow", func(t *testing.T) {
		follower := newAuthenticatedTestClient(t, srv)
		target := newAuthenticatedTestClient(t, srv)

		followUser(t, follower, target.ID)

		// profile shows is_following
		resp, err := follower.GetUserProfileWithResponse(t.Context(), target.Username)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.True(t, resp.JSON200.IsFollowing)
		assert.Equal(t, int32(1), resp.JSON200.FollowersCount)

		unfollowUser(t, follower, target.ID)

		resp2, err := follower.GetUserProfileWithResponse(t.Context(), target.Username)
		require.NoError(t, err)
		assert.False(t, resp2.JSON200.IsFollowing)
		assert.Equal(t, int32(0), resp2.JSON200.FollowersCount)
	})

	t.Run("follow is idempotent", func(t *testing.T) {
		follower := newAuthenticatedTestClient(t, srv)
		target := newAuthenticatedTestClient(t, srv)

		followUser(t, follower, target.ID)
		followUser(t, follower, target.ID)

		resp, err := follower.GetUserProfileWithResponse(t.Context(), target.Username)
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.JSON200.FollowersCount)
	})

	t.Run("unfollow is idempotent", func(t *testing.T) {
		follower := newAuthenticatedTestClient(t, srv)
		target := newAuthenticatedTestClient(t, srv)

		unfollowUser(t, follower, target.ID)
		unfollowUser(t, follower, target.ID)
	})

	t.Run("cannot follow self", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.FollowUserWithResponse(t.Context(), c.ID)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode())
	})

	t.Run("follow unknown user returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.FollowUserWithResponse(t.Context(), "nonexistent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		target := newAuthenticatedTestClient(t, srv)
		unauth := newTestClient(t, srv)
		resp, err := unauth.FollowUserWithResponse(t.Context(), target.ID)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestFollowStats(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("is_followed is true when target follows back", func(t *testing.T) {
		a := newAuthenticatedTestClient(t, srv)
		b := newAuthenticatedTestClient(t, srv)

		followUser(t, a, b.ID)
		followUser(t, b, a.ID)

		// a sees is_followed=true on b's profile
		resp, err := a.GetUserProfileWithResponse(t.Context(), b.Username)
		require.NoError(t, err)
		assert.True(t, resp.JSON200.IsFollowing)
		assert.True(t, resp.JSON200.IsFollowed)
	})

	t.Run("following_count increases when user follows others", func(t *testing.T) {
		follower := newAuthenticatedTestClient(t, srv)
		t1 := newAuthenticatedTestClient(t, srv)
		t2 := newAuthenticatedTestClient(t, srv)

		followUser(t, follower, t1.ID)
		followUser(t, follower, t2.ID)

		resp, err := follower.GetUserProfileWithResponse(t.Context(), follower.Username)
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.JSON200.FollowingCount)
	})

	t.Run("anonymous sees zero flags", func(t *testing.T) {
		target := newAuthenticatedTestClient(t, srv)
		anon := newTestClient(t, srv)

		resp, err := anon.GetUserProfileWithResponse(t.Context(), target.Username)
		require.NoError(t, err)
		assert.False(t, resp.JSON200.IsFollowing)
		assert.False(t, resp.JSON200.IsFollowed)
	})
}

func TestListFollowingAndFollowers(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("lists following users", func(t *testing.T) {
		follower := newAuthenticatedTestClient(t, srv)
		t1 := newAuthenticatedTestClient(t, srv)
		t2 := newAuthenticatedTestClient(t, srv)

		followUser(t, follower, t1.ID)
		followUser(t, follower, t2.ID)

		resp, err := follower.ListFollowingWithResponse(t.Context(), &client.ListFollowingParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
	})

	t.Run("lists followers", func(t *testing.T) {
		target := newAuthenticatedTestClient(t, srv)
		f1 := newAuthenticatedTestClient(t, srv)
		f2 := newAuthenticatedTestClient(t, srv)

		followUser(t, f1, target.ID)
		followUser(t, f2, target.ID)

		resp, err := target.ListFollowersWithResponse(t.Context(), &client.ListFollowersParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
	})

	t.Run("unauth returns 401", func(t *testing.T) {
		unauth := newTestClient(t, srv)

		resp, err := unauth.ListFollowingWithResponse(t.Context(), &client.ListFollowingParams{})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestFollowingArticlesFeed(t *testing.T) {
	srv := setupTestServer(t)

	filterFollowing := client.Following

	t.Run("returns articles from followed users", func(t *testing.T) {
		follower := newAuthenticatedTestClient(t, srv)
		author := newAuthenticatedTestClient(t, srv)

		followUser(t, follower, author.ID)
		createArticle(t, author, defaultArticlePayload)
		createArticle(t, author, defaultArticlePayload)

		resp, err := follower.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{
			Filter: &filterFollowing,
		})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
	})

	t.Run("does not return articles from non-followed users", func(t *testing.T) {
		follower := newAuthenticatedTestClient(t, srv)
		other := newAuthenticatedTestClient(t, srv)

		createArticle(t, other, defaultArticlePayload)

		resp, err := follower.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{
			Filter: &filterFollowing,
		})
		require.NoError(t, err)
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("empty when not following anyone", func(t *testing.T) {
		follower := newAuthenticatedTestClient(t, srv)

		resp, err := follower.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{
			Filter: &filterFollowing,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		unauth := newTestClient(t, srv)
		resp, err := unauth.ListArticlesWithResponse(t.Context(), &client.ListArticlesParams{
			Filter: &filterFollowing,
		})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}
