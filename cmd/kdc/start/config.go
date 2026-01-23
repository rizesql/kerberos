package start

import (
	"time"

	"github.com/urfave/cli/v3"
)

type Config struct {
	DBPath     string
	Realm      string
	Port       string
	TicketLife time.Duration
}

func newConfig(cmd *cli.Command) Config {
	return Config{
		DBPath:     cmd.String("db"),
		Realm:      cmd.String("realm"),
		Port:       cmd.String("port"),
		TicketLife: 8 * time.Hour,
	}
}
