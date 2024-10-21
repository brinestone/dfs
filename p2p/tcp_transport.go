package p2p

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/brinestone/dfs/encoding"
)

// TcpPeer represents a remote host on the network
type TcpPeer struct {
	// The underlying connection to the peer host
	net.Conn
	// true if the connection was dialed and false if otherwise
	outbound bool
	// The encoder for this peer
	encoder encoding.Encoder
	// The decoder for this peer
	decoder encoding.Decoder
}

func (t *TcpPeer) SetEncoder(e encoding.Encoder) {
	t.encoder = e
}

func (t *TcpPeer) SetDecoder(d encoding.Decoder) {
	t.decoder = d
}

func (t *TcpPeer) Inbound() bool {
	return !t.outbound
}

func (t *TcpPeer) Send(data []byte) error {
	out, err := t.encoder.Encode(data)
	if err != nil {
		return err
	}

	if _, err = t.Write(out); err != nil {
		return err
	}
	return nil
}

func newTcpPeer(conn net.Conn, outbound bool) *TcpPeer {
	return &TcpPeer{
		Conn:     conn,
		outbound: outbound,
	}
}

type TcpTransportConfig struct {
	ListenAddr    string
	ExecHandshake HandshakeFunc
	Logger        *slog.Logger
}

type TcpTransport struct {
	TcpTransportConfig
	listener            net.Listener
	rpcch               chan Rpc
	connectCallbacks    []func(Peer)
	disconnectCallbacks []func(Peer)

	muIsClosed sync.RWMutex
	isClosed   bool
}

func (t *TcpTransport) OnPeerConnected(f func(Peer)) {
	t.connectCallbacks = append(t.connectCallbacks, f)
}

func (t *TcpTransport) OnPeerDisconnected(f func(Peer)) {
	t.disconnectCallbacks = append(t.disconnectCallbacks, f)
}

func (t *TcpTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	peer := newTcpPeer(conn, true)
	if err := t.ExecHandshake(peer); err != nil {
		defer peer.Close()
		return err
	}

	if len(t.connectCallbacks) > 0 {
		for _, callback := range t.connectCallbacks {
			callback(peer)
		}
	}

	go t.startPeerReadLoop(peer)

	return nil
}

func (t *TcpTransport) Consume() <-chan Rpc {
	return t.rpcch
}

func (t *TcpTransport) ListenAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	t.muIsClosed.Lock()
	defer t.muIsClosed.Unlock()

	t.isClosed = false
	go t.startAcceptLoop()

	return err
}

func (t *TcpTransport) Close() error {
	var ans error
	t.muIsClosed.Lock()

	t.isClosed = true
	t.muIsClosed.Unlock()
	close(t.rpcch)
	if t.listener != nil {
		ans = t.listener.Close()
	}
	return ans
}

func NewTcpTransport(config TcpTransportConfig) *TcpTransport {
	return &TcpTransport{
		TcpTransportConfig: config,
		rpcch:              make(chan Rpc),
		isClosed:           true,
	}
}

func (t *TcpTransport) startAcceptLoop() {
	for !t.isClosed {
		conn, err := t.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				break
			}
			t.Logger.Error(err.Error())
			continue
		}
		peer := newTcpPeer(conn, false)

		if err := t.ExecHandshake(peer); err != nil {
			_ = peer.Close()
			continue
		}

		go t.startPeerReadLoop(peer)
	}
}

func (t *TcpTransport) startPeerReadLoop(peer *TcpPeer) {
	if len(t.connectCallbacks) > 0 {
		for _, f := range t.connectCallbacks {
			go f(peer)
		}
	}
	defer func() {
		if len(t.disconnectCallbacks) > 0 {
			for _, f := range t.disconnectCallbacks {
				go f(peer)
			}
		}
	}()

	rpc := Rpc{}
	buf := new(bytes.Buffer)
	for !t.isClosed {
		buf.Reset()
		if err := peer.decoder.DecodeStream(peer, buf); err != nil {
			t.Logger.Error("tcp decode error", "msg", err.Error())
			continue
		}
		if !t.isClosed {
			rpc.Payload = buf.Bytes()
			rpc.From = peer.RemoteAddr().String()
			t.rpcch <- rpc
		}
	}
}
