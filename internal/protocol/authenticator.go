package protocol

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrAuthenticatorInvalidClient  = errors.New("authenticator client principal cannot be empty")
	ErrAuthenticatorInvalidAddress = errors.New("authenticator client address cannot be empty")
)

type Authenticator struct {
	client     Principal
	clientAddr Address
	issuedAt   time.Time
}

func NewAuthenticator(
	client Principal,
	clientAddr Address,
	issuedAt time.Time,
) (Authenticator, error) {
	if client == (Principal{}) {
		return Authenticator{}, ErrAuthenticatorInvalidClient
	}
	if clientAddr.IsZero() {
		return Authenticator{}, ErrAuthenticatorInvalidAddress
	}

	return Authenticator{
		client:     client,
		clientAddr: clientAddr,
		issuedAt:   issuedAt,
	}, nil
}

func (a Authenticator) Client() Principal   { return a.client }
func (a Authenticator) ClientAddr() Address { return a.clientAddr }
func (a Authenticator) IssuedAt() time.Time { return a.issuedAt }

type authenticator struct {
	Client     Principal `json:"client"`
	ClientAddr Address   `json:"client_addr"`
	IssuedAt   time.Time `json:"issued_at"`
}

func (a Authenticator) MarshalJSON() ([]byte, error) {
	return json.Marshal(authenticator{
		Client:     a.client,
		ClientAddr: a.clientAddr,
		IssuedAt:   a.issuedAt,
	})
}

func (a *Authenticator) UnmarshalJSON(data []byte) error {
	var tmp authenticator
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	au, err := NewAuthenticator(
		tmp.Client,
		tmp.ClientAddr,
		tmp.IssuedAt,
	)
	if err != nil {
		return err
	}

	*a = au
	return nil
}
