package cmd

import (
	"ducker/volume"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Volume = &cli.Command{
	Name:  "volume",
	Usage: "Manage volumes",
	Subcommands: []*cli.Command{
		{
			Name:      "create",
			Usage:     "Create a volume",
			ArgsUsage: "[NAME]",
			Action: func(c *cli.Context) error {
				return volume.Create(c.Args().First())
			},
		},
		{
			Name:    "ls",
			Aliases: []string{"list"},
			Usage:   "List volumes",
			Action: func(c *cli.Context) error {
				return volume.List()
			},
		},
		{
			Name:      "rm",
			Aliases:   []string{"remove"},
			Usage:     "Remove one or more volumes",
			ArgsUsage: "VOLUME [VOLUME...]",
			Action: func(c *cli.Context) error {
				if c.NArg() < 1 {
					return fmt.Errorf("requires at least 1 argument")
				}
				for _, name := range c.Args().Slice() {
					if err := volume.Remove(name); err != nil {
						return fmt.Errorf("remove volume %s: %w", name, err)
					}
				}
				return nil
			},
		},
		{
			Name:      "inspect",
			Usage:     "Display detailed information on a volume",
			ArgsUsage: "VOLUME",
			Action: func(c *cli.Context) error {
				if c.NArg() < 1 {
					return fmt.Errorf("requires 1 argument")
				}
				return volume.Inspect(c.Args().First())
			},
		},
	},
}
