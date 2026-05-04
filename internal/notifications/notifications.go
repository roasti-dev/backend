package notifications

// Notification is the internal domain type used when creating notifications from events.
type Notification struct {
	ID       string
	UserID   string
	Type     string
	ActorID  string
	EntityID string
}

const (
	TypeLikeRecipe     = "like_recipe"
	TypeLikeArticle    = "like_article"
	TypeLikeBean       = "like_bean"
	TypeCommentRecipe  = "comment_recipe"
	TypeCommentArticle = "comment_article"
	TypeCommentBean    = "comment_bean"
	TypeFollow         = "follow"
)
