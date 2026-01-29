package protocol

import (
	"encoding/json"
	"net/http"
)

type ASEndpoint struct{}

func (*ASEndpoint) Method() string { return http.MethodPost }
func (*ASEndpoint) Path() string   { return "/as" }

type ASReq struct {
	client     Principal
	service    Principal
	clientAddr Address
	nonce      Nonce
}

func NewASReq(client, service Principal, addr Address, nonce Nonce) (ASReq, error) {
	if client == (Principal{}) || service == (Principal{}) {
		return ASReq{}, ErrInvalidPrincipal
	}

	if nonce == (Nonce{}) {
		return ASReq{}, ErrNonceInvalid
	}

	return ASReq{
		client:     client,
		service:    service,
		clientAddr: addr,
		nonce:      nonce,
	}, nil
}

func (r ASReq) Client() Principal   { return r.client }
func (r ASReq) Service() Principal  { return r.service }
func (r ASReq) ClientAddr() Address { return r.clientAddr }
func (r ASReq) Nonce() Nonce        { return r.nonce }

type asReq struct {
	Client     Principal `json:"client"`
	Service    Principal `json:"service"`
	ClientAddr Address   `json:"client_addr"`
	Nonce      Nonce     `json:"nonce"`
}

func (r ASReq) MarshalJSON() ([]byte, error) {
	return json.Marshal(asReq{
		Client:     r.client,
		Service:    r.service,
		ClientAddr: r.clientAddr,
		Nonce:      r.nonce,
	})
}

func (r *ASReq) UnmarshalJSON(data []byte) error {
	var tmp asReq
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	req, err := NewASReq(tmp.Client, tmp.Service, tmp.ClientAddr, tmp.Nonce)
	if err != nil {
		return err
	}

	*r = req
	return nil
}

type ASRep struct {
	ticket     EncryptedData
	secretPart EncryptedData
}

func NewASRep(ticket, secretPart EncryptedData) (ASRep, error) {
	return ASRep{
		ticket:     ticket,
		secretPart: secretPart,
	}, nil
}

func (r ASRep) Ticket() EncryptedData     { return r.ticket }
func (r ASRep) SecretPart() EncryptedData { return r.secretPart }

type asRep struct {
	Ticket     EncryptedData `json:"ticket"`
	SecretPart EncryptedData `json:"secret_part"`
}

func (r ASRep) MarshalJSON() ([]byte, error) {
	return json.Marshal(asRep{
		Ticket:     r.ticket,
		SecretPart: r.secretPart,
	})
}

func (r *ASRep) UnmarshalJSON(data []byte) error {
	var tmp asRep
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	rep, err := NewASRep(tmp.Ticket, tmp.SecretPart)
	if err != nil {
		return err
	}

	*r = rep
	return nil
}
