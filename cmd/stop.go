package cmd

import (
	"ducker/container"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Stop = &cli.Command{
	Name:      "stop",
	Usage:     "Stop one or more running containers",
	ArgsUsage: "CONTAINER [CONTAINER...]",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:    "time",
			Aliases: []string{"t"},
			Usage:   "Seconds to wait for stop before killing the container",
			Value:   10,
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("at least one container ID required")
		}
		return container.Stop(c.Args().Slice(), c.Int("time"))
	},
}
