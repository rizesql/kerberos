package setup

import "github.com/urfave/cli/v3"

type Config struct {
	DBPath string
	Realm  string
	Secret string
}

func newConfig(cmd *cli.Command) Config {
	return Config{
		DBPath: cmd.String("db"),
		Realm:  cmd.String("realm"),
		Secret: cmd.String("secret"),
	}
}
