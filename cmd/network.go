package cmd

import (
	"ducker/container"
	network "ducker/net"
	"fmt"

	"github.com/urfave/cli/v2"
)

var Network = &cli.Command{
	Name:  "network",
	Usage: "Manage networks",
	Subcommands: []*cli.Command{
		{
			Name:  "create",
			Usage: "Create a new network",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "subnet",
					Usage: "Subnet in CIDR format",
				},
				&cli.StringFlag{
					Name:  "gateway",
					Usage: "IPv4 gateway in CIDR format",
				},
				&cli.StringFlag{
					Name:  "ip-range",
					Usage: "Allocate container IP from a sub-range",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() != 1 {
					return fmt.Errorf("network name required")
				}
				return network.Create(c.Args().First(), c.String("subnet"), c.String("gateway"), c.String("ip-range"))
			},
		},
		{
			Name:    "ls",
			Aliases: []string{"list"},
			Usage:   "List networks",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "quiet",
					Aliases: []string{"q"},
					Usage:   "Only display network names",
				},
			},
			Action: func(c *cli.Context) error {
				return network.List(c.Bool("quiet"))
			},
		},
		{
			Name:  "rm",
			Usage: "Remove one or more networks",
			Action: func(c *cli.Context) error {
				if c.NArg() == 0 {
					return fmt.Errorf("at least one network name required")
				}
				for _, name := range c.Args().Slice() {
					if err := network.Remove(name); err != nil {
						return fmt.Errorf("remove network %s: %w", name, err)
					}
				}
				return nil
			},
		},
		{
			Name:  "connect",
			Usage: "Connect a container to a network",
			Action: func(c *cli.Context) error {
				if c.NArg() != 2 {
					return fmt.Errorf("usage: ducker network connect NETWORK CONTAINER")
				}
				cont, err := container.Get(c.Args().Get(1))
				if err != nil {
					return fmt.Errorf("get container: %w", err)
				}
				return network.Connect(c.Args().Get(0), cont.ID, cont.PID)
			},
		},
		{
			Name:  "disconnect",
			Usage: "Disconnect a container from a network",
			Action: func(c *cli.Context) error {
				if c.NArg() != 2 {
					return fmt.Errorf("usage: ducker network disconnect NETWORK CONTAINER")
				}
				return network.Disconnect(c.Args().Get(0), c.Args().Get(1))
			},
		},
	},
}
