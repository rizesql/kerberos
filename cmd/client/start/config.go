package start

import (
	"github.com/urfave/cli/v3"
)

type Config struct {
	Port    string
	KDCAddr string
	WebDir  string
}

func newConfig(cmd *cli.Command) Config {
	port := cmd.String("port")
	if port == "" {
		port = ":3000"
	}
	kdcAddr := cmd.String("kdc")
	if kdcAddr == "" {
		kdcAddr = "http://localhost:8080"
	}
	webDir := cmd.String("web")
	if webDir == "" {
		webDir = "./cmd/client/web"
	}
	return Config{
		Port:    port,
		KDCAddr: kdcAddr,
		WebDir:  webDir,
	}
}
