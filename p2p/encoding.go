package p2p

import (
	"bytes"
	"errors"
	"io"

	"github.com/brinestone/dfs/encdec"
)

type EncodingConfig struct {
	BufferSize int64
}

type Encoder interface {
	Encode(io.Writer, []byte) (int, error)
	EncodeStream(io.Reader) (io.Reader, error)
}

type SecureEncoder struct {
	publicKey string
	EncodingConfig
}

func (se SecureEncoder) EncodeStream(in io.Reader) (io.Reader, error) {
	var ans = new(bytes.Buffer)
	for {
		buf := make([]byte, se.BufferSize)
		n, err := in.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		ciphertext, err := encdec.Encrypt(se.publicKey, buf[:n])
		if err != nil {
			return nil, err
		}

		_, err = ans.Write(ciphertext)
		if err != nil {
			return nil, err
		}
	}
	return ans, nil
}

func (ee SecureEncoder) Encode(out io.Writer, data []byte) (int, error) {
	ciphertext, err := encdec.Encrypt(ee.publicKey, data)
	if err != nil {
		return 0, err
	}

	return out.Write(ciphertext)
}

func NewSecureEncoder(publicKey string, config EncodingConfig) SecureEncoder {
	return SecureEncoder{
		publicKey:      publicKey,
		EncodingConfig: config,
	}
}

type PlainEncoder struct {
	EncodingConfig
}

func (pe PlainEncoder) EncodeStream(in io.Reader) (io.Reader, error) {
	var ans = new(bytes.Buffer)
	for {
		buf := make([]byte, pe.BufferSize)
		n, err := in.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		_, err = ans.Write(buf)
		if err != nil {
			return nil, err
		}
	}
	return ans, nil
}

func (pe PlainEncoder) Encode(out io.Writer, data []byte) (int, error) {
	return out.Write(data)
}

// Creates a new PlainEncoder
func NewPlainEncoder(config EncodingConfig) PlainEncoder {
	return PlainEncoder{
		EncodingConfig: config,
	}
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
