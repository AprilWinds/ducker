package cmd

import (
	"ducker/container"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Exec = &cli.Command{
	Name:      "exec",
	Usage:     "Run a command in a running container",
	ArgsUsage: "CONTAINER COMMAND [ARG...]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "interactive",
			Aliases: []string{"i", "it"},
			Usage:   "Keep STDIN open even if not attached",
		},
		&cli.BoolFlag{
			Name:    "detach",
			Aliases: []string{"d"},
			Usage:   "Detached mode: run command in background",
		},
		&cli.StringSliceFlag{
			Name:    "env",
			Aliases: []string{"e"},
			Usage:   "Set environment variables",
		},
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory inside the container",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() < 2 {
			return fmt.Errorf("usage: ducker exec [OPTIONS] CONTAINER COMMAND [ARG...]")
		}
		containerName := c.Args().Get(0)
		interactive := c.Bool("interactive") || !c.Bool("detach")
		return container.Exec(containerName, interactive, c.StringSlice("env"), c.Args().Slice()[1:], c.String("workdir"))
	},
}
