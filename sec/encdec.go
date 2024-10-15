// The enc package contains all functions and types concerning encryption/decryption operations.
package sec

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
)

type KeyPair struct {
	Private []byte
	Public  []byte
}

// Generates asymmetric key-pair
func NewKeyPair() (*KeyPair, error) {
	pvk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	pvkB := x509.MarshalPKCS1PrivateKey(pvk)
	pvkPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pvkB,
	})

	pbk := &pvk.PublicKey
	pkB := x509.MarshalPKCS1PublicKey(pbk)
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
func Enc(keypath string, plain []byte) ([]byte, error) {
	ciphertext := make([]byte, 0)

	pkPem, err := os.ReadFile(keypath)
	if err != nil {
		return ciphertext, err
	}

	keyblock, _ := pem.Decode(pkPem)
	pk, err := x509.ParsePKIXPublicKey(keyblock.Bytes)
	if err != nil {
		return ciphertext, err
	}

	ciphertext, err = rsa.EncryptPKCS1v15(rand.Reader, pk.(*rsa.PublicKey), plain)
	if err != nil {
		return make([]byte, 0), err
	}

	return ciphertext, nil
}
