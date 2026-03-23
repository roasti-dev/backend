package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/nikpivkin/roasti-app-backend/internal/app"
	"github.com/nikpivkin/roasti-app-backend/internal/log"
	"github.com/nikpivkin/roasti-app-backend/internal/server"

	_ "embed"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Error", log.Err(err))
		os.Exit(1)
	}
}

var appVersion = "dev"

const (
	shutdownTimeout = 5 * time.Second
)

func run() error {

	if err := godotenv.Load(".env"); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}

	serverPort := getEnvOrDefault("SERVER_PORT", "9090")
	if _, err := strconv.Atoi(serverPort); err != nil {
		return fmt.Errorf("invalid port %q: must be a valid integer (e.g. 8080)", serverPort)
	}

	appEnv := log.Env(getEnvOrDefault("APP_ENV", string(log.EnvDevelopment)))
	if !appEnv.IsValid() {
		slog.Warn("unknown APP_ENV, falling back to development", "value", appEnv)
		appEnv = log.EnvDevelopment
	}
	debug := os.Getenv("DEBUG") != ""
	logger := log.InitLogger(appVersion, appEnv, debug)

	slog.SetDefault(logger)

	emulatorHost := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if emulatorHost != "" {
		logger.Info("Firebase auth emulator is used", slog.String("host", emulatorHost))
	} else {
		logger.Info("Firebase auth production is used")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a, err := app.New(ctx, app.Config{
		Debug:                         debug,
		DBPath:                        getEnvOrDefault("DATABASE_PATH", "data.db"),
		UploadsPath:                   getEnvOrDefault("UPLOADS_PATH", "./uploads"),
		AppVersion:                    appVersion,
		FirebaseProjectID:             os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseAPIKey:                os.Getenv("FIREBASE_API_KEY"),
		FirebaseCredentialsJSONBase64: os.Getenv("FIREBASE_CREDENTIALS_JSON_BASE64"),
		FirebaseIdentityBaseURL:       getEnvOrDefault("FIREBASE_IDENTITY_BASE_URL", "https://identitytoolkit.googleapis.com/v1/accounts"),
		FirebaseTokenBaseURL:          getEnvOrDefault("FIREBASE_TOKEN_BASE_URL", "https://securetoken.googleapis.com/v1/token"),
	}, logger)
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	serverAddr := ":" + serverPort
	s := server.New(serverAddr, a.Handler())

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

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
