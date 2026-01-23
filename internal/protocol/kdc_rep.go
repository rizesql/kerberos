package protocol

import (
	"encoding/json"
	"time"
)

type EncKDCRepPart struct {
	sessionKey SessionKey
	nonce      Nonce
	issuedAt   time.Time
	lifetime   time.Duration
	server     Principal
}

func NewEncKDCRepPart(
	sessionKey SessionKey,
	nonce Nonce,
	issuedAt time.Time,
	lifetime time.Duration,
	server Principal,
) (EncKDCRepPart, error) {
	if nonce == (Nonce{}) {
		return EncKDCRepPart{}, ErrNonceInvalid
	}

	if server == (Principal{}) {
		return EncKDCRepPart{}, ErrInvalidPrincipal
	}

	return EncKDCRepPart{
		sessionKey: sessionKey,
		nonce:      nonce,
		issuedAt:   issuedAt,
		lifetime:   lifetime,
		server:     server,
	}, nil
}

func (e EncKDCRepPart) SessionKey() SessionKey  { return e.sessionKey }
func (e EncKDCRepPart) Nonce() Nonce            { return e.nonce }
func (e EncKDCRepPart) IssuedAt() time.Time     { return e.issuedAt }
func (e EncKDCRepPart) Lifetime() time.Duration { return e.lifetime }
func (e EncKDCRepPart) Server() Principal       { return e.server }

type encKDCRepPart struct {
	SessionKey SessionKey    `json:"session_key"`
	Nonce      Nonce         `json:"nonce"`
	IssuedAt   time.Time     `json:"issued_at"`
	Lifetime   time.Duration `json:"lifetime"`
	Server     Principal     `json:"server"`
}

func (e EncKDCRepPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(encKDCRepPart{
		SessionKey: e.sessionKey,
		Nonce:      e.nonce,
		IssuedAt:   e.issuedAt,
		Lifetime:   e.lifetime,
		Server:     e.server,
	})
}

func (e *EncKDCRepPart) UnmarshalJSON(data []byte) error {
	var tmp encKDCRepPart
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	enc, err := NewEncKDCRepPart(
		tmp.SessionKey,
		tmp.Nonce,
		tmp.IssuedAt,
		tmp.Lifetime,
		tmp.Server,
	)
	if err != nil {
		return err
	}

	*e = enc
	return nil
}
