package cmd

import (
	"ducker/image"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Rmi = &cli.Command{
	Name:      "rmi",
	Usage:     "Remove one or more images",
	ArgsUsage: "IMAGE [IMAGE...]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Force removal of images",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("at least one image name required")
		}
		return image.Rm(c.Args().Slice(), c.Bool("force"))
	},
}
