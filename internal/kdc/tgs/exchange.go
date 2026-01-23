package tgs

import (
	"context"
	"fmt"
	"time"

	"github.com/rizesql/kerberos/internal/clock"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/kdc"
	"github.com/rizesql/kerberos/internal/kdc/shared"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/replay"
)

type Exchange struct {
	db          kdb.Database
	logger      *logging.Logger
	clock       clock.Clock
	keygen      crypto.KeyGenerator
	replayCache replay.Cache
	cfg         kdc.Config
}

func NewExchange(platform *kdc.Platform, cfg kdc.Config) *Exchange {
	return &Exchange{
		db:          platform.Database,
		logger:      platform.Logger,
		clock:       platform.Clock,
		keygen:      platform.KeyGenerator,
		replayCache: platform.ReplayCache,
		cfg:         cfg,
	}
}

func (e *Exchange) Handle(ctx context.Context, req protocol.TGSReq) (protocol.TGSRep, error) {
	now := e.clock.Now().UTC()

	tgsPrincipal, err := protocol.NewKrbtgt(e.cfg.Realm)
	if err != nil {
		return protocol.TGSRep{}, fmt.Errorf("failed to create TGS principal: %w", err)
	}

	tgsKey, err := shared.FetchPrincipalKey(ctx, e.db, e.logger, tgsPrincipal)
	if err != nil {
		return protocol.TGSRep{}, fmt.Errorf("failed to fetch TGS key: %w", err)
	}

	tgt, err := shared.DecryptEntity[protocol.Ticket](tgsKey, req.TGT())
	if err != nil {
		e.logger.Warn("failed to decrypt TGT", "err", err)
		return protocol.TGSRep{}, fmt.Errorf("invalid TGT")
	}

	auth, err := shared.DecryptEntity[protocol.Authenticator](tgt.SessionKey(), req.Authenticator())
	if err != nil {
		e.logger.Warn("failed to decrypt authenticator", "err", err)
		return protocol.TGSRep{}, fmt.Errorf("invalid authenticator")
	}

	if err := e.validateAuthenticator(tgt, auth); err != nil {
		return protocol.TGSRep{}, err
	}

	serviceKey, err := shared.FetchPrincipalKey(ctx, e.db, e.logger, req.Server())
	if err != nil {
		return protocol.TGSRep{}, err
	}

	newSessionKey, err := e.keygen.Generate(32)
	if err != nil {
		return protocol.TGSRep{}, err
	}

	encTicket, err := e.encryptTicket(
		req.Server(),
		tgt,
		now,
		newSessionKey,
		serviceKey,
	)
	if err != nil {
		return protocol.TGSRep{}, err
	}

	encRepPart, err := e.encryptRepPart(
		req,
		now,
		newSessionKey,
		tgt.SessionKey(),
	)
	if err != nil {
		return protocol.TGSRep{}, err
	}

	return protocol.NewTGSRep(encTicket, encRepPart)
}

func (e *Exchange) validateAuthenticator(tgt protocol.Ticket, auth protocol.Authenticator) error {
	if tgt.Client().String() != auth.Client().String() {
		return fmt.Errorf("client mismatch: ticket=%s, auth=%s", tgt.Client(), auth.Client())
	}

	// Verify timestamp freshness.
	skew := e.clock.Now().Sub(auth.IssuedAt())
	if skew < -5*time.Minute || skew > 5*time.Minute {
		return fmt.Errorf("clock skew too great")
	}

	// Check for replay attack.
	if err := e.replayCache.Check(auth.Client().String(), auth.IssuedAt()); err != nil {
		e.logger.Warn("replay attack detected", "client", auth.Client(), "timestamp", auth.IssuedAt())
		return err
	}

	// Verify ticket validity period.
	if tgt.IsExpired(e.clock.Now()) {
		return fmt.Errorf("TGT expired")
	}

	return nil
}

func (e *Exchange) encryptTicket(
	server protocol.Principal,
	tgt protocol.Ticket,
	now time.Time,
	sessionKey protocol.SessionKey,
	serviceKey protocol.SessionKey,
) (protocol.EncryptedData, error) {
	ticket, err := protocol.NewTicket(
		server,
		tgt.Client(),
		tgt.ClientAddr(),
		now,
		e.cfg.TicketLifetime,
		sessionKey,
	)
	if err != nil {
		return protocol.EncryptedData{}, err
	}

	return shared.EncryptEntity(serviceKey, ticket)
}

func (e *Exchange) encryptRepPart(
	req protocol.TGSReq,
	now time.Time,
	sessionKey protocol.SessionKey,
	key protocol.SessionKey,
) (protocol.EncryptedData, error) {
	repPart, err := protocol.NewEncKDCRepPart(
		sessionKey,
		req.Nonce(),
		now,
		e.cfg.TicketLifetime,
		req.Server(),
	)
	if err != nil {
		return protocol.EncryptedData{}, err
	}

	return shared.EncryptEntity(key, repPart)
}
