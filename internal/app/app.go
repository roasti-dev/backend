package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
	"google.golang.org/api/option"

	"github.com/nikpivkin/roasti-app-backend/docs"
	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/db"
	"github.com/nikpivkin/roasti-app-backend/internal/handlers"
	"github.com/nikpivkin/roasti-app-backend/internal/log"
	"github.com/nikpivkin/roasti-app-backend/internal/middleware"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
)

type Config struct {
	DBPath                        string
	UploadsPath                   string
	AppVersion                    string
	FirebaseProjectID             string
	FirebaseCredentialsJSONBase64 string
	FirebaseAPIKey                string
	FirebaseIdentityBaseURL       string
	FirebaseTokenBaseURL          string
}

type App struct {
	handler http.Handler
}

func New(ctx context.Context, cfg Config, logger *slog.Logger) (*App, error) {
	database, err := db.NewSQLite(ctx, cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("create db: %w", err)
	}

	if err := db.InitSchema(database); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	config := &firebase.Config{ProjectID: cfg.FirebaseProjectID}

	opts := []option.ClientOption{}
	if cfg.FirebaseCredentialsJSONBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(cfg.FirebaseCredentialsJSONBase64)
		if err != nil {
			return nil, fmt.Errorf("decode firebase credentials: %w", err)
		}
		opts = append(opts, option.WithCredentialsJSON(decoded))
	}

	firebaseApp, err := firebase.NewApp(ctx, config, opts...)
	if err != nil {
		return nil, fmt.Errorf("create a new firebase app: %w", err)
	}

	firebaseAuth, err := firebaseApp.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("create a new firebase auth client: %w", err)
	}

	logger.InfoContext(ctx, "firebase config",
		slog.String("identity_base_url", cfg.FirebaseIdentityBaseURL),
		slog.String("token_base_url", cfg.FirebaseTokenBaseURL),
	)

	uploadRepo := uploads.NewRepository(database)
	uploader := uploads.NewService(cfg.UploadsPath, uploadRepo)
	startTmpCleanup(ctx, uploader)

	recipeRepo := recipe.NewRepository(database, logger)
	recipeService := recipe.NewService(recipeRepo, uploader)
	userRepo := auth.NewUserRepository(database)

	revokedTokenRepo := auth.NewRevokedTokenRepository(database)
	startRevokedTokenCleanup(ctx, revokedTokenRepo)

	passwordSigner := auth.NewFirebasePasswordSigner(
		cfg.FirebaseAPIKey, cfg.FirebaseIdentityBaseURL, cfg.FirebaseTokenBaseURL,
	)
	authService := auth.NewService(userRepo, revokedTokenRepo, uploader, firebaseAuth, passwordSigner)

	swagger, err := handlers.GetSwagger()
	if err != nil {
		return nil, err
	}

	strictHandler := handlers.NewServerHandler(recipeService, authService, uploader)
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
		oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapimiddleware.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: middleware.Authenticate(firebaseAuth),
			},
		}),
		middleware.ApplyForRoutes(
			middleware.RefreshToken,
			"/api/v1/auth/refresh",
			"/api/v1/auth/logout",
		),
		middleware.RequestLogging(logger),
		middleware.RequestID,
	)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			apiHandler.ServeHTTP(w, r)
		} else {
			router.ServeHTTP(w, r)
		}
	})

	return &App{
		handler: corsMiddleware(finalHandler),
	}, nil
}

func (a *App) Handler() http.Handler {
	return a.handler
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func startRevokedTokenCleanup(ctx context.Context, repo *auth.RevokedTokenRepository) {
	go func() {
		if err := repo.DeleteExpired(ctx); err != nil {
			slog.ErrorContext(ctx, "delete expired tokens", log.Err(err))
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := repo.DeleteExpired(ctx); err != nil {
					slog.ErrorContext(ctx, "delete expired tokens", log.Err(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func startTmpCleanup(ctx context.Context, svc *uploads.Service) {
	go func() {
		if err := svc.DeleteUnconfirmed(ctx, 24*time.Hour); err != nil {
			slog.ErrorContext(ctx, "cleanup tmp uploads", log.Err(err))
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := svc.DeleteUnconfirmed(ctx, 24*time.Hour); err != nil {
					slog.ErrorContext(ctx, "cleanup tmp uploads", log.Err(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
