package handlers

import (
	"context"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/posts"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

var _ StrictServerInterface = (*ServerHandler)(nil)

// Config holds handler-level configuration.
type Config struct {
	SecureCookies bool
}

// UserLibrary provides access to a user's saved/liked content.
type UserLibrary interface {
	ListLikedRecipes(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedRecipe], error)
	ListLikedPosts(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedPost], error)
}

// PostService handles post creation and feed listing.
type PostService interface {
	CreatePost(ctx context.Context, userID string, req models.CreatePostRequest) (models.Post, error)
	DeletePost(ctx context.Context, userID, postID string) error
	GetPost(ctx context.Context, userID, postID string) (models.Post, error)
	UpdatePost(ctx context.Context, userID, postID string, req models.UpdatePostRequest) (models.Post, error)
	ToggleLike(ctx context.Context, userID, postID string) (likes.ToggleResult, error)
	ListPosts(ctx context.Context, userID string, params posts.ListPostsParams) (models.GenericPage[models.Post], error)
	CreateComment(ctx context.Context, userID, postID, text string, parentID *string) (models.PostComment, error)
	DeleteComment(ctx context.Context, userID, commentID string) error
}

type ServerHandler struct {
	logger        *slog.Logger
	cfg           Config
	authService   *auth.Service
	uploadService *uploads.Service
	userService   *users.Service
	recipeService *recipes.Service
	postService   PostService
	userLibrary   UserLibrary
}

func NewServerHandler(
	recipeService *recipes.Service,
	authService *auth.Service,
	userService *users.Service,
	uploader *uploads.Service,
	postService PostService,
	userLibrary UserLibrary,
	cfg Config,
) *ServerHandler {
	return &ServerHandler{
		logger:        slog.Default(),
		cfg:           cfg,
		recipeService: recipeService,
		authService:   authService,
		userService:   userService,
		uploadService: uploader,
		postService:   postService,
		userLibrary:   userLibrary,
	}
}
