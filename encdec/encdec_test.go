package encdec_test

import (
	"bytes"
	"io"
	"os"
	"path"
	"testing"

	"github.com/brinestone/dfs/encdec"
	"github.com/stretchr/testify/assert"
)

func Test_NewKeyPair(t *testing.T) {
	kp, err := encdec.NewKeyPair()
	if err != nil {
		t.Error(err)
	}
	assert.NotNil(t, kp)
}

func createTempKeyPair(t *testing.T) (string, string) {
	kp, err := encdec.NewKeyPair()
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
	private, public := createTempKeyPair(t)
	data := []byte("some really random data")
	var (
		ciphertext = new(bytes.Buffer)
		plaintext  = new(bytes.Buffer)
	)

	t.Run("encryption", func(t *testing.T) {
		testEncrypt(t, data, ciphertext, public)
	})

	t.Run("decryption", func(t *testing.T) {
		testDecrypt(t, plaintext, ciphertext.Bytes(), private)
	})

	assert.EqualValues(t, data, plaintext.Bytes())
}

func testEncrypt(t *testing.T, plaintext []byte, ciphertext io.Writer, pk string) {
	var err error
	res, err := encdec.Encrypt(pk, plaintext)
	if err != nil {
		t.Error(err)
	}
	assert.NotEqualValues(t, plaintext, res)
	ciphertext.Write(res)
}

func testDecrypt(t *testing.T, plaintext io.Writer, ciphertext []byte, pk string) {
	var err error
	res, err := encdec.Decrypt(pk, ciphertext)
	if err != nil {
		t.Error(err)
	}
	assert.NotEqualValues(t, ciphertext, res)
	plaintext.Write(res)
}
