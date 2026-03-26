package handlers

import (
	"log/slog"

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

type ServerHandler struct {
	logger        *slog.Logger
	cfg           Config
	authService   *auth.Service
	uploadService *uploads.Service
	userService   *users.Service
	recipeService *recipes.Service
}

func NewServerHandler(
	recipeService *recipes.Service,
	authService *auth.Service,
	userService *users.Service,
	uploader *uploads.Service,
	cfg Config,
) *ServerHandler {
	return &ServerHandler{
		logger:        slog.Default(),
		cfg:           cfg,
		recipeService: recipeService,
		authService:   authService,
		userService:   userService,
		uploadService: uploader,
	}
}
