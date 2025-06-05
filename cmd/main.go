package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/brinestone/dfs/api"
	"github.com/brinestone/dfs/fs"
	"github.com/brinestone/dfs/p2p"
	"github.com/brinestone/dfs/storage"
)

var (
	storageRoot = flag.String("root", path.Join(homeDir, ".dfs"), "Filesystem path to be used as root.")
	bufferSize  = flag.Int64("bs", 4096, "Buffer Size")
	homeDir, _  = os.UserHomeDir()
	logger      = slog.Default().WithGroup("DFS")
	ctx         = context.Background()
)

func makeServer(ctx context.Context, listenAddr string, id string) (*fs.FileServer, context.CancelFunc) {
	tcpTransportConfig := p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Handshake:  p2p.NoopHandshake,
		Decoder: p2p.DefaultDecoder{
			EncodingConfig: p2p.EncodingConfig{
				BufferSize: *bufferSize,
			},
		},
		Logger: logger.WithGroup("FS/TCP"),
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
		Logger:          logger.WithGroup("FS/Server"),
		StreamChunkSize: *bufferSize,
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

func startApi(addr string) *api.Server {
	apiServer := api.NewServer(api.Config{
		Addr:   addr,
		Ctx:    ctx,
		Logger: logger.WithGroup("API"),
	})

	go func() {
		err := apiServer.Start()
		if err != nil {
			defer apiServer.Stop()
			logger.Error("error while starting API", err.Error())
			return
		}
	}()

	return apiServer
}

func main() {
	flag.Parsed()
	ctx = context.Background()
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
			logger.Error("error while stopping File Server", "cause", err.Error())
		}
		err = apiServer.Stop()
		if err != nil {
			logger.Error("error while stopping API", "cause", err.Error())
		}
		return
	}
}
