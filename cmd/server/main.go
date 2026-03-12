package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	middleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/nikpivkin/roasti-app-backend/docs"
	"github.com/nikpivkin/roasti-app-backend/internal/api"
	"github.com/nikpivkin/roasti-app-backend/internal/db"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
	"github.com/nikpivkin/roasti-app-backend/internal/seed"
	"github.com/nikpivkin/roasti-app-backend/internal/server"

	_ "embed"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err.Error())
	}
}

const (
	serverAddr      = ":9090"
	shutdownTimeout = 5 * time.Second
)

func run() error {
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

	swagger, err := api.GetSwagger()
	if err != nil {
		return err
	}

	strictHandler := api.NewServerHandler(recipeService)
	handler := api.NewStrictHandler(strictHandler, nil)

	router := http.NewServeMux()
	router.HandleFunc("/openapi.json", serveOpenAPIJSON(swagger))
	router.Handle("/docs/", serveSwaggerStatic(docs.SwaggerHTML))
	router.Handle("/docs", http.RedirectHandler("/docs/", http.StatusMovedPermanently))

	api.HandlerFromMux(handler, router)

	s := server.New(serverAddr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.UserMiddleware(middleware.OapiRequestValidator(swagger)(router)).ServeHTTP(w, r)
		} else {
			router.ServeHTTP(w, r)
		}
	}))

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Server started at %s", serverAddr)
		if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println(context.Cause(ctx).Error())
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	log.Printf("Shutdown server")
	if err := s.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	log.Printf("Server stopped")
	return nil
}

func serveSwaggerStatic(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
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
