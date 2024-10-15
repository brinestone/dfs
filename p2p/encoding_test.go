package p2p_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/brinestone/dfs/p2p"
	"github.com/stretchr/testify/assert"
)

var encodingConfig = p2p.EncodingConfig{
	BufferSize: 5049,
}

func TestDecode(t *testing.T) {
	data := make([]byte, encodingConfig.BufferSize)
	_, err := io.ReadFull(rand.Reader, data)
	if err != nil {
		t.Error(err)
	}

	decoder := p2p.DefaultDecoder{
		EncodingConfig: encodingConfig,
	}

	assert.Equal(t, encodingConfig.BufferSize, decoder.BufferSize)

	rpc := p2p.Rpc{}

	if err := decoder.Decode(bytes.NewReader(data), &rpc); err != nil {
		t.Error(err)
	}

	assert.ObjectsAreEqualValues(data, rpc.Payload)
}
