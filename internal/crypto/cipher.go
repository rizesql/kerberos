package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/rizesql/kerberos/internal/protocol"
)

var (
	ErrInvalidKey          = errors.New("invalid key size")
	ErrNonceGeneration     = errors.New("failed to generate nonce")
	ErrMalformedCiphertext = errors.New("malformed ciphertext")
	ErrAuthFailed          = errors.New("authentication failed (integrity check)")
)

func Encrypt(key protocol.SessionKey, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key.Expose())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNonceGeneration, err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func Decrypt(key protocol.SessionKey, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key.Expose())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("%w: data too short", ErrMalformedCiphertext)
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthFailed, err)
	}

	return plaintext, nil
}
