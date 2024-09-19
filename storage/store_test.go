package storage_test

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/brinestone/dfs/storage"

	"github.com/stretchr/testify/assert"
)

var logger = log.New(os.Stdout, "store_test\t", log.LstdFlags)
var storeConfig = storage.StoreConfig{
	KeyTransformer: storage.NoopKeyTransformer,
	Logger:         logger,
}

func TestNewStore(t *testing.T) {
	store := storage.NewStore(storeConfig)
	assert.NotNil(t, store)
}

func TestWrite(t *testing.T) {

	store := storage.NewStore(storeConfig)

	datakey := "somekey"
	defer os.RemoveAll(datakey)

	data := bytes.NewReader([]byte("some random data"))

	if err := store.Write(datakey, data); err != nil {
		t.Error(err)
	}
}

func TestCASKeyTransformer(t *testing.T) {
	key := "randomkey"
	expectedTransform := "5a8e9/d7284/cd1a2/58466/06e62/c44ce/e4980/2c4fd"
	pathname := storage.CASKeyTransformer(key)
	assert.Equal(t, expectedTransform, pathname)
}
