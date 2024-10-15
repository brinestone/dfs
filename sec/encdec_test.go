package sec_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"path"
	"testing"

	"github.com/brinestone/dfs/sec"
	"github.com/stretchr/testify/assert"
)

func Test_NewKeyPair(t *testing.T) {
	kp, err := sec.NewKeyPair()
	if err != nil {
		t.Error(err)
	}
	assert.NotNil(t, kp)
}

func createTempKeyPair(t *testing.T) (string, string) {
	kp, err := sec.NewKeyPair()
	if err != nil {
		t.Error(err)
	}
	pvPath := path.Join(t.TempDir(), "priv_key")
	pbPath := path.Join(t.TempDir(), "pub.pub")

	h1, err := os.Create(pvPath)
	if err != nil {
		t.Error(err)
	}
	defer h1.Close()

	h2, err := os.Create(pbPath)
	if err != nil {
		t.Error(err)
	}
	defer h2.Close()

	_, err = io.Copy(h1, bytes.NewReader(kp.Private))
	if err != nil {
		t.Error(err)
	}

	_, err = io.Copy(h2, bytes.NewReader(kp.Public))
	if err != nil {
		t.Error(err)
	}

	return pvPath, pbPath
}

func TestEnc(t *testing.T) {
	_, public := createTempKeyPair(t)
	data := make([]byte, 128)
	_, err := io.ReadFull(rand.Reader, data)
	if err != nil {
		t.Error(err)
	}

	transform, err := sec.Enc(public, data)
	if err != nil {
		t.Error(err)
	}

	assert.NotEqualValues(t, data, transform)
}
