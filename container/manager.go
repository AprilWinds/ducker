package container

import (
	"ducker/util"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
)

func Run(name, imageTag string, opts *RunOptions) (*container, error) {
	if name != "" {
		if _, err := Get(name); err == nil {
			return nil, fmt.Errorf("container name %s already exists", name)
		}
	}

	cont, err := newContainer(name, imageTag, opts)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	if err := cont.start(); err != nil {
		return nil, fmt.Errorf("run container: %w", err)
	}

	return cont, nil
}

func Start(targets []string, attach, interactive bool) error {
	for _, target := range targets {
		cont, err := Get(target)
		if err != nil {
			return fmt.Errorf("find container %s: %w", target, err)
		}
		if err := cont.start(); err != nil {
			return fmt.Errorf("start container %s: %w", target, err)
		}
	}
	return nil
}

func Stop(targets []string, timeout int) error {
	for _, target := range targets {
		cont, err := Get(target)
		if err != nil {
			return fmt.Errorf("find container %s: %w", target, err)
		}
		if err := cont.stop(timeout); err != nil {
			return fmt.Errorf("stop container %s: %w", target, err)
		}
	}
	return nil
}

func Exec(target string, interactive bool, env, cmd []string, workDir string) error {
	cont, err := Get(target)
	if err != nil {
		return fmt.Errorf("find container %s: %w", target, err)
	}
	if err := cont.exec(interactive, env, cmd, workDir); err != nil {
		return fmt.Errorf("exec container %s: %w", target, err)
	}
	return nil
}

func Copy(srcPath, destPath string) error {
	srcParts := strings.Split(srcPath, ":")
	destParts := strings.Split(destPath, ":")

	srcIsContainer := len(srcParts) == 2
	destIsContainer := len(destParts) == 2

	if srcIsContainer == destIsContainer {
		return fmt.Errorf("invalid format: one side must be container:path, the other must be host path")
	}

	if srcIsContainer {
		cont, err := Get(srcParts[0])
		if err != nil {
			return err
		}
		return cont.copy(srcParts[1], destPath, true)
	}

	cont, err := Get(destParts[0])
	if err != nil {
		return err
	}
	return cont.copy(srcPath, destParts[1], false)
}

func GetUpperDir(target string) (string, error) {
	cont, err := Get(target)
	if err != nil {
		return "", fmt.Errorf("find container %s: %w", target, err)
	}
	return util.GetContainerUpperDir(cont.ID), nil
}

func Rm(targets []string, force, volumes bool) error {
	for _, target := range targets {
		cont, err := Get(target)
		if err != nil {
			return fmt.Errorf("find container %s: %w", target, err)
		}
		if cont.Status == StatusRunning {
			if !force {
				return fmt.Errorf("container %s is running, use -f to force remove", target)
			}
			if err := cont.stop(0); err != nil {
				return fmt.Errorf("stop container %s: %w", target, err)
			}
		}
		if err := cont.remove(); err != nil {
			return fmt.Errorf("remove container %s: %w", target, err)
		}
	}
	return nil
}

func List(showAll, quiet bool) error {
	containers, err := getAllContainers()
	if err != nil {
		return fmt.Errorf("get containers: %w", err)
	}

	if !showAll {
		containers = slices.DeleteFunc(containers, func(c *container) bool {
			return c.Status != StatusRunning
		})
	}

	slices.SortFunc(containers, func(a, b *container) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	if quiet {
		for _, c := range containers {
			fmt.Println(c.ID)
		}
		return nil
	}
	printContainerInfo(containers)
	return nil
}

func Commit(target, tag string) error {
	c, err := Get(target)
	if err != nil {
		return fmt.Errorf("find container %s: %w", target, err)
	}
	if err := c.commit(tag); err != nil {
		return fmt.Errorf("commit container %s: %w", target, err)
	}
	return nil
}

func Logs(target string, follow bool, tail int) error {
	c, err := Get(target)
	if err != nil {
		return fmt.Errorf("find container %s: %w", target, err)
	}
	if err := c.logs(follow, tail); err != nil {
		return fmt.Errorf("get logs for %s: %w", target, err)
	}
	return nil
}

func Get(nameOrID string) (*container, error) {
	var containerID string
	if util.IsValidID(nameOrID) {
		containerID = nameOrID
	} else {
		containerID = util.GenerateID(nameOrID)
	}

	cont, err := util.FindBy[container](util.TypeContainer, containerID)
	if err != nil {
		return nil, fmt.Errorf("container not found: %s", nameOrID)
	}
	// 按名称查找时验证名称匹配
	if !util.IsValidID(nameOrID) && cont.Name != nameOrID {
		return nil, fmt.Errorf("container not found: %s", nameOrID)
	}
	return cont, nil
}

func getAllContainers() ([]*container, error) {
	entries, err := os.ReadDir(util.GetContainerRootDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []*container{}, nil
		}
		return nil, err
	}

	var containers []*container
	for _, entry := range entries {
		if entry.IsDir() {
			if cont, err := util.FindBy[container](util.TypeContainer, entry.Name()); err == nil {
				containers = append(containers, cont)
			}
		}
	}
	return containers, nil
}

func printContainerInfo(containers []*container) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	fmt.Fprintln(writer, "CONTAINER ID\tIMAGE\tCOMMAND\tCREATED\tSTATUS\tNAMES")
	for _, cont := range containers {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n",
			cont.ID, cont.ImageTag,
			strings.Join(cont.Cmd, " "),
			util.FormatDuration(cont.CreatedAt),
			string(cont.Status), cont.Name,
		)
	}
}

func InitChildProc() error {
	containerID := os.Getenv(EnvDuckerID)
	if containerID == "" {
		return fmt.Errorf("container ID not set")
	}

	cont, err := util.FindBy[container](util.TypeContainer, containerID)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cont.runChildProc(); err != nil {
		return fmt.Errorf("init container: %w", err)
	}
	return nil
}
