package cmd

import (
	"ducker/image"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

var Load = &cli.Command{
	Name:      "load",
	Usage:     "Load an image from a tar.gz",
	ArgsUsage: "[FILE]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "input",
			Aliases: []string{"i"},
			Usage:   "Read from a tar.gz file instead of STDIN",
		},
	},
	Action: func(c *cli.Context) error {
		source := c.String("input")
		if source == "" && c.NArg() > 0 {
			source = c.Args().First()
		}
		if source == "" || !strings.HasSuffix(source, ".tar.gz") {
			return fmt.Errorf("input file must be a tar.gz (use -i flag or positional argument)")
		}
		tag := strings.TrimSuffix(filepath.Base(source), ".tar.gz")
		_, err := image.Load(source, tag)
		return err
	},
}
