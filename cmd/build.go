package cmd

import (
	"ducker/image"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Build = &cli.Command{
	Name:      "build",
	Usage:     "Build an image from a Duckerfile",
	ArgsUsage: "PATH",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "tag",
			Aliases: []string{"t"},
			Usage:   "Name and optionally a tag in the 'name:tag' format",
		},
		&cli.StringFlag{
			Name:    "file",
			Aliases: []string{"f"},
			Usage:   "Name of the Duckerfile (Default is 'PATH/Duckerfile')",
			Value:   "Duckerfile",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			return fmt.Errorf("exactly one build context path required")
		}
		return image.Build(c.String("tag"), c.String("file"), c.Args().First())
	},
}
