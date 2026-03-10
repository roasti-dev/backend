package api

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
)

func NewRouter(recipeService *recipe.Service) http.Handler {
	mux := http.NewServeMux()

	recipeHandler := recipe.NewHandler(recipeService)
	mux.Handle("GET /api/v1/recipes", UserMiddleware(http.HandlerFunc(recipeHandler.List)))
	mux.Handle("POST /api/v1/recipes", UserMiddleware(http.HandlerFunc(recipeHandler.Create)))
	mux.Handle("DELETE /api/v1/recipes/{recipe_id}", UserMiddleware(http.HandlerFunc(recipeHandler.Delete)))
	mux.HandleFunc("/health", Health)
	return mux
}
