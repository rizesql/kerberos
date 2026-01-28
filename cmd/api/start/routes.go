package start

import (
	"net/http"

	"github.com/rizesql/kerberos/internal/ap"
	"github.com/rizesql/kerberos/internal/server"
)

// WhoAmIRoute - returns the authenticated principal
type WhoAmIRoute struct{}

func (r *WhoAmIRoute) Method() string { return http.MethodGet }
func (r *WhoAmIRoute) Path() string   { return "/api/whoami" }
func (r *WhoAmIRoute) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client, ok := ap.ClientFromContext(r.Context())
		if !ok {
			server.EncodeError(w, http.StatusUnauthorized, ap.ErrMissingAuthHeader)
			return
		}

		server.Encode(w, http.StatusOK, map[string]string{
			"authenticated_as": client.String(),
			"message":          "Welcome to the protected resource!",
		})
	}
}

// SecretRoute - returns a secret message
type SecretRoute struct{}

func (r *SecretRoute) Method() string { return http.MethodGet }
func (r *SecretRoute) Path() string   { return "/api/secret" }
func (r *SecretRoute) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client, _ := ap.ClientFromContext(r.Context())

		server.Encode(w, http.StatusOK, map[string]string{
			"secret": "secret!",
			"for":    client.String(),
		})
	}
}
