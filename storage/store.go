package storage

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"
)

type KeyPath struct {
	Pathname string
	Root     string
	Filename string
}

func (p KeyPath) FilePath() string {
	return path.Join(p.Pathname, p.Filename)
}

type KeyTransformer func(string) KeyPath

func NoopKeyTransformer(root string) KeyTransformer {
	return func(k string) KeyPath {
		return KeyPath{
			Pathname: k,
			Root:     root,
			Filename: strings.ReplaceAll(k, " \t", "_"),
		}
	}
}

func CASKeyTransformer(root string) KeyTransformer {
	return func(k string) KeyPath {
		digest := sha1.Sum([]byte(k))
		digestStr := hex.EncodeToString(digest[:])

		blockSize := 5
		sliceLen := len(digestStr) / blockSize

		segments := make([]string, sliceLen)

		for i := 0; i < sliceLen; i++ {
			from, to := i*blockSize, (i*blockSize)+blockSize
			segments[i] = digestStr[from:to]
		}

		pathName := []string{root}
		pathName = append(pathName, segments...)

		return KeyPath{
			Filename: digestStr,
			Pathname: path.Join(pathName...),
			Root:     root,
		}
	}
}

type StoreConfig struct {
	TransformKey KeyTransformer
	Logger       *slog.Logger
	Root         string
}

type Store struct {
	StoreConfig
}

func NewStore(config StoreConfig) *Store {
	if config.TransformKey == nil {
		config.TransformKey = NoopKeyTransformer(config.Root)
	}

	if len(config.Root) == 0 {
		homeDir, err := os.UserCacheDir()
		if err != nil {
			homeDir = ""
		}
		config.Root = homeDir
	}

	return &Store{
		StoreConfig: config,
	}
}

func (s *Store) Clean() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Has(key string) bool {
	pk := s.TransformKey(key)
	_, err := os.Stat(pk.FilePath())
	return !errors.Is(err, fs.ErrNotExist)
}

func (s *Store) Delete(key string) error {
	pathKey := s.TransformKey(key)
	return os.RemoveAll(pathKey.Root)
}

func (s *Store) Write(offset int64, key string, r io.Reader) (int64, error) {
	return s.writeStream(offset, key, r)
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
	pathKey := s.TransformKey(key)
	filePath := pathKey.FilePath()

	return os.Open(filePath)
}

func (s *Store) writeStream(offset int64, key string, r io.Reader) (int64, error) {
	pathKey := s.TransformKey(key)

	if err := os.MkdirAll(pathKey.Pathname, os.ModePerm); err != nil {
		return 0, err
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return 0, err
	}

	filePath := pathKey.FilePath()

	var handle *os.File
	if offset == 0 {
		var err error
		handle, err = os.Create(filePath)
		if err != nil {
			return 0, err
		}
	} else {
		var err error
		handle, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return 0, err
		}
		pos, err := handle.Seek(offset, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		if pos != offset {
			return 0, errors.New("data corrupt")
		}
	}
	defer handle.Close()

	n, err := io.Copy(handle, buf)
	if err != nil {
		return 0, err
	}

	return n, nil
}
