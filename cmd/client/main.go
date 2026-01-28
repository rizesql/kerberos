package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rizesql/kerberos/cmd/client/start"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "client",
		Usage: "Kerberos Client with Web UI",
		Commands: []*cli.Command{
			start.Cmd,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
