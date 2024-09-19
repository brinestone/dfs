package storage

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path"
)

type KeyTransformer func(string) string

func NoopKeyTransformer(k string) string { return k }

func CASKeyTransformer(k string) string {
	digest := sha1.Sum([]byte(k))
	digestStr := hex.EncodeToString(digest[:])

	blockSize := 5
	sliceLen := len(digestStr) / blockSize

	segments := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		segments[i] = digestStr[from:to]
	}

	return path.Join(segments...)
}

type StoreConfig struct {
	KeyTransformer
	Logger *log.Logger
}

type Store struct {
	StoreConfig
}

func NewStore(config StoreConfig) *Store {
	return &Store{
		StoreConfig: config,
	}
}

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathName := key

	if err := os.MkdirAll(pathName, os.ModePerm); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	io.Copy(buf, r)

	fileNameBytes := md5.Sum(buf.Bytes())
	fileName := hex.EncodeToString(fileNameBytes[:])
	filePath := path.Join(pathName, fileName)

	handle, err := os.Create(filePath)
	if err != nil {
		return err
	}

	n, err := io.Copy(handle, buf)
	if err != nil {
		return err
	}

	s.Logger.Printf("written [%d] bytes to storage", n)

	return nil
}
