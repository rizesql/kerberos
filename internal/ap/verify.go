package ap

import (
	"errors"
	"fmt"
	"time"

	"github.com/rizesql/kerberos/internal/clock"
	"github.com/rizesql/kerberos/internal/kdc/shared"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/replay"
)

var (
	ErrInvalidTicket        = errors.New("invalid ticket")
	ErrInvalidAuthenticator = errors.New("invalid authenticator")
	ErrClientMismatch       = errors.New("client mismatch between ticket and authenticator")
	ErrClockSkewTooGreat    = errors.New("clock skew too great")
	ErrTicketExpired        = errors.New("ticket expired")
)

type VerifyResult struct {
	Client     protocol.Principal
	SessionKey protocol.SessionKey
}

type Verifier struct {
	serverKey   protocol.SessionKey
	clock       clock.Clock
	replayCache replay.Cache
	maxSkew     time.Duration
}

func NewVerifier(
	serverKey protocol.SessionKey,
	clock clock.Clock,
	replayCache replay.Cache,
) *Verifier {
	return &Verifier{
		serverKey:   serverKey,
		clock:       clock,
		replayCache: replayCache,
		maxSkew:     5 * time.Minute,
	}
}

func (v *Verifier) Verify(req protocol.APReq) (VerifyResult, error) {
	ticket, err := shared.DecryptEntity[protocol.Ticket](v.serverKey, req.Ticket())
	if err != nil {
		return VerifyResult{}, ErrInvalidTicket
	}

	auth, err := shared.DecryptEntity[protocol.Authenticator](ticket.SessionKey(), req.Authenticator())
	if err != nil {
		return VerifyResult{}, ErrInvalidAuthenticator
	}

	if ticket.Client().String() != auth.Client().String() {
		return VerifyResult{}, fmt.Errorf("%w: ticket=%s, auth=%s",
			ErrClientMismatch, ticket.Client(), auth.Client())
	}

	now := v.clock.Now()
	skew := now.Sub(auth.IssuedAt())
	if skew < -v.maxSkew || skew > v.maxSkew {
		return VerifyResult{}, ErrClockSkewTooGreat
	}

	if err := v.replayCache.Check(auth.Client().String(), auth.IssuedAt()); err != nil {
		return VerifyResult{}, err
	}

	if ticket.IsExpired(now) {
		return VerifyResult{}, ErrTicketExpired
	}

	return VerifyResult{
		Client:     ticket.Client(),
		SessionKey: ticket.SessionKey(),
	}, nil
}
