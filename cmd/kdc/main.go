package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rizesql/kerberos/cmd/kdc/setup"
	"github.com/rizesql/kerberos/cmd/kdc/start"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "kdc",
		Usage: "Kerberos Key Distribution Center",
		Commands: []*cli.Command{
			setup.Cmd,
			start.Cmd,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
