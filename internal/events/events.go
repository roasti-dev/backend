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

// PostLikeToggled is published when a user likes or unlikes a post.
type PostLikeToggled struct {
	PostID   string
	OwnerID  string
	ByUserID string
	Liked    bool
}

// PostCommentCreated is published when a user comments on a post.
type PostCommentCreated struct {
	PostID    string
	OwnerID   string
	ByUserID  string
	CommentID string
}

// BeanLikeToggled is published when a user likes or unlikes a bean.
type BeanLikeToggled struct {
	BeanID   string
	OwnerID  string
	ByUserID string
	Liked    bool
}

// BeanCommentCreated is published when a user comments on a bean.
type BeanCommentCreated struct {
	BeanID    string
	OwnerID   string
	ByUserID  string
	CommentID string
}

// RecipeCommentCreated is published when a user comments on a recipe.
type RecipeCommentCreated struct {
	RecipeID  string
	OwnerID   string
	ByUserID  string
	CommentID string
}
