package p2p

import (
	"io"
)

type Decoder interface {
	Decode(io.Reader, *Rpc) error
}

type DefaultDecoder struct{}

const incomingStream byte = 0x2

func (dec DefaultDecoder) Decode(r io.Reader, msg *Rpc) error {
	peekBuff := make([]byte, 1)
	if _, err := r.Read(peekBuff); err != nil {
		return err
	}

	stream := peekBuff[0] == incomingStream
	msg.Stream = stream
	if stream {
		return nil
	}

	buff := make([]byte, 1028)

	n, err := r.Read(buff)
	if err != nil {
		return err
	}
	msg.Payload = make([]byte, 1+n)
	msg.Payload[0] = peekBuff[0]

	for i, b := range buff[:n] {
		msg.Payload[i+1] = b
	}

	return nil
}
