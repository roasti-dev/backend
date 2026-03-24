package models

import (
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/ptr"
)

// CloneFor creates a copy of the recipe attributed to newAuthorID.
// The clone gets a new ID, zeroed likes, cleared note, and an origin reference.
func (r Recipe) CloneFor(newAuthorID, newID string, now time.Time) Recipe {
	clone := r
	clone.Id = newID
	clone.AuthorId = newAuthorID
	clone.Title = "Copy of " + r.Title
	clone.Note = nil
	clone.LikesCount = 0
	clone.CreatedAt = now
	clone.UpdatedAt = now
	// Only RecipeId is stored; the remaining origin fields (author, username, avatar)
	// are populated by a JOIN when the recipe is fetched from the repository.
	clone.Origin = &RecipeOrigin{RecipeId: r.Id}
	return clone
}

// RedactForUser removes fields that are private to the recipe author.
// Must be called before returning a recipe to any user other than the author.
func (r *Recipe) RedactForUser(userID string) {
	if r.AuthorId != userID {
		r.Note = nil
	}
}

func (p ListRecipesParams) Pagination() PaginationParams {
	return NewPaginationParams(ptr.FromPtr(p.Page), ptr.FromPtr(p.Limit))
}

func (p ListUserLikesParams) Pagination() PaginationParams {
	return NewPaginationParams(ptr.FromPtr(p.Page), ptr.FromPtr(p.Limit))
}
