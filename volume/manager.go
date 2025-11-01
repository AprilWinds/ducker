package volume

import (
	"ducker/util"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
)

type Info struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func Create(name string) error {
	vol, err := getOrCreate(name, false)
	if err != nil {
		return err
	}
	fmt.Println(vol.Name)
	return nil
}

func Get(nameOrID string) (*Info, error) {
	return util.FindBy[Info](util.TypeVolume, nameOrID)
}

func getOrCreate(name string, allowExist bool) (*Info, error) {
	if name == "" {
		name = util.GenerateID("")[:12]
	}

	if vol, err := Get(name); err == nil {
		if allowExist {
			return vol, nil
		}
		return nil, fmt.Errorf("volume %s already exists", name)
	}

	vol := &Info{
		ID:        util.GenerateID(name),
		Name:      name,
		CreatedAt: time.Now(),
	}

	if err := util.EnsureDir(util.GetVolumeDataDir(name)); err != nil {
		return nil, fmt.Errorf("create volume dir: %w", err)
	}

	data, _ := json.MarshalIndent(vol, "", "  ")
	if err := os.WriteFile(util.GetVolumeConfigPath(name), data, 0644); err != nil {
		os.RemoveAll(util.GetVolumeDir(name))
		return nil, fmt.Errorf("save config: %w", err)
	}

	return vol, nil
}

func Remove(name string) error {
	return os.RemoveAll(util.GetVolumeDir(name))
}

func Inspect(name string) error {
	vol, err := Get(name)
	if err != nil {
		return err
	}
	info := map[string]any{
		"ID":         vol.ID,
		"Name":       vol.Name,
		"CreatedAt":  vol.CreatedAt.Format(time.RFC3339),
		"Mountpoint": util.GetVolumeDataDir(vol.Name),
	}
	data, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println(string(data))
	return nil
}

func List() error {
	entries, err := os.ReadDir(util.GetVolumeRootDir())
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read volume dir: %w", err)
	}

	var volumes []*Info
	for _, entry := range entries {
		if entry.IsDir() {
			if vol, err := Get(entry.Name()); err == nil {
				volumes = append(volumes, vol)
			}
		}
	}

	slices.SortFunc(volumes, func(a, b *Info) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "VOLUME ID\tVOLUME NAME\tSIZE\tCREATED")
	for _, vol := range volumes {
		size := util.GetDirSize(util.GetVolumeDataDir(vol.Name))
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
			vol.ID[:12], vol.Name, util.FormatSize(size), util.FormatDuration(vol.CreatedAt))
	}
	return nil
}

// Mount 挂载卷或目录到容器路径
func Mount(sourcePath, containerPath, mergedDir string) error {
	containerPath = filepath.Join(mergedDir, containerPath)

	var hostPath string
	if strings.HasPrefix(sourcePath, "/") {
		hostPath = sourcePath
	} else {
		if _, err := getOrCreate(sourcePath, true); err != nil {
			return fmt.Errorf("get or create volume %s: %w", sourcePath, err)
		}
		hostPath = util.GetVolumeDataDir(sourcePath)
	}

	hostInfo, err := os.Stat(hostPath)
	if err != nil {
		return fmt.Errorf("stat host path %s: %w", hostPath, err)
	}

	if hostInfo.IsDir() {
		if err := util.EnsureDir(containerPath); err != nil {
			return fmt.Errorf("create mount point: %w", err)
		}
	} else {
		if err := util.EnsureDir(filepath.Dir(containerPath)); err != nil {
			return fmt.Errorf("create parent dir: %w", err)
		}
		if _, err := os.Stat(containerPath); os.IsNotExist(err) {
			if err := os.WriteFile(containerPath, nil, 0644); err != nil {
				return fmt.Errorf("create mount file: %w", err)
			}
		}
	}

	return syscall.Mount(hostPath, containerPath, "", syscall.MS_BIND, "")
}
