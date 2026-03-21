package handlers

import (
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
)

var _ StrictServerInterface = (*ServerHandler)(nil)

type ServerHandler struct {
	logger        *slog.Logger
	authService   *auth.Service
	uploadService *uploads.Service
	recipeService *recipe.Service
	likeService   *likes.Service
}

func NewServerHandler(
	recipeService *recipe.Service,
	authService *auth.Service,
	uploader *uploads.Service,
	likeService *likes.Service,
) *ServerHandler {
	return &ServerHandler{
		logger:        slog.Default(),
		recipeService: recipeService,
		authService:   authService,
		uploadService: uploader,
		likeService:   likeService,
	}
}
