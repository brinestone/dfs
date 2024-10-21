// The encoding package contains all functions and types concerning encoding/decoding operations.
package encoding

import (
	"io"

	"github.com/brinestone/dfs/crypt"
)

type Encoder interface {
	Encode([]byte) ([]byte, error)
	EncodeStream(io.Reader, io.Writer) error
}

type Decoder interface {
	Decode([]byte) ([]byte, error)
	DecodeStream(io.Reader, io.Writer) error
}

type EncodingConfig struct {
	BuffSize int64
}

type SecureEncoderDecoder struct {
	EncodingConfig
	secret []byte
}

type PlainEncoderDecoder struct {
	EncodingConfig
}

func (p *PlainEncoderDecoder) DecodeStream(in io.Reader, out io.Writer) error {
	_, err := io.Copy(out, in)
	return err
}

func (p *PlainEncoderDecoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}

func (p *PlainEncoderDecoder) EncodeStream(in io.Reader, out io.Writer) error {
	_, err := io.Copy(out, in)
	return err
}

func (p *PlainEncoderDecoder) Encode(data []byte) ([]byte, error) {
	return data, nil
}

func NewPlainEncoderDecoder(cnf EncodingConfig) *PlainEncoderDecoder {
	return &PlainEncoderDecoder{
		EncodingConfig: cnf,
	}
}

func (s *SecureEncoderDecoder) Decode(data []byte) ([]byte, error) {
	return crypt.Dec(s.secret, data)
}

func (s *SecureEncoderDecoder) DecodeStream(in io.Reader, out io.Writer) error {
	return crypt.DecStream(s.secret, int(s.BuffSize), in, out)
}

func (s *SecureEncoderDecoder) EncodeStream(in io.Reader, out io.Writer) error {
	return crypt.EncStream(s.secret, int(s.BuffSize), in, out)
}

func (s *SecureEncoderDecoder) Encode(data []byte) ([]byte, error) {
	return crypt.Enc(s.secret, data)
}

func NewSecureEncoderDecoder(secret []byte, cnf EncodingConfig) *SecureEncoderDecoder {
	return &SecureEncoderDecoder{
		EncodingConfig: cnf,
		secret:         secret,
	}
}
