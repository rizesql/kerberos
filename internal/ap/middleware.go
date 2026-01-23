package ap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/server"
)

type contextKey string

const ClientContextKey contextKey = "kerberos_client"

var (
	ErrMissingAuthHeader = errors.New("missing Authorization header")
	ErrInvalidScheme     = errors.New("invalid Authorization scheme")
	ErrInvalidBase64     = errors.New("invalid base64 encoding")
	ErrInvalidAPReq      = errors.New("invalid AP-REQ format")
)

func Middleware(verifier *Verifier) server.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				server.EncodeError(w, http.StatusUnauthorized, ErrMissingAuthHeader)
				return
			}

			const prefix = "Kerberos "
			if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
				server.EncodeError(w, http.StatusUnauthorized, ErrInvalidScheme)
				return
			}

			encoded := authHeader[len(prefix):]
			data, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				server.EncodeError(w, http.StatusUnauthorized, ErrInvalidBase64)
				return
			}

			var apReq protocol.APReq
			if err := json.Unmarshal(data, &apReq); err != nil {
				server.EncodeError(w, http.StatusUnauthorized, ErrInvalidAPReq)
				return
			}

			result, err := verifier.Verify(apReq)
			if err != nil {
				server.EncodeError(w, http.StatusUnauthorized, err)
				return
			}

			ctx := context.WithValue(r.Context(), ClientContextKey, result.Client)
			next(w, r.WithContext(ctx))
		}
	}
}

func ClientFromContext(ctx context.Context) (protocol.Principal, bool) {
	client, ok := ctx.Value(ClientContextKey).(protocol.Principal)
	return client, ok
}
