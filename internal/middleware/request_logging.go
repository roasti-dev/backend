package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter

	status int
	size   int
	body   bytes.Buffer
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status >= 400 {
		w.body.Write(b)
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func RequestLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			query := r.URL.Query()
			queryAttrs := make([]any, 0, len(query))

			for key, values := range query {
				queryAttrs = append(queryAttrs, slog.Any(key, values))
			}

			logger.InfoContext(r.Context(), "request started",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("user_agent", r.UserAgent()),
				slog.String("remote_ip", r.RemoteAddr),
				slog.Group("query", queryAttrs...),
			)

			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := float64(time.Since(start).Microseconds()) / 1000.0

			level := slog.LevelInfo
			if rw.status >= 500 {
				level = slog.LevelError
			} else if rw.status >= 400 {
				level = slog.LevelWarn
			}

			if rw.status >= 400 {
				logger.ErrorContext(r.Context(), "request failed",
					slog.Int("status", rw.status),
					slog.String("error_body", rw.body.String()),
				)
			}

			logger.LogAttrs(r.Context(), level, "request finished",
				slog.Int("status", rw.status),
				slog.Float64("duration", duration),
				slog.Int("size", rw.size),
			)
		})
	}
}
