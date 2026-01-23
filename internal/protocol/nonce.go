package protocol

import (
	"encoding/json"
	"errors"
)

var (
	ErrNonceInvalid = errors.New("nonce must be non-zero")
)

type Nonce struct{ val int32 }

func NewNonce(val int32) (Nonce, error) {
	if val == 0 {
		return Nonce{}, ErrNonceInvalid
	}
	return Nonce{val}, nil
}

func (n Nonce) Int32() int32 { return n.val }

func (n Nonce) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.val)
}

func (n *Nonce) UnmarshalJSON(data []byte) error {
	var val int32
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	nonce, err := NewNonce(val)
	if err != nil {
		return err
	}

	*n = nonce
	return nil
}
