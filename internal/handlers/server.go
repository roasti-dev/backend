package handlers

import (
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

var _ StrictServerInterface = (*ServerHandler)(nil)

type ServerHandler struct {
	logger        *slog.Logger
	secureCookies bool
	authService   *auth.Service
	uploadService *uploads.Service
	userService   *users.Service
	recipeService *recipes.Service
	likeService   *likes.Service
}

func NewServerHandler(
	recipeService *recipes.Service,
	authService *auth.Service,
	userService *users.Service,
	uploader *uploads.Service,
	likeService *likes.Service,
	secureCookies bool,
) *ServerHandler {
	return &ServerHandler{
		logger:        slog.Default(),
		secureCookies: secureCookies,
		recipeService: recipeService,
		authService:   authService,
		userService:   userService,
		uploadService: uploader,
		likeService:   likeService,
	}
}
