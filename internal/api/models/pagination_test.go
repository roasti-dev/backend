package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

func TestNewPage(t *testing.T) {
	type testCase[T any] struct {
		name     string
		items    []T
		pag      models.PaginationParams
		total    int
		expected models.PaginationMeta
	}

	tests := []testCase[int]{
		{
			name:  "middle page",
			items: []int{1, 2, 3},
			pag:   models.NewPaginationParams(2, 10),
			total: 50,
			expected: models.PaginationMeta{
				ItemsCount:  3,
				CurrentPage: 2,
				NextPage:    3,
				LastPage:    5,
			},
		},
		{
			name:  "last page",
			items: []int{1, 2},
			pag:   models.NewPaginationParams(5, 10),
			total: 42,
			expected: models.PaginationMeta{
				ItemsCount:  2,
				CurrentPage: 5,
				NextPage:    5,
				LastPage:    5,
			},
		},
		{
			name:  "empty result",
			items: nil,
			pag:   models.NewPaginationParams(1, 20),
			total: 0,
			expected: models.PaginationMeta{
				ItemsCount:  0,
				CurrentPage: 1,
				NextPage:    1,
				LastPage:    1,
			},
		},
		{
			name:  "single page",
			items: []int{1, 2, 3},
			pag:   models.NewPaginationParams(1, 20),
			total: 3,
			expected: models.PaginationMeta{
				ItemsCount:  3,
				CurrentPage: 1,
				NextPage:    1,
				LastPage:    1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := models.NewPage(tc.items, tc.pag, tc.total)
			assert.Equal(t, tc.expected, result.Pagination)
			assert.NotNil(t, result.Items)
		})
	}
}

func TestPaginationParams(t *testing.T) {
	t.Run("defaults when nil", func(t *testing.T) {
		pag := models.PaginationParams{}
		assert.Equal(t, int32(models.DefaultPage), pag.GetPage())
		assert.Equal(t, int32(models.DefaultLimit), pag.GetLimit())
		assert.Equal(t, int32(0), pag.Offset())
	})

	t.Run("custom values", func(t *testing.T) {
		pag := models.NewPaginationParams(3, 10)
		assert.Equal(t, int32(3), pag.GetPage())
		assert.Equal(t, int32(10), pag.GetLimit())
		assert.Equal(t, int32(20), pag.Offset())
	})

	t.Run("page below 1 defaults to 1", func(t *testing.T) {
		pag := models.NewPaginationParams(0, 10)
		assert.Equal(t, int32(models.DefaultPage), pag.GetPage())
	})

	t.Run("limit below 1 defaults to DefaultLimit", func(t *testing.T) {
		pag := models.NewPaginationParams(1, 0)
		assert.Equal(t, int32(models.DefaultLimit), pag.GetLimit())
	})

	t.Run("limit above MaxLimit defaults to DefaultLimit", func(t *testing.T) {
		pag := models.NewPaginationParams(1, models.MaxLimit+1)
		assert.Equal(t, int32(models.DefaultLimit), pag.GetLimit())
	})

	t.Run("offset calculation", func(t *testing.T) {
		pag := models.NewPaginationParams(4, 25)
		// (4-1) * 25 = 75
		assert.Equal(t, int32(75), pag.Offset())
	})
}
