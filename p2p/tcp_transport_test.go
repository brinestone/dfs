package p2p_test

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"

	"github.com/brinestone/dfs/p2p"

	"github.com/stretchr/testify/assert"
)

var logger = log.New(os.Stdout, "[tcp_transport_test]\t", log.LstdFlags)

func TestNewTcpTransport(t *testing.T) {
	portNumber := rand.Int31n(64_512) + 1025
	listenAddr := fmt.Sprintf(":%d", portNumber)
	config := p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Logger:     logger,
		Handshaker: p2p.NoopHandshaker,
		Decoder:    p2p.DefaultDecoder{},
		OnPeer:     p2p.NoopHandshaker,
	}

	tr := p2p.NewTcpTransport(config)

	assert.Equal(t, listenAddr, tr.TcpTransportConfig.ListenAddr)
	assert.Nil(t, tr.ListenAndAccept())
}
