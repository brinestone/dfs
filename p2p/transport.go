package p2p

type Rpc struct {
	Payload []byte
	Stream  bool
	From    string
}

type Peer interface {
	Close() error
}

type Transport interface {
	ListenAndAccept() error
	Dial(string) error
	Consume() <-chan Rpc
	Close() error
}

type HandshakeFunc func(Peer) error

var NoopHandshaker HandshakeFunc = func(p Peer) error { return nil }
