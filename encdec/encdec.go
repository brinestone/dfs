// The enc package contains all functions and types concerning encryption/decryption operations.
package encdec

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"os"
)

const keySize = 4096

type KeyPair struct {
	Private []byte
	Public  []byte
}

var ErrInvalidKeyPair = errors.New("key pair is not valid")

// Generates asymmetric key-pair
func NewKeyPair() (*KeyPair, error) {
	pvk, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, err
	}

	pvkB := x509.MarshalPKCS1PrivateKey(pvk)
	pvkPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pvkB,
	})

	pbk := &pvk.PublicKey
	pkB, err := x509.MarshalPKIXPublicKey(pbk)
	if err != nil {
		return nil, err
	}

	pkPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pkB,
	})

	return &KeyPair{
		Private: pvkPem,
		Public:  pkPem,
	}, nil
}

// Encrypts the data using the specified public key
func Encrypt(keypath string, plaintext []byte) ([]byte, error) {
	var ciphertext []byte

	pkPem, err := os.ReadFile(keypath)
	if err != nil {
		return ciphertext, err
	}

	keyblock, _ := pem.Decode(pkPem)
	pk, err := x509.ParsePKIXPublicKey(keyblock.Bytes)
	if err != nil {
		return ciphertext, err
	}

	resBuf := new(bytes.Buffer)
	src := bytes.NewReader(plaintext)

	for {
		buf := make([]byte, keySize/16)
		n, err := src.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return make([]byte, 0), err
		}

		res, err := rsa.EncryptPKCS1v15(rand.Reader, pk.(*rsa.PublicKey), buf[:n])
		if err != nil {
			return make([]byte, 0), err
		}
		resBuf.Write(res)
	}
	ciphertext = resBuf.Bytes()

	return ciphertext, nil
}

// Decrypts the ciphertext back to plaintext
func Decrypt(keypath string, ciphertext []byte) ([]byte, error) {
	var plain []byte
	var err error

	pvPem, err := os.ReadFile(keypath)
	if err != nil {
		return plain, err
	}

	keyblock, _ := pem.Decode(pvPem)
	pvk, err := x509.ParsePKCS1PrivateKey(keyblock.Bytes)
	if err != nil {
		return plain, err
	}

	resBuf := new(bytes.Buffer)
	src := bytes.NewReader(ciphertext)

	for {
		buf := make([]byte, keySize/8)
		n, err := src.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return make([]byte, 0), err
		}

		plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, pvk, buf[:n])
		if err != nil {
			return []byte{}, err
		}
		resBuf.Write(plaintext)
	}
	plain = resBuf.Bytes()

	return plain, err
}

func EncryptedReader(keyPath string, in io.Reader) (io.Reader, error) {
	var ans io.Reader

	pkPem, err := os.ReadFile(keyPath)
	if err != nil {
		return ans, err
	}

	keyblock, _ := pem.Decode(pkPem)
	pk, err := x509.ParsePKIXPublicKey(keyblock.Bytes)
	if err != nil {
		return ans, err
	}

	out := new(bytes.Buffer)

	for {
		buf := make([]byte, keySize/16)
		n, err := in.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return ans, err
		}

		b, err := rsa.EncryptPKCS1v15(rand.Reader, pk.(*rsa.PublicKey), buf[:n])
		if err != nil {
			return ans, err
		}

		out.Write(b)
	}

	ans = out

	return ans, nil
}
