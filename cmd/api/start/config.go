package start

import (
	"time"

	"github.com/urfave/cli/v3"
)

type Config struct {
	Port         string
	ServerKeyHex string
	ReplayWindow time.Duration
}

func newConfig(cmd *cli.Command) Config {
	port := cmd.String("port")
	if port == "" {
		port = ":9090"
	}
	return Config{
		Port:         port,
		ServerKeyHex: cmd.String("key"),
		ReplayWindow: 5 * time.Minute,
	}
}
