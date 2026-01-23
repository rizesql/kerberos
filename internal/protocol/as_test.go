package protocol_test

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestASReq(t *testing.T) {
	client, _ := protocol.NewPrincipal("client", "", "ATHENA.MIT.EDU")
	service, _ := protocol.NewPrincipal("krbtgt", "", "ATHENA.MIT.EDU")
	addr, _ := protocol.NewAddress(net.IPv4(127, 0, 0, 1))

	t.Run("NewASReq", func(t *testing.T) {
		nonce, _ := protocol.NewNonce(12345)
		_, err := protocol.NewASReq(client, service, addr, nonce)
		assert.Err(t, err, nil)

		_, err = protocol.NewASReq(protocol.Principal{}, service, addr, nonce)
		assert.Err(t, err, protocol.ErrInvalidPrincipal)

		_, err = protocol.NewASReq(client, service, addr, protocol.Nonce{})
		assert.Err(t, err, protocol.ErrNonceInvalid)
	})

	t.Run("JSON Roundtrip", func(t *testing.T) {
		nonce, _ := protocol.NewNonce(999)
		original, err := protocol.NewASReq(client, service, addr, nonce)
		assert.Err(t, err, nil)

		data, err := json.Marshal(original)
		assert.Err(t, err, nil)

		var decoded protocol.ASReq
		err = json.Unmarshal(data, &decoded)
		assert.Err(t, err, nil)

		assert.Equal(t, decoded.Client(), original.Client())
		assert.Equal(t, decoded.Service(), original.Service())
		assert.Equal(t, decoded.Nonce(), original.Nonce())
		assert.Equal(t, decoded.ClientAddr().IP().String(), original.ClientAddr().IP().String())
	})
}

func TestASRep(t *testing.T) {
	ticket, _ := protocol.NewEncryptedData([]byte("ticket"))
	secretPart, _ := protocol.NewEncryptedData([]byte("secret"))

	t.Run("NewASRep", func(t *testing.T) {
		_, err := protocol.NewASRep(ticket, secretPart)
		assert.Err(t, err, nil)
	})

	t.Run("JSON Roundtrip", func(t *testing.T) {
		original, _ := protocol.NewASRep(ticket, secretPart)

		data, err := json.Marshal(original)
		assert.Err(t, err, nil)

		var decoded protocol.ASRep
		err = json.Unmarshal(data, &decoded)
		assert.Err(t, err, nil)

		assert.Equal(t, string(decoded.Ticket().Ciphertext()), string(original.Ticket().Ciphertext()))
		assert.Equal(t, string(decoded.SecretPart().Ciphertext()), string(original.SecretPart().Ciphertext()))
	})
}
