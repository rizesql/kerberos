package crypto

import (
	"crypto/rand"
	"fmt"
	"io"

	"github.com/rizesql/kerberos/internal/protocol"
)

func GenerateRandomKey(size int) (protocol.SessionKey, error) {
	if size <= 0 {
		return protocol.SessionKey{}, fmt.Errorf("%w: size must be positive", ErrInvalidKey)
	}

	bytes := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return protocol.SessionKey{}, fmt.Errorf("%w: %v", ErrNonceGeneration, err)
	}

	return protocol.NewSessionKey(bytes)
}
