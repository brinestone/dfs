package p2p

import (
	"errors"
	"log"
	"net"
)

// TcpPeer represents a remote host on the network
type TcpPeer struct {
	// The underlying connection to the peer host
	conn net.Conn
	// true if the connection was dialed and false if otherwise
	outbound bool
}

func (t *TcpPeer) Close() error {
	return t.conn.Close()
}

func newTcpPeer(conn net.Conn, outbound bool) *TcpPeer {
	return &TcpPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TcpTransportConfig struct {
	ListenAddr string
	Handshaker HandshakeFunc
	Logger     *log.Logger
	Decoder    Decoder
	OnPeer     func(Peer) error
}

type TcpTransport struct {
	TcpTransportConfig
	listener net.Listener
	rpcch    chan Rpc
}

func (t *TcpTransport) Dial(addr string) error {
	t.Logger.Printf("attempting to dial %s\n", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	peer := newTcpPeer(conn, true)
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

	// t.Logger.Printf("tcp-transport::listen -> \"%s\"👂\n", strings.Trim(t.ListenAddr, "\n"))
	go t.startAcceptLoop()

	return err
}

func (t *TcpTransport) Close() error {
	close(t.rpcch)
	return t.listener.Close()
}

func NewTcpTransport(config TcpTransportConfig) *TcpTransport {
	// defer config.Logger.Println("tcp-transport::new\t✅")
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
			t.Logger.Printf("tcp accept error: %v\n", err)
		}
		peer := newTcpPeer(conn, false)
		t.Logger.Printf("new incoming peer connection: %+v\n", peer.conn.RemoteAddr().String())

		if err := t.TcpTransportConfig.Handshaker(peer); err != nil {
			t.Logger.Println(err)
			_ = peer.Close()
			continue
		}

		if t.TcpTransportConfig.OnPeer != nil {
			if err := t.OnPeer(peer); err != nil {
				t.Logger.Fatalf("onpeer failed: \"%v\"\n", err)
			}
		}
		go t.startPeerReadLoop(peer)
	}
}

func (t *TcpTransport) startPeerReadLoop(peer *TcpPeer) {
	defer func() {
		t.Logger.Printf("peer dropped: %s", peer.conn.RemoteAddr().String())
	}()
	rpc := Rpc{}
	for {
		if err := t.TcpTransportConfig.Decoder.Decode(peer.conn, &rpc); err != nil {
			if errors.Is(err, net.ErrClosed) || err.Error() == "EOF" {
				break
			}
			t.Logger.Println(err)
			continue
		}
		rpc.From = peer.conn.RemoteAddr().String()
		t.rpcch <- rpc
	}
}
