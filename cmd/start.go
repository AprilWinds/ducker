package cmd

import (
	"ducker/container"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Start = &cli.Command{
	Name:      "start",
	Usage:     "Start one or more stopped containers",
	ArgsUsage: "CONTAINER [CONTAINER...]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "attach",
			Aliases: []string{"a"},
			Usage:   "Attach STDOUT/STDERR and forward signals",
		},
		&cli.BoolFlag{
			Name:    "interactive",
			Aliases: []string{"i"},
			Usage:   "Attach STDIN when --attach is used",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("at least one container ID required")
		}
		return container.Start(c.Args().Slice(), c.Bool("attach"), c.Bool("interactive"))
	},
}
