package setup

import (
	"context"

	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "setup",
	Usage: "Setup the KDC database",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "db",
			Usage: "Path to SQLite database",
			Value: "kdc.db",
		},
		&cli.StringFlag{
			Name:     "realm",
			Usage:    "Kerberos Realm (e.g. ATHENA.MIT.EDU)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "secret",
			Usage:    "Master secret for krbtgt principal",
			Required: true,
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return Run(ctx, newConfig(cmd))
	},
}
