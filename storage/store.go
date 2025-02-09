package storage

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"path"
	"strings"
	"sync"
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
	storeMuMu sync.Mutex
	storeMu   map[string]*sync.Mutex
}

type DataStat struct {
	Total       uint64
	PieceSize   uint64
	ChecksumMap map[uint64][16]byte
}

func (s *Store) GetMissingDeltas(key string, stat *DataStat) ([]uint64, error) {
	localStat, err := s.DataStatFor(key, UsingPieceSize(stat.PieceSize))
	ans := make([]uint64, 0)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	for k, v := range stat.ChecksumMap {
		if localStat != nil {
			if sum, ok := localStat.ChecksumMap[k]; ok && v == sum {
				continue
			}
		}

		ans = append(ans, k)
	}

	return ans, nil
}

const MaxPieceSize float64 = 8 * 1024 * 1024

func PieceSize(fs int64) int64 {
	return int64(math.Min(float64(fs/10), MaxPieceSize))
}

type StoreOpOption struct {
	name  string
	Value any
}

const (
	opPieceSize = "piece-size"
	opOffset    = "offset"
)

func UsingPieceSize(size uint64) StoreOpOption {
	return StoreOpOption{
		name:  opPieceSize,
		Value: int64(size),
	}
}

func UsingComputedPieceSize() StoreOpOption {
	return StoreOpOption{
		name:  opPieceSize,
		Value: int64(-1),
	}
}

func UsingOffset(offset uint64) StoreOpOption {
	return StoreOpOption{
		name:  opOffset,
		Value: int64(offset),
	}
}

func (s *Store) DataStatFor(key string, options ...StoreOpOption) (*DataStat, error) {
	opts := readStoreOptions(options...)
	var pieceSize uint64 = uint64(opts[opPieceSize].(int64))
	var ans *DataStat

	pk := s.TransformKey(key)

	filePath := pk.FilePath()
	handle, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	stat, err := handle.Stat()
	if err != nil {
		return nil, err
	}

	if pieceSize == 0 {
		pieceSize = uint64(PieceSize(stat.Size()))
	}

	var offset int
	var checksumMap = make(map[uint64][16]byte)
	for {
		buf := make([]byte, pieceSize)
		n, err := handle.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		checksumMap[uint64(offset)] = md5.Sum(buf[:n])
		offset += n
	}

	ans = &DataStat{
		Total:       uint64(stat.Size()),
		PieceSize:   pieceSize,
		ChecksumMap: checksumMap,
	}

	return ans, nil
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
		storeMu:     make(map[string]*sync.Mutex),
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

func (s *Store) Write(key string, r io.Reader, options ...StoreOpOption) (int64, error) {
	opts := readStoreOptions(options...)
	var offset int64 = opts[opOffset].(int64)

	s.storeMuMu.Lock()
	defer s.storeMuMu.Unlock()
	var mu *sync.Mutex
	var ok bool
	if mu, ok = s.storeMu[key]; !ok {
		mu = &sync.Mutex{}
	}

	mu.Lock()
	defer mu.Unlock()
	defer delete(s.storeMu, key)
	return s.writeStream(offset, key, r)
}

func (s *Store) getSize(key string) (int64, error) {
	path := s.TransformKey(key).FilePath()
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

func (s *Store) Read(key string, options ...StoreOpOption) (io.Reader, error) {
	opts := readStoreOptions(options...)
	offset, ok := opts[opOffset].(int64)
	if !ok {
		offset = 0
	}

	readSize, ok := opts[opPieceSize].(int64)
	if !ok || readSize < 0 {
		total, err := s.getSize(key)
		if err != nil {
			return nil, err
		}
		readSize = PieceSize(total)
	}

	handle, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = handle.Close()
	}()

	if _, err := handle.Seek(offset, io.SeekCurrent); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if _, err := io.CopyN(buf, handle, readSize); err != nil {
		return nil, err
	}

	return buf, nil
}

func (s *Store) readStream(key string) (io.ReadSeekCloser, error) {
	pathKey := s.TransformKey(key)

	return os.Open(pathKey.FilePath())
}

func (s *Store) writeStream(offset int64, key string, r io.Reader) (int64, error) {
	pathKey := s.TransformKey(key)

	if err := os.MkdirAll(pathKey.Pathname, os.ModePerm); err != nil {
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

	n, err := io.Copy(handle, r)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func readStoreOptions(options ...StoreOpOption) map[string]any {
	ans := make(map[string]any)
	for _, v := range options {
		ans[v.name] = v.Value
	}

	return ans
}
