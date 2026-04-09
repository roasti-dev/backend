package beans_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/beans"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

type mockLikeChecker struct {
	likedIDs map[string]bool
	isLiked  bool
}

func (m *mockLikeChecker) IsLiked(_ context.Context, _, _ string, _ models.LikeTargetType) (bool, error) {
	return m.isLiked, nil
}

func (m *mockLikeChecker) GetLikedIDs(_ context.Context, _ string, _ models.LikeTargetType, targetIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(targetIDs))
	for _, id := range targetIDs {
		result[id] = m.likedIDs[id]
	}
	return result, nil
}

func (m *mockLikeChecker) CountByTarget(_ context.Context, _ string, _ models.LikeTargetType) (int, error) {
	return 0, nil
}

func (m *mockLikeChecker) CountByTargets(_ context.Context, targetIDs []string, _ models.LikeTargetType) (map[string]int, error) {
	return make(map[string]int, len(targetIDs)), nil
}

type mockLikeToggler struct {
	liked bool
	count int
}

func (m *mockLikeToggler) Toggle(_ context.Context, _, _ string, _ models.LikeTargetType) (likes.ToggleResult, error) {
	m.liked = !m.liked
	if m.liked {
		m.count++
	} else {
		m.count--
	}
	return likes.ToggleResult{Liked: m.liked, LikesCount: m.count}, nil
}

func setupBeanService(t *testing.T) (*beans.Service, *beans.Repository) {
	t.Helper()
	database := testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, database, "user-1")
	testutil.CreateTestUser(t, database, "user-2")
	repo := beans.NewRepository(database, database)
	svc := beans.NewService(nil, repo, nil, &mockLikeChecker{likedIDs: make(map[string]bool)}, &mockLikeToggler{})
	return svc, repo
}

func TestBeanService_ToggleLike(t *testing.T) {
	t.Run("returns not found for non-existent bean", func(t *testing.T) {
		svc, _ := setupBeanService(t)

		_, err := svc.ToggleLike(t.Context(), "user-1", "non-existent")
		assert.ErrorIs(t, err, beans.ErrNotFound)
	})

	t.Run("toggles like on existing bean", func(t *testing.T) {
		svc, repo := setupBeanService(t)
		b := testutil.CreateTestBean(t, repo, "bean-1", "user-1")

		result, err := svc.ToggleLike(t.Context(), "user-1", b.Id)
		require.NoError(t, err)
		assert.True(t, result.Liked)
		assert.Equal(t, 1, result.LikesCount)
	})

	t.Run("unlike removes like", func(t *testing.T) {
		svc, repo := setupBeanService(t)
		b := testutil.CreateTestBean(t, repo, "bean-1", "user-1")

		_, err := svc.ToggleLike(t.Context(), "user-1", b.Id)
		require.NoError(t, err)

		result, err := svc.ToggleLike(t.Context(), "user-1", b.Id)
		require.NoError(t, err)
		assert.False(t, result.Liked)
		assert.Equal(t, 0, result.LikesCount)
	})
}
