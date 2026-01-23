package start

import (
	"context"

	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "start",
	Usage: "Start the KDC service",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "db",
			Usage: "Path to SQLite database",
			Value: "kdc.db",
		},
		&cli.StringFlag{
			Name:     "realm",
			Usage:    "Kerberos Realm",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "port",
			Usage: "HTTP Listen Port (e.g. :8080)",
			Value: ":8080",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return Run(ctx, newConfig(cmd))
	},
}
