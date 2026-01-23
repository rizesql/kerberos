package protocol

import (
	"encoding/json"
	"errors"
)

var (
	ErrSessionKeyInvalid = errors.New("session key cannot be empty")
)

type SessionKey struct {
	value []byte
}

func NewSessionKey(key []byte) (SessionKey, error) {
	if len(key) == 0 {
		return SessionKey{}, ErrSessionKeyInvalid
	}

	value := make([]byte, len(key))
	copy(value, key)

	return SessionKey{value}, nil
}

func (s SessionKey) Expose() []byte { return s.value }

func (s SessionKey) IsZero() bool {
	return len(s.value) == 0
}

func (s SessionKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.value)
}

func (s *SessionKey) UnmarshalJSON(data []byte) error {
	var val []byte
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	key, err := NewSessionKey(val)
	if err != nil {
		return err
	}

	*s = key
	return nil
}
