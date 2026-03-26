package handlers

import (
	"context"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

var _ StrictServerInterface = (*ServerHandler)(nil)

// Config holds handler-level configuration.
type Config struct {
	SecureCookies bool
}

// LikedRecipesFetcher returns a paginated list of recipes liked by a user.
type LikedRecipesFetcher interface {
	ListLikedRecipes(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedRecipe], error)
}

type ServerHandler struct {
	logger              *slog.Logger
	cfg                 Config
	authService         *auth.Service
	uploadService       *uploads.Service
	userService         *users.Service
	recipeService       *recipes.Service
	likedRecipesFetcher LikedRecipesFetcher
}

func NewServerHandler(
	recipeService *recipes.Service,
	authService *auth.Service,
	userService *users.Service,
	uploader *uploads.Service,
	likedRecipesFetcher LikedRecipesFetcher,
	cfg Config,
) *ServerHandler {
	return &ServerHandler{
		logger:              slog.Default(),
		cfg:                 cfg,
		recipeService:       recipeService,
		authService:         authService,
		userService:         userService,
		uploadService:       uploader,
		likedRecipesFetcher: likedRecipesFetcher,
	}
}
