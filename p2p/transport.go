package p2p

import "net"

type Rpc struct {
	Payload []byte
	Stream  bool
	From    string
}

type Peer interface {
	net.Conn
	Inbound() bool
	Send([]byte) error
}

type Transport interface {
	ListenAndAccept() error
	Dial(string) error
	Consume() <-chan Rpc
	Close() error
	OnPeerConnected(func(Peer))
	OnPeerDisconnected(func(Peer))
}

type HandshakeFunc func(Peer) error

var NoopHandshaker HandshakeFunc = func(p Peer) error { return nil }
