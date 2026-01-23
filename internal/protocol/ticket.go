package protocol

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrTicketInvalidServer     = errors.New("ticket server principal cannot be empty")
	ErrTicketInvalidClient     = errors.New("ticket client principal cannot be empty")
	ErrTicketInvalidAddress    = errors.New("ticket client address cannot be empty")
	ErrTicketInvalidSessionKey = errors.New("ticket session key cannot be empty")
)

type Ticket struct {
	server     Principal
	client     Principal
	clientAddr Address
	issuedAt   time.Time
	lifetime   time.Duration
	sessionKey SessionKey
}

func NewTicket(
	server Principal,
	client Principal,
	clientAddr Address,
	issuedAt time.Time,
	lifetime time.Duration,
	sessionKey SessionKey,
) (Ticket, error) {
	if server == (Principal{}) {
		return Ticket{}, ErrTicketInvalidServer
	}
	if client == (Principal{}) {
		return Ticket{}, ErrTicketInvalidClient
	}
	if clientAddr.IsZero() {
		return Ticket{}, ErrTicketInvalidAddress
	}
	if sessionKey.IsZero() {
		return Ticket{}, ErrTicketInvalidSessionKey
	}

	return Ticket{
		server:     server,
		client:     client,
		clientAddr: clientAddr,
		issuedAt:   issuedAt,
		lifetime:   lifetime,
		sessionKey: sessionKey,
	}, nil
}

func (t Ticket) Server() Principal       { return t.server }
func (t Ticket) Client() Principal       { return t.client }
func (t Ticket) ClientAddr() Address     { return t.clientAddr }
func (t Ticket) IssuedAt() time.Time     { return t.issuedAt }
func (t Ticket) Lifetime() time.Duration { return t.lifetime }
func (t Ticket) SessionKey() SessionKey  { return t.sessionKey }

func (t Ticket) IsExpired(now time.Time) bool {
	expiry := t.issuedAt.Add(t.lifetime)
	return now.After(expiry)
}

type ticket struct {
	Server     Principal     `json:"server"`
	Client     Principal     `json:"client"`
	ClientAddr Address       `json:"client_addr"`
	IssuedAt   time.Time     `json:"issued_at"`
	Lifetime   time.Duration `json:"lifetime"`
	SessionKey SessionKey    `json:"session_key"`
}

func (t Ticket) MarshalJSON() ([]byte, error) {
	return json.Marshal(&ticket{
		Server:     t.server,
		Client:     t.client,
		ClientAddr: t.clientAddr,
		IssuedAt:   t.issuedAt,
		Lifetime:   t.lifetime,
		SessionKey: t.sessionKey,
	})
}

func (t *Ticket) UnmarshalJSON(data []byte) error {
	var tmp ticket
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	ti, err := NewTicket(
		tmp.Server,
		tmp.Client,
		tmp.ClientAddr,
		tmp.IssuedAt,
		tmp.Lifetime,
		tmp.SessionKey,
	)
	if err != nil {
		return err
	}

	*t = ti
	return nil
}
