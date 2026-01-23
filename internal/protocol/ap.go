package protocol

import "encoding/json"

type APReq struct {
	ticket        EncryptedData
	authenticator EncryptedData
}

func NewAPReq(ticket, authenticator EncryptedData) (APReq, error) {
	return APReq{
		ticket:        ticket,
		authenticator: authenticator,
	}, nil
}

func (r APReq) Ticket() EncryptedData        { return r.ticket }
func (r APReq) Authenticator() EncryptedData { return r.authenticator }

type apReq struct {
	Ticket        EncryptedData `json:"ticket"`
	Authenticator EncryptedData `json:"authenticator"`
}

func (r APReq) MarshalJSON() ([]byte, error) {
	return json.Marshal(apReq{
		Ticket:        r.ticket,
		Authenticator: r.authenticator,
	})
}

func (r *APReq) UnmarshalJSON(data []byte) error {
	var tmp apReq
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	req, err := NewAPReq(tmp.Ticket, tmp.Authenticator)
	if err != nil {
		return err
	}

	*r = req
	return nil
}
