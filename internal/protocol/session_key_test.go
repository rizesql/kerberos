package protocol_test

import (
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestSessionKeyValidation(t *testing.T) {
	_, err := protocol.NewSessionKey(nil)
	assert.Err(t, err, protocol.ErrSessionKeyInvalid)

	_, err = protocol.NewSessionKey([]byte{})
	assert.Err(t, err, protocol.ErrSessionKeyInvalid)
}
