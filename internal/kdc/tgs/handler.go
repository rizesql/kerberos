package tgs

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
	protocol.TGSEndpoint
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
		req, err := server.Decode[protocol.TGSReq](r)
		if err != nil {
			h.logger.Error("failed to decode TGS request", "err", err)
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
	case errors.Is(err, shared.ErrWrongRealm):
		server.EncodeError(w, http.StatusBadRequest, err)
	case errors.Is(err, shared.ErrPrincipalNotFound):
		server.EncodeError(w, http.StatusNotFound, err)
	default:
		h.logger.Error("TGS exchange failed", "err", err)
		server.EncodeError(w, http.StatusInternalServerError, err)
	}
}
