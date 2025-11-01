package cmd

import (
	"ducker/image"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Save = &cli.Command{
	Name:      "save",
	Usage:     "Save one or more images to a tar archive",
	ArgsUsage: "IMAGE [IMAGE...]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Write to a file, instead of STDOUT",
			Value:   "image.tar.gz",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("at least one image name required")
		}
		return image.Save(c.Args().Slice(), c.String("output"))
	},
}
