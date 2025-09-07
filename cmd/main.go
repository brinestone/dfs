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
	"syscall"

	"github.com/akamensky/argparse"

	"github.com/google/uuid"

	"github.com/brinestone/dfs/api"
	"github.com/brinestone/dfs/fs"
	"github.com/brinestone/dfs/p2p"
	"github.com/brinestone/dfs/storage"
)

var (
	storageRoot *string
	bufferSize  *int
	homeDir, _  = os.UserHomeDir()
	logger      = slog.Default().WithGroup("dfs")
	ctx         = context.Background()
)

func makeServer(ctx context.Context, listenAddr string, id string) (*fs.FileServer, context.CancelFunc) {
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
	tcpTransport := p2p.NewTcpTransport(tcpTransportConfig)
	serverGroup := fmt.Sprintf("fs-%s", id)
	hash := md5.Sum([]byte(serverGroup))
	root := path.Join(*storageRoot, hex.EncodeToString(hash[:]))

	c, cancel := context.WithCancel(ctx)

	serverConfig := fs.FileServerConfig{
		ListenAddr:      listenAddr,
		StorageRoot:     root,
		KeyTransformer:  storage.CASKeyTransformer(root),
		Transport:       tcpTransport,
		Id:              id,
		Context:         c,
		Logger:          fsLogger.WithGroup("server"),
		StreamChunkSize: int64(*bufferSize),
	}

	s := fs.NewFileServer(serverConfig)
	return s, cancel
}

func startFileServer() *fs.FileServer {
	serverId, err := os.Hostname()
	if err != nil {
		serverId = uuid.NewString() // todo: rethink this
	}
	server, cancel := makeServer(ctx, ":5060", serverId)

	go func() {
		defer cancel()
		if err = server.Start(); err != nil {
			logger.Error("error while starting file server", "reason", err.Error())
		}
	}()

	return server
}

func startApi(addr string) *api.Api {
	apiServer := api.NewApi(api.Config{
		Addr:   addr,
		Ctx:    ctx,
		Logger: logger.WithGroup("api"),
	})

	go func() {
		err := apiServer.Start()
		if err != nil {
			defer apiServer.Stop()
			logger.Error("error while starting API", "err", err.Error())
			return
		}
	}()

	return apiServer
}

func parseArgs() {
	parser := argparse.NewParser("print", "Prints the provided string to stdout")
	storageRoot = parser.String("r", "storage-root", &argparse.Options{Required: false, Default: path.Join(homeDir, ".dfs"), Help: "The file system path to be used to store data"})
	bufferSize = parser.Int("b", "buffer-size", &argparse.Options{Help: "The buffer-size to use", Default: 4096})
	if err := parser.Parse(os.Args); err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
}

func main() {
	parseArgs()
	apiServer := startApi(":8000")
	fileServer := startFileServer()

	apiServer.Config.FileServer = fileServer
	endCh := make(chan os.Signal, 1)
	signal.Notify(endCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		apiServer.Stop()
		return
	case <-endCh:
		logger.Info("Shutting down")
		err := fileServer.Shutdown()
		if err != nil {
			logger.Error("error while stopping File Api", "cause", err.Error())
		}
		err = apiServer.Stop()
		if err != nil {
			logger.Error("error while stopping API", "cause", err.Error())
		}
		return
	}
}
