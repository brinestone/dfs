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

var ErrHandshakeTimeout = errors.New("handshake timeout")

type HandshakeFunc func(Peer) (encoding.Encoder, encoding.Decoder, error)

func NoopHandshaker(config encoding.EncodingConfig) HandshakeFunc {
	return func(p Peer) (encoding.Encoder, encoding.Decoder, error) {
		return encoding.NewPlainEncoderDecoder(config), encoding.NewPlainEncoderDecoder(config), nil
	}
}

// Performs exchanging of public keys between peers and sets the encoder for the peer.
func KeyExchangeHandShaker(l *slog.Logger, timeout time.Duration, cnf encoding.EncodingConfig) HandshakeFunc {
	var sendPublicKey = func(ctx context.Context, p Peer, key *ecdh.PublicKey) error {

		kb, err := x509.MarshalPKIXPublicKey(key)
		if err != nil {
			return err
		}

		_, err = p.Write(kb)
		if errors.Is(err, net.ErrClosed) || (ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)) {
			return ErrHandshakeTimeout
		}
		return nil
	}

	var readPeerPublicKey = func(ctx context.Context, p Peer) (*ecdh.PublicKey, error) {

		var publicKey *ecdh.PublicKey

		for {
			if ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, ErrHandshakeTimeout
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

	return func(p Peer) (encoding.Encoder, encoding.Decoder, error) {
		ctx, cancel := context.WithTimeout(context.TODO(), timeout)
		defer cancel()

		private, localPublic, err := crypt.KeyGen()
		var remotePublic *ecdh.PublicKey
		if err != nil {
			return nil, nil, err
		}

		if p.Inbound() {
			if remotePublic, err = readPeerPublicKey(context.TODO(), p); err != nil {
				return nil, nil, err
			}
			if err := sendPublicKey(ctx, p, localPublic); err != nil {
				return nil, nil, err
			}
		} else {
			if err := sendPublicKey(ctx, p, localPublic); err != nil {
				return nil, nil, err
			}
			if remotePublic, err = readPeerPublicKey(context.TODO(), p); err != nil {
				return nil, nil, err
			}
		}

		secret, err := crypt.SecGen(private, remotePublic)
		if err != nil {
			return nil, nil, err
		}

		ec := encoding.NewSecureEncoderDecoder(secret, cnf)

		return ec, ec, nil
	}
}
