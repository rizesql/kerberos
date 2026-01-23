package crypto

import (
	"crypto/sha256"

	"github.com/rizesql/kerberos/internal/protocol"
	"golang.org/x/crypto/pbkdf2"
)

const (
	defaultIterations = 4096
	defaultKeySize    = 32
)

type config struct {
	iterations int
	keySize    int
}

type Option func(*config)

func KDFOptions(opts ...Option) config {
	cfg := config{iterations: defaultIterations, keySize: defaultKeySize}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

func WithIterations(iter int) Option {
	return func(cfg *config) {
		if iter > 0 {
			cfg.iterations = iter
		}
	}
}

func WithKeySize(size int) Option {
	return func(cfg *config) {
		if size > 0 {
			cfg.keySize = size
		}
	}
}

func DeriveKey(password string, salt string, opts ...Option) (protocol.SessionKey, error) {
	cfg := KDFOptions(opts...)

	keyBytes := pbkdf2.Key(
		[]byte(password),
		[]byte(salt),
		cfg.iterations,
		cfg.keySize,
		sha256.New,
	)

	return protocol.NewSessionKey(keyBytes)
}
