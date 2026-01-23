package crypto_test

import (
	"encoding/hex"
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/crypto"
)

func TestDeriveKey(t *testing.T) {
	password := "secret"
	salt := "ATHENA.MIT.EDUtestuser"

	key1, err := crypto.DeriveKey(password, salt)
	assert.Err(t, err, nil)

	assert.Equal(t, len(key1.Expose()), 32)

	key2, err := crypto.DeriveKey(password, salt)
	assert.Err(t, err, nil)

	assert.Equal(t, key1.Expose(), key2.Expose())

	key3, err := crypto.DeriveKey(password, "differentsalt")
	assert.Err(t, err, nil)

	assert.True(t, hex.EncodeToString(key1.Expose()) != hex.EncodeToString(key3.Expose()))
}

func TestDeriveKey_Options(t *testing.T) {
	password := "secret"
	salt := "salt"

	key, err := crypto.DeriveKey(password, salt, crypto.WithKeySize(16))
	assert.Err(t, err, nil)

	assert.Equal(t, len(key.Expose()), 16)
}
