package cmd

import (
	"ducker/container"
	"ducker/image"
	"fmt"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

var Run = &cli.Command{
	Name:                   "run",
	Usage:                  "Create and run a new container",
	ArgsUsage:              "IMAGE [COMMAND] [ARG...]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "Assign a name to the container",
		},
		&cli.BoolFlag{
			Name:    "interactive",
			Aliases: []string{"it", "i"},
			Usage:   "Keep STDIN open even if not attached",
		},
		&cli.BoolFlag{
			Name:    "detach",
			Aliases: []string{"d"},
			Usage:   "Run container in background and print container ID",
		},
		&cli.BoolFlag{
			Name:  "rm",
			Usage: "Automatically remove the container when it exits",
		},
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory inside the container",
		},
		&cli.StringSliceFlag{
			Name:    "env",
			Aliases: []string{"e"},
			Usage:   "Set environment variables",
		},
		&cli.StringSliceFlag{
			Name:    "volume",
			Aliases: []string{"v"},
			Usage:   "Bind mount a volume (host_path:container_path)",
		},
		&cli.StringFlag{
			Name:  "network",
			Usage: "Connect a container to a network",
		},
		&cli.StringSliceFlag{
			Name:    "publish",
			Aliases: []string{"p"},
			Usage:   "Publish a container's port(s) to the host",
		},
		&cli.Float64Flag{
			Name:  "cpus",
			Usage: "Number of CPUs",
		},
		&cli.StringFlag{
			Name:    "memory",
			Aliases: []string{"m"},
			Usage:   "Memory limit",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() < 1 {
			return fmt.Errorf("please specify an image name")
		}

		imageName := c.Args().Get(0)
		containerName := c.String("name")

		imageRunOpts, err := image.GetRunOptions(imageName)
		if err != nil {
			return err
		}

		opts := buildRunOptions(c, imageRunOpts)
		_, err = container.Run(containerName, imageName, opts)
		return err
	},
}

func buildRunOptions(ctx *cli.Context, imageOpts *image.RunOptions) *container.RunOptions {
	coalesce := func(value, fallback string) string {
		if value != "" {
			return value
		}
		return fallback
	}
	coalesceSlice := func(value, fallback []string) []string {
		if len(value) > 0 {
			return value
		}
		return fallback
	}

	return &container.RunOptions{
		Interactive: ctx.Bool("interactive") || !ctx.Bool("detach"),
		AutoRemove:  ctx.Bool("rm"),
		Volume:      parseKeyValueArgs(ctx.StringSlice("volume")),
		Ports:       parseKeyValueArgs(ctx.StringSlice("publish")),
		Network:     ctx.String("network"),
		CPUs:        ctx.Float64("cpus"),
		Memory:      parseMemoryString(ctx.String("memory")),
		WorkDir:     coalesce(ctx.String("workdir"), imageOpts.WorkDir),
		Env:         coalesceSlice(ctx.StringSlice("env"), imageOpts.Env),
		Cmd:         coalesceSlice(ctx.Args().Tail(), imageOpts.Cmd),
	}
}

func parseKeyValueArgs(args []string) map[string]string {
	result := make(map[string]string)
	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// parseMemoryString 解析内存字符串，支持 k/m/g 后缀
func parseMemoryString(memStr string) uint64 {
	if memStr == "" {
		return 0
	}
	memStr = strings.ToLower(strings.TrimSpace(memStr))
	if memStr == "" {
		return 0
	}

	multiplier := uint64(1)
	suffix := memStr[len(memStr)-1]
	switch suffix {
	case 'k':
		multiplier = 1024
		memStr = memStr[:len(memStr)-1]
	case 'm':
		multiplier = 1024 * 1024
		memStr = memStr[:len(memStr)-1]
	case 'g':
		multiplier = 1024 * 1024 * 1024
		memStr = memStr[:len(memStr)-1]
	}

	val, err := strconv.ParseUint(memStr, 10, 64)
	if err != nil {
		return 0
	}
	return val * multiplier
}
