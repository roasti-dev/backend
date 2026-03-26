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

// UserLibrary provides access to a user's saved/liked content.
type UserLibrary interface {
	ListLikedRecipes(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedRecipe], error)
}

type ServerHandler struct {
	logger        *slog.Logger
	cfg           Config
	authService   *auth.Service
	uploadService *uploads.Service
	userService   *users.Service
	recipeService *recipes.Service
	userLibrary   UserLibrary
}

func NewServerHandler(
	recipeService *recipes.Service,
	authService *auth.Service,
	userService *users.Service,
	uploader *uploads.Service,
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
		userLibrary:   userLibrary,
	}
}
