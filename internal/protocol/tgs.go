package protocol

import (
	"encoding/json"
	"errors"
)

var (
	ErrInvalidPrincipal = errors.New("principal cannot be empty")
)

type TGSReq struct {
	server        Principal
	tgt           EncryptedData
	authenticator EncryptedData
	nonce         Nonce
}

func NewTGSReq(
	server Principal,
	tgt EncryptedData,
	authenticator EncryptedData,
	nonce Nonce,
) (TGSReq, error) {
	if server == (Principal{}) {
		return TGSReq{}, ErrInvalidPrincipal
	}

	if nonce == (Nonce{}) {
		return TGSReq{}, ErrNonceInvalid
	}

	return TGSReq{
		server:        server,
		tgt:           tgt,
		authenticator: authenticator,
		nonce:         nonce,
	}, nil
}

func (r TGSReq) Server() Principal            { return r.server }
func (r TGSReq) TGT() EncryptedData           { return r.tgt }
func (r TGSReq) Authenticator() EncryptedData { return r.authenticator }
func (r TGSReq) Nonce() Nonce                 { return r.nonce }

type tgsReq struct {
	Server        Principal     `json:"server"`
	TGT           EncryptedData `json:"tgt"`
	Authenticator EncryptedData `json:"authenticator"`
	Nonce         Nonce         `json:"nonce"`
}

func (r TGSReq) MarshalJSON() ([]byte, error) {
	return json.Marshal(tgsReq{
		Server:        r.server,
		TGT:           r.tgt,
		Authenticator: r.authenticator,
		Nonce:         r.nonce,
	})
}

func (r *TGSReq) UnmarshalJSON(data []byte) error {
	var tmp tgsReq
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	req, err := NewTGSReq(tmp.Server, tmp.TGT, tmp.Authenticator, tmp.Nonce)
	if err != nil {
		return err
	}

	*r = req
	return nil
}

type TGSRep = ASRep

func NewTGSRep(ticket, secretPart EncryptedData) (TGSRep, error) {
	return NewASRep(ticket, secretPart)
}
