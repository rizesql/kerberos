package protocol_test

import (
	"encoding/json"
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/protocol"
)

func TestPrincipalValidation(t *testing.T) {
	_, err := protocol.NewPrincipal("", "admin", "REALM")
	assert.Err(t, err, protocol.ErrPrincipalEmptyPrimary)

	_, err = protocol.NewPrincipal("user", "", "")
	assert.Err(t, err, protocol.ErrPrincipalEmptyRealm)
}

func TestPrincipalSerialization(t *testing.T) {
	p, err := protocol.NewPrincipal("user", "admin", "REALM")
	assert.Err(t, err, nil)

	data, err := json.Marshal(p)
	assert.Err(t, err, nil)

	var loaded protocol.Principal
	err = json.Unmarshal(data, &loaded)
	assert.Err(t, err, nil)

	assert.Equal(t, loaded.Primary(), p.Primary())
	assert.Equal(t, loaded.Instance(), p.Instance())
	assert.Equal(t, loaded.Realm(), p.Realm())
}
