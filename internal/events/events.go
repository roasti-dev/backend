package events

// Event is the base type for all domain events.
type Event interface{}

// RecipeLikeToggled is published when a user likes or unlikes a recipe.
type RecipeLikeToggled struct {
	RecipeID string
	OwnerID  string
	ByUserID string
	Liked    bool
}
