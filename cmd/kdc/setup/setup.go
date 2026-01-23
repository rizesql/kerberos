package setup

import (
	"context"
	"fmt"

	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/shutdown"
)

func Run(ctx context.Context, cfg Config) error {
	logger := logging.New()
	logger.Info("Initializing KDC",
		"db", cfg.DBPath,
		"realm", cfg.Realm,
	)

	shutdowns := shutdown.New()

	db, err := kdb.New(kdb.Config{DSN: cfg.DBPath, Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	shutdowns.Register(db.Close)

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}
	logger.Info("Schema applied")

	key, err := crypto.DeriveKey(cfg.Secret, cfg.Realm+"krbtgt"+cfg.Realm)
	if err != nil {
		return fmt.Errorf("failed to derive master key: %w", err)
	}

	_, err = kdb.Query.CreatePrincipal(ctx, db, kdb.CreatePrincipalParams{
		PrimaryName: "krbtgt",
		Instance:    cfg.Realm,
		Realm:       cfg.Realm,
		KeyBytes:    key.Expose(),
		Kvno:        1,
	})
	if err != nil {
		return fmt.Errorf("failed to create krbtgt: %w", err)
	}

	logger.Info("KDC initialized successfully", "principal", fmt.Sprintf("krbtgt/%s@%s", cfg.Realm, cfg.Realm))
	if errs := shutdowns.Shutdown(ctx); len(errs) > 0 {
		err := &shutdown.ShutdownError{Errors: errs}
		logger.Error("shutdown failed", "error", err)
		return err
	}

	logger.Info("Server shutdown complete")
	return nil
}
