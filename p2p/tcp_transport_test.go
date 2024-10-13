package p2p_test

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"testing"

	"github.com/brinestone/dfs/p2p"

	"github.com/stretchr/testify/assert"
)

var logger = slog.Default().WithGroup("[store_test]")

func TestNewTcpTransport(t *testing.T) {
	portNumber := int(math.Max(1025, float64(rand.Int31n(64_512))))
	listenAddr := fmt.Sprintf(":%d", portNumber)
	config := p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Logger:     logger,
		Handshaker: p2p.NoopHandshaker,
		Decoder:    p2p.DefaultDecoder{},
	}

	tr := p2p.NewTcpTransport(config)

	assert.Equal(t, listenAddr, tr.TcpTransportConfig.ListenAddr)
	assert.Nil(t, tr.ListenAndAccept())
}
