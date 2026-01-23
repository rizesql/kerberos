package shared

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/protocol"
)

var (
	ErrPrincipalNotFound = errors.New("principal not found")
	ErrWrongRealm        = errors.New("request for wrong realm")
)

func FetchPrincipalKey(
	ctx context.Context,
	db kdb.Database,
	logger *logging.Logger,
	p protocol.Principal,
) (protocol.SessionKey, error) {
	row, err := kdb.Query.GetPrincipal(ctx, db, kdb.GetPrincipalParams{
		PrimaryName: string(p.Primary()),
		Instance:    string(p.Instance()),
		Realm:       string(p.Realm()),
	})
	if err != nil {
		logger.Warn("lookup failed", "principal", p, "err", err)
		return protocol.SessionKey{}, ErrPrincipalNotFound
	}

	return protocol.NewSessionKey(row.KeyBytes)
}

func EncryptEntity(key protocol.SessionKey, v json.Marshaler) (protocol.EncryptedData, error) {
	b, err := v.MarshalJSON()
	if err != nil {
		return protocol.EncryptedData{}, err
	}

	enc, err := crypto.Encrypt(key, b)
	if err != nil {
		return protocol.EncryptedData{}, err
	}

	return protocol.NewEncryptedData(enc)
}

func DecryptEntity[T any](key protocol.SessionKey, enc protocol.EncryptedData) (T, error) {
	var zero T

	bytes, err := crypto.Decrypt(key, enc.Ciphertext())
	if err != nil {
		return zero, err
	}

	var v T
	if err := json.Unmarshal(bytes, &v); err != nil {
		return zero, err
	}

	return v, nil
}
