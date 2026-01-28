package start

import (
	"context"

	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "start",
	Usage: "Start the Client with Web UI",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "port",
			Usage: "HTTP Listen Port (e.g. :3000)",
			Value: ":3000",
		},
		&cli.StringFlag{
			Name:  "kdc",
			Usage: "KDC Address (e.g. http://localhost:8080)",
			Value: "http://localhost:8080",
		},
		&cli.StringFlag{
			Name:  "web",
			Usage: "Web directory path",
			Value: "./cmd/client/web",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return Run(ctx, newConfig(cmd))
	},
}
