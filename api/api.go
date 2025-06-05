package api

import (
	"context"
	"encoding/json"
	"github.com/brinestone/dfs/fs"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Config struct {
	Addr       string
	Ctx        context.Context
	Logger     *slog.Logger
	FileServer *fs.FileServer
}

type Server struct {
	Config
	mux      *http.ServeMux
	server   *http.Server
	listener net.Listener
}

func (s *Server) Start() (err error) {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return
	}

	s.listener = l
	s.server = &http.Server{
		Handler:      s.mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	s.server.RegisterOnShutdown(func() {
		s.Logger.Info("Server is shutting down", "addr", s.Addr)
	})
	s.Logger.Info("Server started successfully", "addr", s.Addr)
	err = s.server.Serve(l)
	return
}

func (s *Server) Stop() error {
	return s.server.Shutdown(s.Ctx)
}

func NewServer(c Config) (server *Server) {
	server = &Server{
		Config: c,
		mux:    http.NewServeMux(),
	}
	if server.Logger == nil {
		server.Logger = slog.Default()
	}
	server.setupHandlers()
	return
}

func (s *Server) setupHandlers() {
	// Logger middleware
	logMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Logger.Info("Request received", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
			next.ServeHTTP(w, r)
		})
	}

	s.mux.Handle(
		"/",
		logMiddleware(http.HandlerFunc(homeHandler)),
	)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().
		Add("Content-Type", "application/json")
	s, _ := json.Marshal(map[string]string{
		"status": "OK",
	})

	w.Write(s)
}
