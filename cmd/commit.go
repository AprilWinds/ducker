package cmd

import (
	"ducker/container"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Commit = &cli.Command{
	Name:      "commit",
	Usage:     "Create a new image from a container's changes",
	ArgsUsage: "CONTAINER TAG",
	Action: func(c *cli.Context) error {
		if c.NArg() < 2 {
			return fmt.Errorf("usage: ducker commit CONTAINER TAG")
		}
		return container.Commit(c.Args().First(), c.Args().Get(1))
	},
}
