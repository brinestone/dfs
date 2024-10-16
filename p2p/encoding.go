package p2p

import (
	"io"

	"github.com/brinestone/dfs/encdec"
)

type EncodingConfig struct {
	BufferSize int64
}

type Encoder interface {
	Encode(io.Writer, []byte) (int, error)
}

type SecureEncoder struct {
	publicKey string
}

func (ee SecureEncoder) Encode(out io.Writer, data []byte) (int, error) {
	ciphertext, err := encdec.Encrypt(ee.publicKey, data)
	if err != nil {
		return 0, err
	}

	return out.Write(ciphertext)
}

func NewSecureEncoder(publicKey string) SecureEncoder {
	return SecureEncoder{
		publicKey: publicKey,
	}
}

type PlainEncoder struct {
}

func (pe PlainEncoder) Encode(out io.Writer, data []byte) (int, error) {
	return out.Write(data)
}

// Creates a new PlainEncoder
func NewPlainEncoder() PlainEncoder {
	return PlainEncoder{}
}

type Decoder interface {
	Decode(io.Reader, *Rpc) error
}

type SecuredDecoder struct {
	privateKey string
	EncodingConfig
}

func (ed SecuredDecoder) Decode(in io.Reader, msg *Rpc) error {
	buf := make([]byte, ed.BufferSize*4)
	n, err := in.Read(buf)
	if err != nil {
		return err
	}

	buf, err = encdec.Decrypt(ed.privateKey, buf[:n])
	if err != nil {
		return err
	}

	msg.Payload = buf

	return nil
}

// Creates a new [SecuredDecoder] using the config and private key file path
func NewSecuredDecoder(config EncodingConfig, privateKey string) SecuredDecoder {
	return SecuredDecoder{
		privateKey:     privateKey,
		EncodingConfig: config,
	}
}

type PlainDecoder struct {
	EncodingConfig
}

func (dec PlainDecoder) Decode(in io.Reader, msg *Rpc) error {
	buf := make([]byte, dec.BufferSize*3)

	n, err := in.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]
	return nil
}

func NewPlainDecoder(config EncodingConfig) PlainDecoder {
	return PlainDecoder{
		EncodingConfig: config,
	}
}
