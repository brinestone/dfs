package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
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

var serverCount = flag.Int("cnt", 2, "Number of servers to spawn")

var logger = slog.Default().WithGroup("DFS")

var storageRoot = flag.String("root", path.Join(os.Getenv("HOME"), ".dfs"), "Filesystem path to be used as root.")

func makeServer(ctx context.Context, listenAddr string, id string) (*server.FileServer, context.CancelFunc) {
	tcpTransportConfig := p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Handshaker: p2p.NoopHandshaker,
		Decoder:    p2p.DefaultDecoder{},
		Logger:     logger,
		// TODO: onPeer func
	}
	tcpTransport := p2p.NewTcpTransport(tcpTransportConfig)
	serverGroup := fmt.Sprintf("server-%s", id)
	hash := md5.Sum([]byte(serverGroup))
	root := path.Join(*storageRoot, hex.EncodeToString(hash[:]))

	c, cancel := context.WithCancel(ctx)

	serverConfig := server.FileServerConfig{
		ListenAddr:     listenAddr,
		StorageRoot:    root,
		KeyTransformer: storage.CASKeyTransformer(root),
		Transport:      tcpTransport,
		Id:             id,
		Context:        c,
		Logger:         logger.WithGroup(serverGroup),
	}

	s := server.NewFileServer(serverConfig)
	return s, cancel
}

func spawnServers(ctx context.Context, cb func(*server.FileServer)) {
	logger.Debug("Spawning servers", "server-count", *serverCount)
	var addrs = make([]string, *serverCount)
	var c int

	// Find available port
	for c < *serverCount {
		t := int(math.Max(1025, float64(rand.Intn(65_536))))

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

func main() {
	gob.Register(server.Message{Payload: []byte{}})
	flag.Parse()
	ctx := context.Background()

	spawnServers(ctx, func(fs *server.FileServer) {
		time.Sleep(time.Second * 3)
		data := bytes.NewReader([]byte(fmt.Sprintf("some random bytes for server: %s\n", fs.Id)))
		fs.StoreData(fmt.Sprintf("%s--%s", fs.Id, time.Now().String()), data)
	})

	select {}
}
