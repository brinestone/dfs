package p2p_test

import (
	"context"
	cryptoRand "crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/brinestone/dfs/p2p"

	"github.com/stretchr/testify/assert"
)

var logger = slog.Default().WithGroup("[store_test]")

func randomPort(t *testing.T) int {
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
		t.Errorf("cannot find available tcp port")
	}
	return ans
}

func makeTransportConfig(port int) p2p.TcpTransportConfig {
	listenAddr := fmt.Sprintf(":%d", port)
	return p2p.TcpTransportConfig{
		ListenAddr: listenAddr,
		Logger:     logger,
		Handshaker: p2p.NoopHandshaker,
		Decoder: p2p.PlainDecoder{
			EncodingConfig: encodingConfig,
		},
	}
}

func TestTcpTransportTests(t *testing.T) {
	port := randomPort(t)
	config := makeTransportConfig(port)
	tr := p2p.NewTcpTransport(config)
	assert.NotNil(t, tr)
	t.Cleanup(func() {
		tr.Close()
	})
	t.Run("Test_NewTcpTransport", func(t *testing.T) {
		testTcpTransport(t, config.ListenAddr, tr)
	})
	t.Run("Test_ListenAndAccept", func(t *testing.T) {
		testListenAndAccept(t, tr)
	})
	t.Run("Test_ListenAndAccept_With_Used_Port", func(t *testing.T) {
		testListenAndAcceptWithUsedPort(t)
	})
	t.Run("Test_Consume", func(t *testing.T) {
		testConsume(t, tr)
	})
	t.Run("Test_Dial", func(t *testing.T) {
		testDial(t, tr)
	})

}

func testDial(t *testing.T, tr *p2p.TcpTransport) {
	port := randomPort(t)
	config := makeTransportConfig(port)
	t2 := p2p.NewTcpTransport(config)
	assert.Nil(t, t2.ListenAndAccept())

	t.Cleanup(func() {
		t2.Close()
	})

	assert.Nil(t, tr.Dial(config.ListenAddr))
	t.Run("withDataSending", func(t *testing.T) {
		testDial_With_DataSending(t, tr, t2)
	})
	t.Run("witNoTarget", func(t *testing.T) {
		testDial_With_NoReceiver(t, tr)
	})
}

func testDial_With_NoReceiver(t *testing.T, tr *p2p.TcpTransport) {
	port := randomPort(t)
	config := makeTransportConfig(port)

	assert.NotNil(t, tr.Dial(config.ListenAddr))
}

func testDial_With_DataSending(t *testing.T, t1 *p2p.TcpTransport, t2 *p2p.TcpTransport) {
	ctx, cancel := context.WithTimeout(context.TODO(), 50*time.Millisecond)
	t.Cleanup(cancel)
	data := make([]byte, encodingConfig.BufferSize)
	_, err := io.ReadFull(cryptoRand.Reader, data)
	if err != nil {
		t.Error(err)
	}

	t1.OnPeerConnected(func(p p2p.Peer) {
		p.Send(data)
	})

	if err := t1.Dial(t2.ListenAddr); err != nil {
		t.Error(err)
	}

	select {
	case rpc, ok := <-t2.Consume():
		if ok {
			assert.EqualValues(t, data, rpc.Payload)
		}
	case <-ctx.Done():
		t.Errorf("timeout error")
		return
	}
}

func TestClose(t *testing.T) {
	config := makeTransportConfig(randomPort(t))

	tr := p2p.NewTcpTransport(config)
	assert.Nil(t, tr.ListenAndAccept())
	assert.Nil(t, tr.Close())
}

func testListenAndAcceptWithUsedPort(t *testing.T) {
	var port = randomPort(t)
	var config = makeTransportConfig(port)
	l, _ := net.Listen("tcp", config.ListenAddr)
	t.Cleanup(func() {
		l.Close()
	})
	tr := p2p.NewTcpTransport(config)
	assert.NotNil(t, tr.ListenAndAccept())
}

func testListenAndAccept(t *testing.T, tr *p2p.TcpTransport) {
	assert.Nil(t, tr.ListenAndAccept())
}

func testTcpTransport(t *testing.T, listenAddr string, tr *p2p.TcpTransport) {
	assert.Equal(t, listenAddr, tr.TcpTransportConfig.ListenAddr)
}

func testConsume(t *testing.T, tr *p2p.TcpTransport) {
	ch := tr.Consume()
	assert.NotNil(t, ch)
}
