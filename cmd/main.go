package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/akamensky/argparse"

	"github.com/google/uuid"

	"github.com/brinestone/dfs/api"
	"github.com/brinestone/dfs/events"
	"github.com/brinestone/dfs/fs"
	"github.com/brinestone/dfs/p2p"
	"github.com/brinestone/dfs/storage"
)

var (
	storageRoot *string
	verbose     *bool
	bufferSize  *int
	homeDir, _  = os.UserHomeDir()
	logger      *slog.Logger
)

func makeEventBus(ctx context.Context) *events.EventBus {
	return events.NewBus(ctx)
}

func makeStore(root string) (store *storage.Store) {
	storageConfig := storage.StoreConfig{
		TransformKey: storage.CASKeyTransformer(root),
		Root:         root,
		Logger:       logger.WithGroup("store"),
	}

	store = storage.NewStore(storageConfig)
	return
}

func makeServer(ctx context.Context, listenAddr string, id string, store *storage.Store, eb *events.EventBus) *fs.FileServer {
	fsLogger := logger.WithGroup("fs")
	tcpTransportConfig := p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Handshake:  p2p.NoopHandshake,
		Decoder: p2p.DefaultDecoder{
			EncodingConfig: p2p.EncodingConfig{
				BufferSize: int64(*bufferSize),
			},
		},
		Logger: fsLogger.WithGroup("tcp"),
	}
	tcpTransport := p2p.NewTcpTransport(ctx, tcpTransportConfig)

	serverConfig := fs.FileServerConfig{
		ListenAddr:      listenAddr,
		Transport:       tcpTransport,
		Id:              id,
		Logger:          fsLogger.WithGroup("server"),
		StreamChunkSize: int64(*bufferSize),
	}

	s := fs.NewFileServer(ctx, serverConfig, store, eb)
	return s
}

func startFileServer(ctx context.Context, serverId string, store *storage.Store, eb *events.EventBus) *fs.FileServer {
	server := makeServer(ctx, ":5060", serverId, store, eb)

	go func() {
		if err := server.Start(); err != nil {
			logger.Error("error while starting file server", "reason", err.Error())
		}
	}()

	return server
}

func startApi(ctx context.Context, addr string, store *storage.Store, f *fs.FileServer, eb *events.EventBus) *api.Api {
	apiServer := api.NewApi(ctx, f, store, eb, api.Config{
		Addr:     addr,
		BasePath: "/api",
		Verbose:  *verbose,
		Logger:   logger.WithGroup("api"),
	})

	go func() {
		err := apiServer.Start()
		panic(err)
	}()

	return apiServer
}

func parseArgs() {
	parser := argparse.NewParser("print", "Prints the provided string to stdout")
	storageRoot = parser.String("r", "storage-root", &argparse.Options{Required: false, Default: path.Join(homeDir, ".dfs"), Help: "The file system path to be used to store data"})
	bufferSize = parser.Int("b", "buffer-size", &argparse.Options{Help: "The buffer-size to use", Default: 4096})
	verbose = parser.Flag("v", "verbose", &argparse.Options{Help: "Enable verbose logs", Default: true})
	if err := parser.Parse(os.Args); err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
}

func setupLogger() {
	if *verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	logger = slog.Default().WithGroup("dfs")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	parseArgs()
	setupLogger()
	serverId, err := os.Hostname()
	if err != nil {
		serverId = uuid.NewString() // todo: rethink this
	}

	serverGroup := fmt.Sprintf("fs-%s", serverId)
	hash := md5.Sum([]byte(serverGroup))
	root, err := filepath.Abs(path.Join(*storageRoot, hex.EncodeToString(hash[:])))
	if err != nil {
		panic(err)
	}

	eb := makeEventBus(ctx)
	store := makeStore(root)
	fileServer := startFileServer(ctx, serverId, store, eb)
	startApi(ctx, ":8000", store, fileServer, eb)

	endCh := make(chan os.Signal, 1)
	signal.Notify(endCh, os.Interrupt, syscall.SIGTERM)
	ticker := time.NewTicker(5 * time.Second)
	for now := range ticker.C {
		eb.Publish(events.Event{
			Target: storage.EvAllEvents,
			Id:     now.String(),
			Data: map[string]any{
				"foo": "bar",
			},
		})
	}
	select {
	case <-ctx.Done():
		ticker.Stop()
		logger.Info("Shutting down")
		return
	case <-endCh:
		cancel()
	}
}
