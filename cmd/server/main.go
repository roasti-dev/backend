package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	"github.com/nikpivkin/roasti-app-backend/internal/server"

	_ "embed"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Error", log.Err(err))
		os.Exit(1)
	}
}

const (
	appVersion = "0.0.1"

	serverAddr      = ":9090"
	shutdownTimeout = 5 * time.Second
)

func run() error {
	logger := log.InitLogger(appVersion)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.NewSQLite("data.db")
	if err != nil {
		return fmt.Errorf("create db: %w", err)
	}

	if err := db.InitSchema(database); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	recipeRepo := recipe.NewRepository(database)
	recipeService := recipe.NewService(recipeRepo)

	if err := seed.Run(ctx, seed.Services{
		RecipeService: recipeService,
	}); err != nil {
		return err
	}

	swagger, err := handlers.GetSwagger()
	if err != nil {
		return err
	}

	strictHandler := handlers.NewServerHandler(recipeService)
	handler := handlers.NewStrictHandlerWithOptions(strictHandler, nil, handlers.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.ErrorContext(r.Context(), "API handler error",
				slog.Any("error", err),
			)

			if apiErr, ok := err.(*apierr.ApiError); ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(apiErr.Status)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": apiErr.Message})
				return
			}

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		},
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
		middleware.RequestLogging(slog.Default()),
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

	s := server.New(serverAddr, finalHandler)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("Server started", slog.String("addr", serverAddr))
		if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("Signal received", slog.Any("cause", context.Cause(ctx)))
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	slog.Info("Shutdown server")
	if err := s.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	slog.Info("Server stopped")
	return nil
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
