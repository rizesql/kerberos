package crypto_test

import (
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestEncryptDecrypt(t *testing.T) {
	keyBytes := make([]byte, 32)
	key, err := protocol.NewSessionKey(keyBytes)
	assert.Err(t, err, nil)

	plaintext := []byte("secret message")

	ciphertext, err := crypto.Encrypt(key, plaintext)
	assert.Err(t, err, nil)

	assert.True(t, string(ciphertext) != string(plaintext))

	decrypted, err := crypto.Decrypt(key, ciphertext)
	assert.Err(t, err, nil)

	assert.Equal(t, decrypted, plaintext)
}

func TestIntegrityCheck(t *testing.T) {
	keyBytes := make([]byte, 32)
	key, err := protocol.NewSessionKey(keyBytes)
	assert.Err(t, err, nil)

	plaintext := []byte("integrity check")
	ciphertext, err := crypto.Encrypt(key, plaintext)
	assert.Err(t, err, nil)

	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = crypto.Decrypt(key, ciphertext)
	assert.Err(t, err, crypto.ErrAuthFailed)
}

func TestInvalidKeySize(t *testing.T) {
	badKeyBytes := make([]byte, 10)
	key, err := protocol.NewSessionKey(badKeyBytes)
	assert.Err(t, err, nil)

	_, err = crypto.Encrypt(key, []byte("data"))
	assert.Err(t, err, crypto.ErrInvalidKey)
}
