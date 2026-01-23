package as

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
)

type Config struct {
	Realm      string
	TicketLife time.Duration
}

type Exchange struct {
	db     kdb.Database
	logger *logging.Logger
	clock  clock.Clock
	keygen crypto.KeyGenerator
	cfg    Config
}

func NewExchange(platform *kdc.Platform, cfg Config) *Exchange {
	if cfg.TicketLife == 0 {
		cfg.TicketLife = 8 * time.Hour
	}

	return &Exchange{
		db:     platform.Database,
		logger: platform.Logger,
		clock:  platform.Clock,
		keygen: platform.KeyGenerator,
		cfg:    cfg,
	}
}

func (e *Exchange) Handle(ctx context.Context, req protocol.ASReq) (protocol.ASRep, error) {
	if err := e.validateRealm(req); err != nil {
		return protocol.ASRep{}, err
	}

	now := e.clock.Now().UTC()

	clientKey, err := shared.FetchPrincipalKey(ctx, e.db, e.logger, req.Client())
	if err != nil {
		return protocol.ASRep{}, err
	}

	serviceKey, err := shared.FetchPrincipalKey(ctx, e.db, e.logger, req.Service())
	if err != nil {
		return protocol.ASRep{}, err
	}

	sessionKey, err := e.keygen.Generate(32)
	if err != nil {
		return protocol.ASRep{}, err
	}

	encTicket, err := e.encryptTicket(req, now, sessionKey, serviceKey)
	if err != nil {
		return protocol.ASRep{}, err
	}

	encRepPart, err := e.encryptRepPart(req, now, sessionKey, clientKey)
	if err != nil {
		return protocol.ASRep{}, err
	}

	return protocol.NewASRep(encTicket, encRepPart)
}

func (e *Exchange) validateRealm(req protocol.ASReq) error {
	if string(req.Client().Realm()) != e.cfg.Realm {
		return fmt.Errorf("%w: client realm %s != kdc realm %s",
			shared.ErrWrongRealm, req.Client().Realm(), e.cfg.Realm)
	}
	return nil
}

func (e *Exchange) encryptTicket(
	req protocol.ASReq,
	now time.Time,
	sessionKey protocol.SessionKey,
	serviceKey protocol.SessionKey,
) (protocol.EncryptedData, error) {
	ticket, err := protocol.NewTicket(
		req.Service(),
		req.Client(),
		req.ClientAddr(),
		now,
		e.cfg.TicketLife,
		sessionKey,
	)
	if err != nil {
		return protocol.EncryptedData{}, err
	}

	return shared.EncryptEntity(serviceKey, ticket)
}

func (e *Exchange) encryptRepPart(
	req protocol.ASReq,
	now time.Time,
	sessionKey protocol.SessionKey,
	clientKey protocol.SessionKey,
) (protocol.EncryptedData, error) {
	repPart, err := protocol.NewEncKDCRepPart(
		sessionKey,
		req.Nonce(),
		now,
		e.cfg.TicketLife,
		req.Service(),
	)
	if err != nil {
		return protocol.EncryptedData{}, err
	}

	return shared.EncryptEntity(clientKey, repPart)
}
