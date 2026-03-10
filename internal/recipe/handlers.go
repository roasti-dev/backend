package recipe

import (
	"encoding/json"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GET /api/v1/recipes
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	params, err := ParseListRecipesParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	recipes, err := h.service.ListRecipes(r.Context(), userID, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}

// POST /api/v1/recipes
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	authorID := auth.UserIDFromContext(r.Context())

	var recipe Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := h.service.CreateRecipe(r.Context(), authorID, recipe)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// DELETE /api/v1/recipes/{recipe_id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	authorID := auth.UserIDFromContext(r.Context())
	recipeID := r.PathValue("recipe_id")
	if err := h.service.DeleteRecioe(r.Context(), authorID, recipeID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
