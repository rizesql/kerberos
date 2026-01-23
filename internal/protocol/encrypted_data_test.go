package protocol_test

import (
	"encoding/json"
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestEncryptedData(t *testing.T) {
	t.Run("NewEncryptedData", func(t *testing.T) {
		_, err := protocol.NewEncryptedData([]byte("ciphertext"))
		assert.Err(t, err, nil)

		_, err = protocol.NewEncryptedData(nil)
		assert.Err(t, err, protocol.ErrInvalidCiphertext)

		_, err = protocol.NewEncryptedData([]byte{})
		assert.Err(t, err, protocol.ErrInvalidCiphertext)
	})

	t.Run("JSON Roundtrip", func(t *testing.T) {
		original, err := protocol.NewEncryptedData([]byte("secret"))
		assert.Err(t, err, nil)

		data, err := json.Marshal(original)
		assert.Err(t, err, nil)

		var decoded protocol.EncryptedData
		err = json.Unmarshal(data, &decoded)
		assert.Err(t, err, nil)

		assert.Equal(t, string(decoded.Ciphertext()), string(original.Ciphertext()))
	})
}
