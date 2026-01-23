package crypto

import (
	"encoding/hex"

	"github.com/rizesql/kerberos/internal/protocol"
)

type KeyGenerator interface {
	Generate(size int) (protocol.SessionKey, error)
}

type RandomKeyGenerator struct{}

func NewKeyGenerator() *RandomKeyGenerator {
	return &RandomKeyGenerator{}
}

var _ KeyGenerator = &RandomKeyGenerator{}

func (RandomKeyGenerator) Generate(size int) (protocol.SessionKey, error) {
	return GenerateRandomKey(size)
}

type TestKeyGenerator struct {
	Key protocol.SessionKey
}

var _ KeyGenerator = &TestKeyGenerator{}

func NewTestKeyGenerator(key ...protocol.SessionKey) *TestKeyGenerator {
	if len(key) == 0 {
		expectedSessionKeyBytes, _ := hex.DecodeString("112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00")
		key, _ := protocol.NewSessionKey(expectedSessionKeyBytes)
		return &TestKeyGenerator{Key: key}
	}

	return &TestKeyGenerator{Key: key[0]}
}

func (m TestKeyGenerator) Generate(size int) (protocol.SessionKey, error) {
	return m.Key, nil
}
