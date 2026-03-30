package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
	"google.golang.org/api/option"

	"github.com/nikpivkin/roasti-app-backend/assets"
	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
	"github.com/nikpivkin/roasti-app-backend/internal/app/middleware"
	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/db"
	"github.com/nikpivkin/roasti-app-backend/internal/events"
	"github.com/nikpivkin/roasti-app-backend/internal/handlers"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/log"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

type App struct {
	handler http.Handler
}

func New(ctx context.Context, cfg Config, logger *slog.Logger) (*App, error) {
	database, err := db.NewSQLite(ctx, cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("create db: %w", err)
	}

	if err := db.Migrate(database); err != nil {
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

	passwordPolicy, err := fetchPasswordPolicy(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("fetch password policy: %w", err)
	}

	runner := db.NewRunner(database, slog.Default(), cfg.Debug)

	uploadRepo := uploads.NewRepository(database)
	uploader := uploads.NewService(cfg.UploadsPath, uploadRepo)
	startTmpCleanup(ctx, uploader)

	bus := events.NewBus(64)
	bus.Start(ctx)

	recipeRepo := recipes.NewRepository(database, runner)
	likeRepo := likes.NewRepository(database)
	likeService := likes.NewService(likeRepo)
	recipeService := recipes.NewService(recipeRepo, uploader, likeService, likeService).WithPublisher(bus)
	userRepo := users.NewUserRepository(database)
	userService := users.NewUserService(userRepo, &firebaseIdentityCreator{firebaseAuth}, uploader)

	revokedTokenRepo := auth.NewRevokedTokenRepository(database)
	startRevokedTokenCleanup(ctx, revokedTokenRepo)

	passwordSigner := auth.NewFirebasePasswordSigner(
		cfg.FirebaseAPIKey, cfg.FirebaseIdentityBaseURL, cfg.FirebaseTokenBaseURL,
	)
	authService := auth.NewService(userService, revokedTokenRepo, firebaseAuth, passwordSigner, passwordPolicy)

	swagger, err := handlers.GetSwagger()
	if err != nil {
		return nil, err
	}

	strictHandler := handlers.NewServerHandler(
		recipeService, authService,
		userService, uploader,
		&userLibrary{users: userRepo, likes: likeService, recipes: recipeService},
		handlers.Config{
			SecureCookies: cfg.SecureCookies,
		},
	)
	handler := handlers.NewStrictHandlerWithOptions(strictHandler, nil, handlers.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: responseErrorHandler,
	})

	router := http.NewServeMux()
	router.HandleFunc("/openapi.json", serveOpenAPIJSON(swagger))
	router.Handle("/docs/", serveSwaggerStatic(assets.SwaggerHTML))
	router.Handle("/docs", http.RedirectHandler("/docs/", http.StatusMovedPermanently))

	handlers.HandlerWithOptions(handler, handlers.StdHTTPServerOptions{
		BaseRouter: router,
		Middlewares: []handlers.MiddlewareFunc{
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
			middleware.RateLimit(cfg.RateLimit),
			middleware.RequestLogging(logger),
			middleware.RequestID,
		},
	})

	return &App{
		handler: corsMiddleware(cfg.AllowedOrigins)(router),
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

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if len(allowedOrigins) == 0 {
				// dev mode
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if slices.Contains(allowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func startRevokedTokenCleanup(ctx context.Context, repo *auth.RevokedTokenRepository) {
	runPeriodic(ctx, "revoked-token-cleanup", 24*time.Hour, func(ctx context.Context) error {
		return repo.DeleteExpired(ctx)
	})
}

func fetchPasswordPolicy(ctx context.Context, cfg Config, logger *slog.Logger) (auth.PasswordPolicy, error) {
	var credJSON []byte
	if cfg.FirebaseCredentialsJSONBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(cfg.FirebaseCredentialsJSONBase64)
		if err != nil {
			return auth.PasswordPolicy{}, fmt.Errorf("decode firebase credentials: %w", err)
		}
		credJSON = decoded
	}

	policy, err := auth.GetPasswordPolicy(ctx, cfg.FirebaseProjectID, credJSON)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch firebase password policy, using defaults", log.Err(err))
		return auth.DefaultPasswordPolicy, nil
	}

	logger.InfoContext(ctx, "firebase password policy loaded",
		slog.Int("min_length", policy.MinLength),
		slog.Int("max_length", policy.MaxLength),
		slog.Bool("require_uppercase", policy.RequireUppercase),
		slog.Bool("require_numeric", policy.RequireNumeric),
	)
	return policy, nil
}

func startTmpCleanup(ctx context.Context, svc *uploads.Service) {
	runPeriodic(ctx, "tmp-upload-cleanup", 24*time.Hour, func(ctx context.Context) error {
		return svc.DeleteUnconfirmed(ctx, 24*time.Hour)
	})
}

func runPeriodic(ctx context.Context, name string, interval time.Duration, fn func(context.Context) error) {
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "panic in background task",
					slog.String("name", name),
					slog.Any("panic", r),
				)
			}
		}()
		if err := fn(ctx); err != nil {
			slog.ErrorContext(ctx, "background task failed",
				slog.String("name", name), log.Err(err))
		}
	}

	go func() {
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				run()
			case <-ctx.Done():
				return
			}
		}
	}()
}
