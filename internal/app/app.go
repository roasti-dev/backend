package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/nikpivkin/roasti-app-backend/docs"
	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
	"github.com/nikpivkin/roasti-app-backend/internal/db"
	"github.com/nikpivkin/roasti-app-backend/internal/handlers"
	"github.com/nikpivkin/roasti-app-backend/internal/log"
	"github.com/nikpivkin/roasti-app-backend/internal/middleware"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
	"github.com/nikpivkin/roasti-app-backend/internal/seed"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
)

type Config struct {
	DBPath      string
	UploadsPath string
	AppVersion  string
}

type App struct {
	handler       http.Handler
	db            *sql.DB
	recipeService *recipe.Service
}

func New(cfg Config, logger *slog.Logger) (*App, error) {
	database, err := db.NewSQLite(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("create db: %w", err)
	}

	if err := db.InitSchema(database); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	uploader := uploads.NewService(cfg.UploadsPath)
	recipeRepo := recipe.NewRepository(database, logger)
	recipeService := recipe.NewService(recipeRepo, uploader)

	swagger, err := handlers.GetSwagger()
	if err != nil {
		return nil, err
	}

	strictHandler := handlers.NewServerHandler(recipeService, uploader)
	handler := handlers.NewStrictHandlerWithOptions(strictHandler, nil, handlers.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: responseErrorHandler,
	})

	router := http.NewServeMux()
	router.HandleFunc("/openapi.json", serveOpenAPIJSON(swagger))
	router.Handle("/docs/", serveSwaggerStatic(docs.SwaggerHTML))
	router.Handle("/docs", http.RedirectHandler("/docs/", http.StatusMovedPermanently))

	handlers.HandlerWithOptions(handler, handlers.StdHTTPServerOptions{
		BaseRouter: router,
	})

	apiHandler := middleware.Chain(
		router,
		oapimiddleware.OapiRequestValidator(swagger),
		middleware.RequestLogging(logger),
		middleware.RequestID,
		middleware.UserID,
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			apiHandler.ServeHTTP(w, r)
		} else {
			router.ServeHTTP(w, r)
		}
	})

	return &App{
		handler:       finalHandler,
		db:            database,
		recipeService: recipeService,
	}, nil
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) Seed(ctx context.Context) error {
	return seed.Run(ctx, seed.Services{
		RecipeService: a.recipeService,
	})
}

func responseErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "API handler error", log.Err(err))

	if apiErr, ok := errors.AsType[*apierr.ApiError](err); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(apiErr.Status)

		if err := json.NewEncoder(w).Encode(map[string]string{"error": apiErr.Message}); err != nil {
			slog.WarnContext(r.Context(), "Encode Api error", log.Err(err))
		}
		return
	}

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func serveSwaggerStatic(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}

func serveOpenAPIJSON(doc *openapi3.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(doc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
