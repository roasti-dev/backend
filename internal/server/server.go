package server

import (
	"context"
	"net/http"
)

type Server struct {
	srv *http.Server
}

func New(addr string, handler http.Handler) *Server {
	s := Server{
		srv: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
	return &s
}

func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
