package cmd

import (
	"ducker/container"

	"github.com/urfave/cli/v2"
)

var Ps = &cli.Command{
	Name:    "ps",
	Aliases: []string{"list"},
	Usage:   "List containers",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Show all containers (default shows just running)",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display container IDs",
		},
	},
	Action: func(c *cli.Context) error {
		return container.List(c.Bool("all"), c.Bool("quiet"))
	},
}
