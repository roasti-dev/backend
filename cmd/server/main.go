package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	a, err := app.New(app.Config{
		DBPath:      "data.db",
		UploadsPath: "./uploads",
		AppVersion:  appVersion,
	}, logger)
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	s := server.New(serverAddr, a.Handler())

	if err := a.Seed(ctx); err != nil {
		return err
	}

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
