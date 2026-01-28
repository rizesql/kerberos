package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rizesql/kerberos/cmd/kadmin/add"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "kadmin",
		Usage: "Kerberos Administration Tool",
		Commands: []*cli.Command{
			add.Cmd,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
