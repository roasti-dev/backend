package middleware

import "net/http"

func Chain(h http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	wrapped := h
	for _, m := range middleware {
		wrapped = m(wrapped)
	}
	return wrapped
}
