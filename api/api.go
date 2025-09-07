package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/brinestone/dfs/fs"
)

type Config struct {
	Addr       string
	Ctx        context.Context
	Logger     *slog.Logger
	FileServer *fs.FileServer
}

type Api struct {
	Config
	mux      *http.ServeMux
	server   *http.Server
	listener net.Listener
}

func (s *Api) Start() (err error) {
	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return
	}

	s.server = &http.Server{Handler: s.mux}
	s.Logger.Info("Api started successfully", "addr", s.Addr)
	err = s.server.Serve(s.listener)
	return
}

func (s *Api) Stop() error {
	s.server.Shutdown(s.Ctx)
	return nil
}

func NewApi(c Config) (server *Api) {
	server = &Api{
		Config: c,
		mux:    http.NewServeMux(),
	}
	if server.Logger == nil {
		server.Logger = slog.Default()
	}
	server.setupHandlers()
	return
}

func (s *Api) registerLoggedHandler(path string, handler http.HandlerFunc) {
	// Logger middleware
	logMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			timeTaken := time.Since(start)
			s.Logger.Info("Request received", "duration", timeTaken, "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		})
	}
	s.mux.Handle(path, logMiddleware(handler))
}

func (s *Api) setupHandlers() {
	s.registerLoggedHandler(
		"/",
		http.HandlerFunc(homeHandler),
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
