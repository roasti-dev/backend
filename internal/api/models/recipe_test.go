package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

func TestRecipe_CloneFor(t *testing.T) {
	note := "author's private note"
	original := models.Recipe{
		Id:         "original-id",
		AuthorId:   "user-1",
		Title:      "My Recipe",
		Note:       &note,
		LikesCount: 42,
	}

	now := time.Now().UTC()
	clone := original.CloneFor("user-2", "clone-id", now)

	require.NotNil(t, clone)
	assert.Nil(t, clone.Note, "note must not be copied from original")
	assert.Zero(t, clone.LikesCount, "likes count must be reset to zero")
}
