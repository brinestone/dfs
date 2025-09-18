package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/brinestone/dfs/events"
	"github.com/brinestone/dfs/fs"
	"github.com/brinestone/dfs/storage"
)

type Config struct {
	Addr     string
	Logger   *slog.Logger
	BasePath string
	Verbose  bool
}

type Api struct {
	Config
	mux      *http.ServeMux
	server   *http.Server
	listener net.Listener
	eb       *events.EventBus
	store    *storage.Store
	fs       *fs.FileServer
	ctx      context.Context
}

func (s *Api) Start() (err error) {
	s.listener, err = net.Listen("tcp", s.Addr)
	go func() {
		<-s.ctx.Done()
		s.stop()
	}()
	if err != nil {
		return
	}

	s.server = &http.Server{Handler: s.mux}
	s.Logger.Info("Api started successfully", "addr", s.Addr)
	err = s.server.Serve(s.listener)
	return
}

func (s *Api) stop() error {
	return errors.Join(s.server.Shutdown(s.ctx), s.listener.Close())
}

func NewApi(ctx context.Context, f *fs.FileServer, s *storage.Store, eb *events.EventBus, c Config) (server *Api) {
	server = &Api{
		Config: c,
		mux:    http.NewServeMux(),
		fs:     f,
		ctx:    ctx,
		store:  s,
		eb:     eb,
	}
	if server.Logger == nil {
		server.Logger = slog.Default()
	}
	server.setupHandlers()
	return
}

func (s *Api) makeApiPath(path string) string {
	pathname, err := url.JoinPath(s.BasePath, path)
	if err != nil {
		panic(err)
	}
	return pathname
}

func (s *Api) makeApiPathWithBase(path string, base string) string {
	pathname, err := url.JoinPath(s.BasePath, base, path)
	if err != nil {
		panic(err)
	}
	return pathname
}

func (s *Api) registerHandler(path string, handler http.Handler) {
	// pathname := s.makeApiPath(path)
	s.mux.Handle(path, handler)
	if s.Verbose {
		s.Logger.Debug("registered handler", "path", path)
	}
}

func (s *Api) registerLoggedHandler(path string, handler http.Handler) {
	// Logger middleware
	logMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			timeTaken := time.Since(start)
			s.Logger.Info("request received", "duration", timeTaken, "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		})
	}
	// s.mux.Handle(path, logMiddleware(handler))
	s.registerHandler(path, logMiddleware(handler))
}

func (s *Api) setupHandlers() {
	s.registerLoggedHandler(
		s.makeApiPath("/health"),
		http.HandlerFunc(homeHandler),
	)
	s.setupStorageEndpointHandlers()
}

func homeHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().
		Add("Content-Type", "application/json")
	s, _ := json.Marshal(map[string]string{
		"status": "OK",
	})

	w.Write(s)
}
