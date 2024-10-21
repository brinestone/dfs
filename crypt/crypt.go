package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"
)

// Decrypts a stream
func DecStream(key []byte, cs int, in io.Reader, out io.Writer) error {
	for {
		buf := make([]byte, cs)
		n, err := in.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		pt, err := Dec(key, buf)
		if err != nil {
			return nil
		}

		if _, err := out.Write(pt); err != nil {
			return err
		}
	}
	return nil
}

// Encrypts a stream
func EncStream(key []byte, cs int, in io.Reader, out io.Writer) error {
	for {
		buf := make([]byte, cs)
		n, err := in.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil
		}

		ct, err := Enc(key, buf)
		if err != nil {
			return err
		}

		if _, err = out.Write(ct); err != nil {
			return err
		}
	}
	return nil
}

// Uses a remote public and local keys to generate a secret key
func SecGen(pk *ecdh.PrivateKey, pbk *ecdh.PublicKey) ([]byte, error) {
	s, err := pk.ECDH(pbk)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Generates a new elliptic-curve Diffie-Hellman key-pair
func KeyGen() (*ecdh.PrivateKey, *ecdh.PublicKey, error) {
	curve := ecdh.P256()
	pvk, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	return pvk, pvk.PublicKey(), nil
}

// Encrypts plaintext to ciphertext
func Enc(key, plaintext []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	ans := gcm.Seal(nonce, nonce, plaintext, nil)
	return ans, nil
}

// Decrypts ciphertext into plaintext
func Dec(key, ciphertext []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	ns := gcm.NonceSize()
	return gcm.Open(nil, ciphertext[:ns], ciphertext[ns:], nil)
}
