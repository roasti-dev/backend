package handlers

import (
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
)

var _ StrictServerInterface = (*ServerHandler)(nil)

type ServerHandler struct {
	logger         *slog.Logger
	recipeService  *recipe.Service
	authService    *auth.Service
	uploadsService *uploads.Service
}

func NewServerHandler(recipeService *recipe.Service, authService *auth.Service, uploader *uploads.Service) *ServerHandler {
	return &ServerHandler{
		logger:         slog.Default(),
		recipeService:  recipeService,
		authService:    authService,
		uploadsService: uploader,
	}
}
