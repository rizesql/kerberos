package start

import (
	"context"

	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "start",
	Usage: "Start the API Server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "key",
			Usage:    "Server's secret key (hex-encoded, must match KDC database)",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "port",
			Usage: "HTTP Listen Port (e.g. :9090)",
			Value: ":9090",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return Run(ctx, newConfig(cmd))
	},
}
