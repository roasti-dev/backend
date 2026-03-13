package handlers

import (
	"context"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
)

var _ StrictServerInterface = (*ServerHandler)(nil)

type ServerHandler struct {
	logger        *slog.Logger
	recipeService *recipe.Service
}

func NewServerHandler(recipeService *recipe.Service) *ServerHandler {
	return &ServerHandler{
		logger:        slog.Default(),
		recipeService: recipeService,
	}
}

func (s *ServerHandler) GetHealth(ctx context.Context, request GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200TextResponse("OK"), nil
}
