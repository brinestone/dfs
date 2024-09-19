package main

import (
	"log"

	"github.com/brinestone/dfs/p2p"
)

var logger = log.Default()

func main() {
	logger.SetPrefix("[DFS]\t")
	listenAddr := ":3000"
	tr := p2p.NewTcpTransport(p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Logger:     logger,
		Handshaker: p2p.NoopHandshaker,
		Decoder:    p2p.DefaultDecoder{},
		OnPeer: func(p p2p.Peer) error {
			return nil
		},
	})

	go func() {
		for msg := range tr.Consume() {
			logger.Printf("message: %+v\n", msg)
		}
	}()

	if err := tr.ListenAndAccept(); err != nil {
		panic(err)
	}
	select {}
}
