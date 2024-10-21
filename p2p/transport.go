package p2p

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/brinestone/dfs/crypt"
	"github.com/brinestone/dfs/encoding"
)

type Rpc struct {
	Payload []byte
	Stream  bool
	From    string
}

type Peer interface {
	net.Conn
	Inbound() bool
	Send([]byte) error
	SetEncoder(encoding.Encoder)
	SetDecoder(encoding.Decoder)
}

type Transport interface {
	ListenAndAccept() error
	Dial(string) error
	Consume() <-chan Rpc
	Close() error
	OnPeerConnected(func(Peer))
	OnPeerDisconnected(func(Peer))
}

var ErrHandshakeTimeout = errors.New("handshake timeout")

type HandshakeFunc func(Peer) error

func NoopHandshaker(config encoding.EncodingConfig) HandshakeFunc {
	return func(p Peer) error {
		p.SetEncoder(encoding.NewPlainEncoderDecoder(config))
		p.SetDecoder(encoding.NewPlainEncoderDecoder(config))
		return nil
	}
}

// Performs exchanging of public keys between peers and sets the encoder for the peer.
func KeyExchangeHandShaker(l *slog.Logger, timeout time.Duration, cnf encoding.EncodingConfig) HandshakeFunc {
	var sendPublicKey = func(p Peer, key *ecdh.PublicKey) error {

		kb, err := x509.MarshalPKIXPublicKey(key)
		if err != nil {
			return err
		}

		if _, err = p.Write(kb); err != nil {
			return err
		}
		return nil
	}

	var readPeerPublicKey = func(ctx context.Context, p Peer) (*ecdh.PublicKey, error) {
		_ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		var publicKey *ecdh.PublicKey

		for {
			if _ctx.Err() != nil && errors.Is(_ctx.Err(), context.DeadlineExceeded) {
				return nil, _ctx.Err()
			}

			key := make([]byte, 1024)
			n, err := p.Read(key)
			if n == 0 && errors.Is(err, io.EOF) {
				continue
			} else if errors.Is(err, net.ErrClosed) {
				return nil, err
			}

			t1, err := x509.ParsePKIXPublicKey(key[:n])
			if err != nil {
				return nil, err
			}

			t2 := t1.(*ecdsa.PublicKey)
			publicKey, err = t2.ECDH()
			if err != nil {
				return nil, err
			}
			break
		}

		return publicKey, nil
	}

	return func(p Peer) error {
		private, localPublic, err := crypt.KeyGen()
		var remotePublic *ecdh.PublicKey
		if err != nil {
			return err
		}

		if p.Inbound() {
			if remotePublic, err = readPeerPublicKey(context.TODO(), p); err != nil {
				return err
			}
			if err := sendPublicKey(p, localPublic); err != nil {
				return err
			}
		} else {
			if err := sendPublicKey(p, localPublic); err != nil {
				return err
			}
			if remotePublic, err = readPeerPublicKey(context.TODO(), p); err != nil {
				return err
			}
		}

		secret, err := crypt.SecGen(private, remotePublic)
		if err != nil {
			return err
		}

		ec := encoding.NewSecureEncoderDecoder(secret, cnf)

		p.SetEncoder(ec)
		p.SetDecoder(ec)

		return nil
	}
}
