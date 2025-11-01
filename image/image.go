package image

import (
	"ducker/util"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// RunOptions 镜像的默认运行配置（来自 Dockerfile）
type RunOptions struct {
	WorkDir string   `json:"workdir"`
	Env     []string `json:"env"`
	Port    []string `json:"port"`
	Cmd     []string `json:"cmd"`
}

type Image struct {
	Tag         string    `json:"tag"`
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Layers      []string  `json:"layers"`
	Size        int64     `json:"size"`
	Hidden      bool      `json:"hidden"`
	*RunOptions `json:"run_options"`
}

func (img *Image) getLayers() []string {
	if len(img.Layers) == 0 {
		return nil
	}
	result := make([]string, 0, len(img.Layers))
	for _, layer := range img.Layers {
		result = append(result, util.GetImageLayerDir(img.ID, layer))
	}
	return result
}

func (img *Image) save(outputPath string) error {
	return util.CreateArchive(util.GetImageDir(img.ID), outputPath, true)
}

func (img *Image) Remove() error {
	return os.RemoveAll(util.GetImageDir(img.ID))
}

func (img *Image) saveConfig() error {
	if img.RunOptions == nil {
		return fmt.Errorf("image config is nil")
	}

	if err := util.EnsureDir(util.GetImageDir(img.ID)); err != nil {
		return fmt.Errorf("ensure dir: %w", err)
	}
	img.Size = util.GetDirSize(util.GetImageDir(img.ID))

	data, err := json.MarshalIndent(img, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(util.GetImageConfigPath(img.ID), data, 0644)
}
