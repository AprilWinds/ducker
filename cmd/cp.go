package cmd

import (
	"ducker/container"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Cp = &cli.Command{
	Name:      "cp",
	Usage:     "Copy files/folders between a container and the local filesystem",
	ArgsUsage: "[OPTIONS] CONTAINER:SRC_PATH DEST_PATH | SRC_PATH CONTAINER:DEST_PATH",
	Action: func(c *cli.Context) error {
		if c.NArg() != 2 {
			return fmt.Errorf("usage: ducker cp CONTAINER:SRC_PATH DEST_PATH | SRC_PATH CONTAINER:DEST_PATH")
		}
		return container.Copy(c.Args().Get(0), c.Args().Get(1))
	},
}
