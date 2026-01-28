package getkey

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "get-key",
	Usage: "Get the hex-encoded key of a principal",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "db",
			Usage:    "Path to the SQLite database",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "realm",
			Usage: "Realm name (optional if provided in principal string)",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		dbPath := cmd.String("db")
		realmFlag := cmd.String("realm")
		principalStr := cmd.Args().First()

		if principalStr == "" {
			return fmt.Errorf("must specify principal name as first argument")
		}

		primary, instance, realm, err := protocol.Parse(principalStr)
		if err != nil {
			return fmt.Errorf("invalid principal: %w", err)
		}

		if realm == "" {
			realm = protocol.Realm(realmFlag)
		}

		if realm == "" {
			return fmt.Errorf("must specify realm either via --realm or in principal string (e.g. alice@REALM)")
		}

		logger := logging.Noop()
		db, err := kdb.New(kdb.Config{DSN: dbPath, Logger: logger})
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		row, err := kdb.Query.GetPrincipal(ctx, db, kdb.GetPrincipalParams{
			PrimaryName: string(primary),
			Instance:    string(instance),
			Realm:       string(realm),
		})
		if err != nil {
			return fmt.Errorf("failed to get principal: %w", err)
		}

		fmt.Println(hex.EncodeToString(row.KeyBytes))
		return nil
	},
}
