package as

import (
	"errors"
	"net/http"

	"github.com/rizesql/kerberos/internal/kdc"
	"github.com/rizesql/kerberos/internal/kdc/shared"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/server"
)

type Handler struct {
	protocol.ASEndpoint
	exchange *Exchange
	logger   *logging.Logger
}

func NewHandler(platform *kdc.Platform, cfg kdc.Config) *Handler {
	return &Handler{
		exchange: NewExchange(platform, cfg),
		logger:   platform.Logger,
	}
}

func (h *Handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := server.Decode[protocol.ASReq](r)
		if err != nil {
			h.logger.Error("failed to decode AS request", "err", err)
			server.EncodeError(w, http.StatusBadRequest, err)
			return
		}

		res, err := h.exchange.Handle(r.Context(), req)
		if err != nil {
			h.handleError(w, err)
			return
		}

		if err := server.Encode(w, http.StatusOK, res); err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}
	}
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, shared.ErrPrincipalNotFound):
		server.EncodeError(w, http.StatusNotFound, err)
	case errors.Is(err, shared.ErrWrongRealm):
		server.EncodeError(w, http.StatusBadRequest, err)
	default:
		h.logger.Error("AS exchange failed", "err", err)
		server.EncodeError(w, http.StatusInternalServerError, err)
	}
}
