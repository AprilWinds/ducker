package cmd

import (
	"ducker/container"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Logs = &cli.Command{
	Name:      "logs",
	Usage:     "Fetch the logs of a container",
	ArgsUsage: "CONTAINER",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "follow",
			Aliases: []string{"f"},
			Usage:   "Follow log output",
		},
		&cli.IntFlag{
			Name:  "tail",
			Value: 100,
			Usage: "Number of lines to show from the end of the logs",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			return fmt.Errorf("exactly one container ID required")
		}
		return container.Logs(c.Args().First(), c.Bool("follow"), c.Int("tail"))
	},
}
