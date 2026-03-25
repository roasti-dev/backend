package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

const errKey = "err"

func Err(err error) slog.Attr {
	return slog.Any(errKey, err)
}

type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if requestID := requestctx.GetRequestID(ctx); requestID != "" {
		r.AddAttrs(slog.String("requestID", requestID))
	}
	if userID := requestctx.GetUserID(ctx); userID != "" {
		r.AddAttrs(slog.String("userID", userID))
	}
	return h.Handler.Handle(ctx, r)
}

func InitLogger(appVersion string, appEnv Env, debug bool) *slog.Logger {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     logLevel,
	}

	var handler slog.Handler
	if appEnv == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(&contextHandler{
		handler.WithAttrs([]slog.Attr{
			slog.String("appVer", appVersion),
			slog.String("env", string(appEnv)),
		}),
	})
}
