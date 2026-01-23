package protocol_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestEncKDCRepPart(t *testing.T) {
	sessionKey, _ := protocol.NewSessionKey(make([]byte, 32))
	server, _ := protocol.NewPrincipal("server", "inst", "REALM")
	now := time.Now().UTC().Truncate(time.Second) // Truncate for JSON precision issues if any
	lifetime := 10 * time.Hour

	t.Run("NewEncKDCRepPart", func(t *testing.T) {
		nonce, _ := protocol.NewNonce(123)
		_, err := protocol.NewEncKDCRepPart(sessionKey, nonce, now, lifetime, server)
		assert.Err(t, err, nil)

		_, err = protocol.NewEncKDCRepPart(sessionKey, protocol.Nonce{}, now, lifetime, server)
		assert.Err(t, err, protocol.ErrNonceInvalid)

		_, err = protocol.NewEncKDCRepPart(sessionKey, nonce, now, lifetime, protocol.Principal{})
		assert.Err(t, err, protocol.ErrInvalidPrincipal)
	})

	t.Run("JSON Roundtrip", func(t *testing.T) {
		nonce, _ := protocol.NewNonce(456)
		original, _ := protocol.NewEncKDCRepPart(sessionKey, nonce, now, lifetime, server)

		data, err := json.Marshal(original)
		assert.Err(t, err, nil)

		var decoded protocol.EncKDCRepPart
		err = json.Unmarshal(data, &decoded)
		assert.Err(t, err, nil)

		assert.Equal(t, decoded.Nonce(), original.Nonce())
		assert.Equal(t, decoded.Server(), original.Server())
		assert.Equal(t, decoded.Lifetime(), original.Lifetime())
		assert.Equal(t, decoded.IssuedAt().Unix(), original.IssuedAt().Unix())
	})
}
