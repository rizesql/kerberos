package protocol_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestAuthenticatorSerialization(t *testing.T) {
	client, err := protocol.NewPrincipal("testuser", "", "ATHENA.MIT.EDU")
	assert.Err(t, err, nil)

	addr, err := protocol.NewAddress(net.ParseIP("192.168.1.1"))
	assert.Err(t, err, nil)

	now := time.Now().UTC().Truncate(time.Second)

	auth, err := protocol.NewAuthenticator(client, addr, now)
	assert.Err(t, err, nil)

	data, err := json.Marshal(auth)
	assert.Err(t, err, nil)

	var loaded protocol.Authenticator
	err = json.Unmarshal(data, &loaded)
	assert.Err(t, err, nil)

	assert.Equal(t, loaded.Client(), auth.Client())
	assert.Equal(t, loaded.ClientAddr().IP(), auth.ClientAddr().IP())
	assert.True(t, loaded.IssuedAt().Equal(auth.IssuedAt()))
}

func TestAuthenticatorValidation(t *testing.T) {
	client, err := protocol.NewPrincipal("cli", "", "REALM")
	assert.Err(t, err, nil)

	addr, err := protocol.NewAddress(net.IPv4(1, 2, 3, 4))
	assert.Err(t, err, nil)

	now := time.Now()

	_, err = protocol.NewAuthenticator(protocol.Principal{}, addr, now)
	assert.Err(t, err, protocol.ErrAuthenticatorInvalidClient)

	_, err = protocol.NewAuthenticator(client, protocol.Address{}, now)
	assert.Err(t, err, protocol.ErrAuthenticatorInvalidAddress)
}
