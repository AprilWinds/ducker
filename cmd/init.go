package cmd

import (
	"ducker/container"

	"github.com/urfave/cli/v2"
)

var Init = &cli.Command{
	Name:   "init",
	Hidden: true,
	Action: func(c *cli.Context) error {
		return container.InitChildProc()
	},
}
