package p2p

import (
	"io"
)

type EncodingConfig struct {
	BufferSize int64
}

type Decoder interface {
	Decode(io.Reader, *Rpc) error
}

type DefaultDecoder struct {
	EncodingConfig
}

func (dec DefaultDecoder) Decode(in io.Reader, msg *Rpc) error {
	buf := make([]byte, dec.BufferSize*3)

	n, err := in.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]
	return nil
}
