package cmd

import (
	"ducker/container"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Rm = &cli.Command{
	Name:  "rm",
	Usage: "Remove one or more containers",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Force removal of running container",
		},
		&cli.BoolFlag{
			Name:    "volumes",
			Aliases: []string{"v"},
			Usage:   "Remove anonymous volumes associated with the container",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("at least one container ID required")
		}
		return container.Rm(c.Args().Slice(), c.Bool("force"), c.Bool("volumes"))
	},
}
