package storage_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brinestone/dfs/storage"

	"github.com/stretchr/testify/assert"
)

func storeCleanup(root string) {
	_ = os.RemoveAll(root)
}

var logger = log.New(os.Stdout, "[store_test]\t", log.LstdFlags)
var storeConfig = storage.StoreConfig{
	TransformKey: storage.NoopKeyTransformer,
	Logger:       logger,
}

func Test_CASKeyTransformer(t *testing.T) {
	storeConfig.Root = t.TempDir()
	key := "randomkey"
	expectedTransform := "5a8e9/d7284/cd1a2/58466/06e62/c44ce/e4980/2c4fd"
	pathKey := storage.CASKeyTransformer(storeConfig.Root, key)
	assert.Equal(t, expectedTransform, pathKey.Pathname)
}

func Test_NewStore(t *testing.T) {
	store := storage.NewStore(storeConfig)
	assert.NotNil(t, store)
	t.Run("No_Transformer", func(t1 *testing.T) {
		conf := storeConfig
		conf.TransformKey = nil
		store := storage.NewStore(conf)
		assert.NotNil(t1, store)
	})

	t.Run("No_Root", func(t1 *testing.T) {
		conf := storeConfig
		conf.Root = ""
		store := storage.NewStore(conf)
		assert.NotNil(t1, store)
	})
}

func TestStore_Write(t *testing.T) {
	storeConfig.Root = t.TempDir()
	// storeConfig.KeyTransformer = storage.CASKeyTransformer
	datakey := "somekey"
	store := storage.NewStore(storeConfig)
	pathkey := storeConfig.TransformKey(storeConfig.Root, datakey)

	t.Cleanup(func() {
		storeCleanup(pathkey.Root)
	})

	data := bytes.NewReader([]byte("some random data"))

	if err := store.Write(datakey, data); err != nil {
		t.Error(err)
	}
}

func TestStore_Read(t *testing.T) {
	storeConfig.Root = t.TempDir()
	//	storeConfig.KeyTransformer = storage.CASKeyTransformer
	store := storage.NewStore(storeConfig)
	data := []byte("some really random bytes")
	dataSource := bytes.NewReader(data)
	key := "some really good key"
	pathkey := storeConfig.TransformKey(storeConfig.Root, key)
	t.Cleanup(func() {
		storeCleanup(pathkey.Root)
	})

	if err := store.Write(key, dataSource); err != nil {
		t.Error(err)
	}

	r, err := store.Read(key)
	if err != nil {
		t.Error(err)
	}

	readData, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}

	assert.EqualValues(t, data, readData)
}

func TestStore_Delete(t *testing.T) {
	storeConfig.Root = t.TempDir()
	key := "somereallyrandomkey"
	store := storage.NewStore(storeConfig)
	data := []byte("some really important data that needs to be protected at all costs")
	pathKey := store.TransformKey(storeConfig.Root, key)
	t.Cleanup(func() {
		storeCleanup(pathKey.Root)
	})

	dataSource := bytes.NewReader(data)
	if err := store.Write(key, dataSource); err != nil {
		t.Error(err)
	}

	if err := store.Delete(key); err != nil {
		t.Error(err)
	}

	_, err := os.Open(pathKey.FilePath())
	if err != nil {
		if !strings.HasSuffix(err.Error(), "no such file or directory") {
			t.Error(err)
		}
	}
}

func TestStore_Has_when_file_exists(t *testing.T) {
	storeConfig.Root = t.TempDir()
	key := "some really good key"
	data := []byte("some really good amount of bytes")
	store := storage.NewStore(storeConfig)
	pk := storeConfig.TransformKey(storeConfig.Root, key)
	t.Cleanup(func() {
		storeCleanup(pk.Root)
	})

	if err := store.Write(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	assert.True(t, store.Has(key))
}

func TestStore_Has_when_file_not_exists(t *testing.T) {
	key := "some really good key"
	// data := []byte("some really good amount of bytes")
	store := storage.NewStore(storeConfig)
	// pk := storeConfig.TransformKey(key)

	assert.False(t, store.Has(key))
}

func TestStore_Clean(t *testing.T) {
	key := "some really good key"
	data := []byte("Really good data")
	storeConfig.Root = path.Join(t.TempDir(), "drop")
	store := storage.NewStore(storeConfig)

	if err := store.Write(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := store.Clean(); err != nil {
		t.Error(err)
	}

	dirEmpty := true
	err := filepath.WalkDir(store.Root, func(path string, d fs.DirEntry, err error) error {
		dirEmpty = d == nil
		return err
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Error(err)
	}

	assert.True(t, dirEmpty)
}
