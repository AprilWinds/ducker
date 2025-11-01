package main

import (
	"ducker/cmd"
	"ducker/image"
	"ducker/net"
	_ "embed"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"
)

//go:embed test/alpine.tar.gz
var alpineImage []byte

func preProcess(c *cli.Context) error {
	if c.Args().First() != "init" {
		if err := net.Init(); err != nil {
			slog.Warn("init network failed", "err", err)
		}

		if err := image.LoadBuiltin(alpineImage, "alpine:latest"); err != nil {
			slog.Warn("load builtin alpine failed", "err", err)
		}
	}
	return nil
}

func main() {
	app := &cli.App{
		Name:   "ducker",
		Usage:  "A simple container runtime",
		Before: preProcess,
		Commands: []*cli.Command{
			cmd.Build,
			cmd.Commit,
			cmd.Cp,
			cmd.Exec,
			cmd.Images,
			cmd.Init,
			cmd.Load,
			cmd.Logs,
			cmd.Network,
			cmd.Ps,
			cmd.Rm,
			cmd.Rmi,
			cmd.Run,
			cmd.Save,
			cmd.Start,
			cmd.Stop,
			cmd.Volume,
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("run failed", "err", err)
		os.Exit(1)
	}
}
