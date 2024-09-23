package storage

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
)

var defaultLogger *log.Logger = log.New(log.Writer(), "[STORE]\t", log.LstdFlags)

type PathKey struct {
	Pathname string
	Root     string
	Filename string
}

func (p PathKey) FilePath() string {
	return path.Join(p.Root, p.Pathname, p.Filename)
}

func (p PathKey) FullPath() string {
	return path.Join(p.Root, p.Pathname)
}

type KeyTransformer func(string, string) PathKey

func NoopKeyTransformer(root string, k string) PathKey {
	return PathKey{
		Pathname: k,
		Root:     root,
		Filename: strings.ReplaceAll(k, " \t\n", "_"),
	}
}

func CASKeyTransformer(root, k string) PathKey {
	digest := sha1.Sum([]byte(k))
	digestStr := hex.EncodeToString(digest[:])

	blockSize := 5
	sliceLen := len(digestStr) / blockSize

	segments := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		segments[i] = digestStr[from:to]
	}

	return PathKey{
		Filename: digestStr,
		Pathname: path.Join(segments...),
		Root:     root,
	}
}

type StoreConfig struct {
	TransformKey KeyTransformer
	Logger       *log.Logger
	// Root directory of the store
	Root string
}

type Store struct {
	StoreConfig
}

func NewStore(config StoreConfig) *Store {
	if config.TransformKey == nil {
		config.TransformKey = NoopKeyTransformer
	}

	if len(config.Root) == 0 {
		homeDir, err := os.UserCacheDir()
		if err != nil {
			homeDir = ""
		}
		config.Root = homeDir
	}

	if config.Logger == nil {
		config.Logger = defaultLogger
	}

	return &Store{
		StoreConfig: config,
	}
}

func (s *Store) Clean() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Has(key string) bool {
	pk := s.TransformKey(s.Root, key)
	_, err := os.Stat(pk.FilePath())
	return !errors.Is(err, fs.ErrNotExist)
}

func (s *Store) Delete(key string) error {
	pathKey := s.TransformKey(s.Root, key)
	return os.RemoveAll(pathKey.FilePath())
}

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

func (s *Store) Read(key string) (io.Reader, error) {
	handle, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = handle.Close()
	}()

	buff := new(bytes.Buffer)
	if _, err := io.Copy(buff, handle); err != nil {
		return nil, err
	}

	return buff, nil
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.TransformKey(s.Root, key)
	filePath := pathKey.FilePath()

	return os.Open(filePath)
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.TransformKey(s.Root, key)

	fullPath := pathKey.FullPath()
	if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return err
	}

	filePath := pathKey.FilePath()

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
