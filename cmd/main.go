package main

import (
	"context"
	"crypto/md5"
	"embed"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net"
	"os"
	"path"
	"time"

	"github.com/brinestone/dfs/p2p"
	"github.com/brinestone/dfs/server"
	"github.com/brinestone/dfs/storage"
)

var bufferSize = flag.Int64("bs", 4096, "Buffer Size")
var serverCount = flag.Int("cnt", 2, "Number of servers to spawn")
var privateKey = flag.String("pk", "", "Private key for decryption")
var publicKey = flag.String("pbk", "", "Public key for encryption")
var logger = slog.Default().WithGroup("DFS")
var storageRoot = flag.String("root", path.Join(os.Getenv("HOME"), ".dfs"), "Filesystem path to be used as root.")

func randomPort() int {
	var ans int
	var failCount = 0
	for failCount <= 50 {
		ans = int(math.Max(1025, float64(rand.Int31n(64_512))))
		if l, err := net.Listen("tcp", fmt.Sprintf(":%d", ans)); err == nil {
			l.Close()
			break
		}
		failCount++
	}
	if ans == 0 {
		panic(errors.New("cannot find available tcp port"))
	}
	return ans
}

func makeServer(ctx context.Context, listenAddr string, id string) (*server.FileServer, context.CancelFunc) {
	var encoder p2p.Encoder
	var decoder p2p.Decoder
	var encodingConfig p2p.EncodingConfig = p2p.EncodingConfig{
		BufferSize: *bufferSize,
	}

	if len(*privateKey) == 0 || len(*publicKey) == 0 {
		encoder = p2p.NewPlainEncoder(encodingConfig)
		decoder = p2p.NewPlainDecoder(encodingConfig)
	} else {
		decoder = p2p.NewSecuredDecoder(p2p.EncodingConfig{
			BufferSize: *bufferSize,
		}, *privateKey)
		encoder = p2p.NewSecureEncoder(*publicKey, encodingConfig)
	}

	tcpTransportConfig := p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Handshaker: p2p.NoopHandshaker,
		Decoder:    decoder,
		Encoder:    encoder,
		Logger:     logger,
		// TODO: onPeer func
	}
	tcpTransport := p2p.NewTcpTransport(tcpTransportConfig)
	serverGroup := fmt.Sprintf("server-%s", id)
	hash := md5.Sum([]byte(serverGroup))
	root := path.Join(*storageRoot, hex.EncodeToString(hash[:]))

	c, cancel := context.WithCancel(ctx)

	serverConfig := server.FileServerConfig{
		Encoder:         encoder,
		Decoder:         decoder,
		ListenAddr:      listenAddr,
		StorageRoot:     root,
		KeyTransformer:  storage.CASKeyTransformer(root),
		Transport:       tcpTransport,
		Id:              id,
		Context:         c,
		Logger:          logger.WithGroup(serverGroup),
		StreamChunkSize: *bufferSize,
	}

	s, err := server.NewFileServer(serverConfig)
	if err != nil {
		panic(err)
	}

	return s, cancel
}

func spawnServers(ctx context.Context, cb func(*server.FileServer)) {
	logger.Debug("Spawning servers", "server-count", *serverCount)
	var addrs = make([]string, *serverCount)
	var c int

	// Find available port
	for c < *serverCount {
		t := randomPort()

		addr := fmt.Sprintf("0.0.0.0:%d", t)
		if l, err := net.Listen("tcp", addr); err == nil {
			addrs[c] = addr
			l.Close()
			c++
			continue
		}
	}

	for i, addr := range addrs {
		s, cancel := makeServer(ctx, addr, fmt.Sprintf("%d", i+1))
		peers := addrs[:i]
		go func() {
			defer cancel()
			if err := s.Start(peers...); err != nil {
				logger.Error(err.Error())
			}
		}()
		go cb(s)
	}

}

//go:embed lorem.txt
var sampleData embed.FS

func main() {
	flag.Parse()
	ctx := context.Background()

	spawnServers(ctx, func(fs *server.FileServer) {
		time.Sleep(time.Second * 3)
		handle, err := sampleData.Open("lorem.txt")
		if err != nil {
			panic(err)
		}

		stat, err := handle.Stat()
		if err != nil {
			panic(err)
		}

		defer handle.Close()
		err = fs.StoreData(fmt.Sprintf("server_%s", fs.Id), stat.Size(), handle)
		if err != nil {
			logger.Error("store error", "msg", err.Error())
		}
	})

	select {}
}
