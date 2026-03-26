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

	cfg := app.ConfigFromEnv(appVersion)
	if _, err := strconv.Atoi(cfg.ServerPort); err != nil {
		return fmt.Errorf("invalid SERVER_PORT %q: must be a valid integer (e.g. 8080)", cfg.ServerPort)
	}

	logger := log.InitLogger(appVersion, cfg.Env, cfg.Debug)

	slog.SetDefault(logger)

	emulatorHost := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if emulatorHost != "" {
		logger.Info("Firebase auth emulator is used", slog.String("host", emulatorHost))
	} else {
		logger.Info("Firebase auth production is used")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a, err := app.New(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	serverAddr := ":" + cfg.ServerPort
	srv := &http.Server{Addr: serverAddr, Handler: a.Handler()}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("Server started", slog.String("addr", serverAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	slog.Info("Server stopped")
	return nil
}
