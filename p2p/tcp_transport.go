package p2p

import (
	"errors"
	"log/slog"
	"net"
)

// TcpPeer represents a remote host on the network
type TcpPeer struct {
	// The underlying connection to the peer host
	net.Conn
	// true if the connection was dialed and false if otherwise
	outbound bool
}

func (t *TcpPeer) Inbound() bool {
	return !t.outbound
}

func (t *TcpPeer) Send(data []byte) error {
	if _, err := t.Write(data); err != nil {
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
	ListenAddr string
	Handshaker HandshakeFunc
	Logger     *slog.Logger
	Decoder    Decoder
}

type TcpTransport struct {
	TcpTransportConfig
	listener            net.Listener
	rpcch               chan Rpc
	connectCallbacks    []func(Peer)
	disconnectCallbacks []func(Peer)
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
	if err := t.Handshaker(peer); err != nil {
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
	go t.startAcceptLoop()

	return err
}

func (t *TcpTransport) Close() error {
	close(t.rpcch)
	return t.listener.Close()
}

func NewTcpTransport(config TcpTransportConfig) *TcpTransport {
	return &TcpTransport{
		TcpTransportConfig: config,
		rpcch:              make(chan Rpc),
	}
}

func (t *TcpTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			t.Logger.Error(err.Error())
		}
		peer := newTcpPeer(conn, false)

		if err := t.Handshaker(peer); err != nil {
			t.Logger.Error(err.Error())
			_ = peer.Close()
			continue
		}

		if len(t.connectCallbacks) > 0 {
			for _, f := range t.connectCallbacks {
				go f(peer)
			}
		}
		go t.startPeerReadLoop(peer)
	}
}

func (t *TcpTransport) startPeerReadLoop(peer *TcpPeer) {
	defer func() {
		if len(t.disconnectCallbacks) > 0 {
			for _, f := range t.disconnectCallbacks {
				go f(peer)
			}
		}
	}()

	rpc := Rpc{}
	for {
		if err := t.Decoder.Decode(peer, &rpc); err != nil {
			if errors.Is(err, net.ErrClosed) || err.Error() == "EOF" {
				break
			}
			t.Logger.Error(err.Error())
			continue
		}
		rpc.From = peer.RemoteAddr().String()
		t.rpcch <- rpc
	}
}
