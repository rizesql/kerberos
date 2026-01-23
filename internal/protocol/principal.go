package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrPrincipalEmptyPrimary = errors.New("principal primary name cannot be empty")
	ErrPrincipalEmptyRealm   = errors.New("principal realm cannot be empty")
)

type Primary string
type Instance string
type Realm string

type Principal struct {
	primary  Primary
	instance Instance
	realm    Realm
}

func NewPrincipal(primary Primary, instance Instance, realm Realm) (Principal, error) {
	if primary == "" {
		return Principal{}, ErrPrincipalEmptyPrimary
	}
	if realm == "" {
		return Principal{}, ErrPrincipalEmptyRealm
	}

	return Principal{
		primary:  primary,
		instance: instance,
		realm:    realm,
	}, nil
}

func NewKrbtgt(realm Realm) (Principal, error) {
	if realm == "" {
		return Principal{}, ErrPrincipalEmptyRealm
	}

	return Principal{
		primary:  "krbtgt",
		instance: Instance(realm),
		realm:    realm,
	}, nil
}

func (p Principal) Primary() Primary   { return p.primary }
func (p Principal) Instance() Instance { return p.instance }
func (p Principal) Realm() Realm       { return p.realm }

func (p Principal) String() string {
	if p.instance == "" {
		return fmt.Sprintf("%s@%s", p.primary, p.realm)
	}

	return fmt.Sprintf("%s.%s@%s", p.primary, p.instance, p.realm)
}

type principal struct {
	Primary  Primary  `json:"primary"`
	Instance Instance `json:"instance"`
	Realm    Realm    `json:"realm"`
}

func (p Principal) MarshalJSON() ([]byte, error) {
	return json.Marshal(principal{
		Primary:  p.primary,
		Instance: p.instance,
		Realm:    p.realm,
	})
}

func (p *Principal) UnmarshalJSON(data []byte) error {
	var tmp principal
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	pr, err := NewPrincipal(tmp.Primary, tmp.Instance, tmp.Realm)
	if err != nil {
		return err
	}

	*p = pr
	return nil
}
