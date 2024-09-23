package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/brinestone/dfs/p2p"
	"github.com/brinestone/dfs/server"
	"github.com/brinestone/dfs/storage"
)

var logger = log.Default()
var listenPort = flag.Int("p", 3000, "Port to listen on")
var storageRoot = flag.String("sr", path.Join(os.Getenv("HOME"), ".dfs"), "Filesystem root to use")
var nodesStr = flag.String("n", "", "Peer servers: [node1,node2,node3]")
var ctx = context.Background()

func makeServer() *server.FileServer {
	listenAddr := fmt.Sprintf(":%d", *listenPort)
	tcpTransportConfig := p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Handshaker: p2p.NoopHandshaker,
		Decoder:    p2p.DefaultDecoder{},
		Logger:     logger,
		// TODO: onPeer func
	}
	tcpTransport := p2p.NewTcpTransport(tcpTransportConfig)

	serverConfig := server.FileServerConfig{
		ListenAddr:   listenAddr,
		StorageRoot:  path.Join(*storageRoot, base64.StdEncoding.EncodeToString([]byte(listenAddr))),
		TransformKey: storage.CASKeyTransformer,
		Transport:    tcpTransport,
		Context:      ctx,
		Nodes:        strings.Split(*nodesStr, ","),
		Logger:       logger,
	}

	s := server.NewFileServer(serverConfig)
	return s
}

func main() {
	flag.Parse()
	logger.SetPrefix("[DFS]\t")
	s := makeServer()

	if err := s.Start(); err != nil {
		logger.Fatal(err)
	}

}
