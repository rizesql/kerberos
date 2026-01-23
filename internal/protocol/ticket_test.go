package protocol_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestTicketSerialization(t *testing.T) {
	server, err := protocol.NewPrincipal("krbtgt", "example.com", "ATHENA.MIT.EDU")
	assert.Err(t, err, nil)

	client, err := protocol.NewPrincipal("testuser", "", "ATHENA.MIT.EDU")
	assert.Err(t, err, nil)

	addr, err := protocol.NewAddress(net.ParseIP("127.0.0.1"))
	assert.Err(t, err, nil)

	key, err := protocol.NewSessionKey(make([]byte, 32))
	assert.Err(t, err, nil)

	now := time.Now().UTC().Truncate(time.Second)

	ticket, err := protocol.NewTicket(
		server,
		client,
		addr,
		now,
		8*time.Hour,
		key,
	)
	assert.Err(t, err, nil)

	data, err := json.Marshal(ticket)
	assert.Err(t, err, nil)

	var loaded protocol.Ticket
	err = json.Unmarshal(data, &loaded)
	assert.Err(t, err, nil)

	assert.Equal(t, loaded.Server(), ticket.Server())
	assert.Equal(t, loaded.Client(), ticket.Client())
	assert.Equal(t, loaded.ClientAddr().IP(), ticket.ClientAddr().IP())
	assert.True(t, loaded.IssuedAt().Equal(ticket.IssuedAt()))
	assert.Equal(t, loaded.Lifetime(), ticket.Lifetime())
	assert.Equal(t, loaded.SessionKey().Expose(), ticket.SessionKey().Expose())
}

func TestTicketValidation(t *testing.T) {
	server, err := protocol.NewPrincipal("srv", "", "REALM")
	assert.Err(t, err, nil)

	client, err := protocol.NewPrincipal("cli", "", "REALM")
	assert.Err(t, err, nil)

	addr, err := protocol.NewAddress(net.IPv4(1, 2, 3, 4))
	assert.Err(t, err, nil)

	key, err := protocol.NewSessionKey(make([]byte, 16))
	assert.Err(t, err, nil)

	now := time.Now()
	life := time.Hour

	_, err = protocol.NewTicket(protocol.Principal{}, client, addr, now, life, key)
	assert.Err(t, err, protocol.ErrTicketInvalidServer)

	_, err = protocol.NewTicket(server, protocol.Principal{}, addr, now, life, key)
	assert.Err(t, err, protocol.ErrTicketInvalidClient)

	_, err = protocol.NewTicket(server, client, protocol.Address{}, now, life, key)
	assert.Err(t, err, protocol.ErrTicketInvalidAddress)

	_, err = protocol.NewTicket(server, client, addr, now, life, protocol.SessionKey{})
	assert.Err(t, err, protocol.ErrTicketInvalidSessionKey)
}
