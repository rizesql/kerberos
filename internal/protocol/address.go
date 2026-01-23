package protocol

import (
	"encoding/json"
	"errors"
	"net"
)

var ErrAddressInvalid = errors.New("address cannot be empty")

type Address struct {
	value net.IP
}

func NewAddress(ip net.IP) (Address, error) {
	if len(ip) == 0 {
		return Address{}, ErrAddressInvalid
	}

	value := make(net.IP, len(ip))
	copy(value, ip)

	return Address{value}, nil
}

func (a Address) IP() net.IP {
	if a.value == nil {
		return nil
	}

	out := make(net.IP, len(a.value))
	copy(out, a.value)

	return out
}

func (a Address) IsZero() bool {
	return len(a.value) == 0
}

func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.value)
}

func (a *Address) UnmarshalJSON(data []byte) error {
	var ip net.IP

	if err := json.Unmarshal(data, &ip); err != nil {
		return err
	}

	addr, err := NewAddress(ip)
	if err != nil {
		return err
	}

	*a = addr
	return nil
}
