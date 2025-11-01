package cmd

import (
	"ducker/image"

	"github.com/urfave/cli/v2"
)

var Images = &cli.Command{
	Name:  "images",
	Usage: "List images",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Show all images (including intermediate)",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display image names",
		},
	},
	Action: func(c *cli.Context) error {
		return image.List(c.Bool("all"), c.Bool("quiet"))
	},
}
