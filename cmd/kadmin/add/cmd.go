package add

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "add",
	Usage: "Add a new principal to the database",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "db",
			Usage:    "Path to the SQLite database",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "principal",
			Usage:    "Principal primary name (e.g. alice) or full name (e.g. http/api-server)",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "instance",
			Usage: "Principal instance (e.g. api-server). Optional if included in principal.",
		},
		&cli.StringFlag{
			Name:     "realm",
			Usage:    "Realm name (e.g. ATHENA.MIT.EDU)",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "password",
			Usage: "Password for the principal (mutually exclusive with --key)",
		},
		&cli.StringFlag{
			Name:  "key",
			Usage: "Hex-encoded 32-byte key (mutually exclusive with --password)",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		dbPath := cmd.String("db")
		principalStr := cmd.String("principal")
		instanceStr := cmd.String("instance")
		realmStr := cmd.String("realm")
		password := cmd.String("password")
		keyHex := cmd.String("key")

		if password != "" && keyHex != "" {
			return fmt.Errorf("cannot specify both --password and --key")
		}
		if password == "" && keyHex == "" {
			return fmt.Errorf("must specify either --password or --key")
		}

		// Parse principal
		var primary, instance string
		if instanceStr != "" {
			if strings.Contains(principalStr, "/") {
				return fmt.Errorf("cannot specify --instance when principal name (%s) already contains an instance", principalStr)
			}
			primary = principalStr
			instance = instanceStr
		} else {
			parts := strings.Split(principalStr, "/")
			if len(parts) == 1 {
				primary = parts[0]
			} else if len(parts) == 2 {
				primary = parts[0]
				instance = parts[1]
			} else {
				return fmt.Errorf("invalid principal format: %s", principalStr)
			}
		}

		// Validate realm
		if realmStr == "" {
			return fmt.Errorf("realm cannot be empty")
		}

		logger := logging.New()
		db, err := kdb.New(kdb.Config{DSN: dbPath, Logger: logger})
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		var keyBytes []byte
		if password != "" {
			// Salt = Realm + Primary + Instance (if any)
			salt := realmStr + primary + instance
			sk, err := crypto.DeriveKey(password, salt)
			if err != nil {
				return fmt.Errorf("failed to derive key: %w", err)
			}
			keyBytes = sk.Expose()
		} else {
			kb, err := hex.DecodeString(keyHex)
			if err != nil {
				return fmt.Errorf("failed to decode key hex: %w", err)
			}
			if len(kb) != 32 {
				return fmt.Errorf("key must be 32 bytes (64 hex characters), got %d bytes", len(kb))
			}
			keyBytes = kb
		}

		// Create Principal
		p, err := protocol.NewPrincipal(protocol.Primary(primary), protocol.Instance(instance), protocol.Realm(realmStr))
		if err != nil {
			return fmt.Errorf("invalid principal data: %w", err)
		}

		created, err := kdb.Query.CreatePrincipal(ctx, db, kdb.CreatePrincipalParams{
			PrimaryName: string(p.Primary()),
			Instance:    string(p.Instance()),
			Realm:       string(p.Realm()),
			KeyBytes:    keyBytes,
			Kvno:        1,
		})
		if err != nil {
			return fmt.Errorf("failed to create principal: %w", err)
		}

		fmt.Printf("Created principal: %s\n", principalFromDB(created).String())
		return nil
	},
}

func principalFromDB(p kdb.Principal) protocol.Principal {
	// We assume DB data is valid as we just inserted it
	pp, _ := protocol.NewPrincipal(protocol.Primary(p.PrimaryName), protocol.Instance(p.Instance), protocol.Realm(p.Realm))
	return pp
}
