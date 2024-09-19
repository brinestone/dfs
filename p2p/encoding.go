package p2p

import "io"

type Decoder interface {
	Decode(io.Reader, *Rpc) error
}

type DefaultDecoder struct{}

const incomingStream = 0x2

func (dec DefaultDecoder) Decode(r io.Reader, msg *Rpc) error {
	peekBuff := make([]byte, 1)
	if _, err := r.Read(peekBuff); err != nil {
		return err
	}

	stream := peekBuff[0] == incomingStream
	if stream {
		msg.Stream = true
		return nil
	}

	buff := make([]byte, 1028)
	n, err := r.Read(buff)
	if err != nil {
		return err
	}
	msg.Payload = buff[:n]

	return nil
}
