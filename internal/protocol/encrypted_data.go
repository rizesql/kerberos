package protocol

import (
	"encoding/json"
	"errors"
)

var (
	ErrInvalidCiphertext = errors.New("ciphertext cannot be empty")
)

type EncryptedData struct {
	ciphertext []byte
}

func NewEncryptedData(ciphertext []byte) (EncryptedData, error) {
	if len(ciphertext) == 0 {
		return EncryptedData{}, ErrInvalidCiphertext
	}

	c := make([]byte, len(ciphertext))
	copy(c, ciphertext)
	return EncryptedData{ciphertext: c}, nil
}

func (e EncryptedData) Ciphertext() []byte {
	c := make([]byte, len(e.ciphertext))
	copy(c, e.ciphertext)
	return c
}

type encryptedData struct {
	Ciphertext []byte `json:"ciphertext"`
}

func (e EncryptedData) MarshalJSON() ([]byte, error) {
	return json.Marshal(encryptedData{Ciphertext: e.ciphertext})
}

func (e *EncryptedData) UnmarshalJSON(data []byte) error {
	var tmp encryptedData
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	enc, err := NewEncryptedData(tmp.Ciphertext)
	if err != nil {
		return err
	}

	*e = enc
	return nil
}
