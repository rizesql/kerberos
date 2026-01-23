package protocol_test

import (
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestAddressValidation(t *testing.T) {
	_, err := protocol.NewAddress(nil)
	assert.Err(t, err, protocol.ErrAddressInvalid)

	_, err = protocol.NewAddress([]byte{})
	assert.Err(t, err, protocol.ErrAddressInvalid)
}
