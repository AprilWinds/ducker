package image

import (
	"ducker/util"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type layer struct {
	hash    string
	tmpPath string
}

// Builder 用于从 Dockerfile 构建镜像
type Builder struct {
	baseImg *Image
	tag     string
	opts    *RunOptions
	layers  []layer  // 新增的层
	tmpDirs []string // 需要清理的临时目录
}

func NewBuilder(baseImg *Image, tag string, opts *RunOptions) *Builder {
	newOpts := &RunOptions{}
	if baseImg.RunOptions != nil {
		*newOpts = *baseImg.RunOptions // 复制基础镜像配置
	}
	if opts != nil {
		*newOpts = *opts // 使用传入的配置覆盖
	}

	return &Builder{
		baseImg: baseImg,
		tag:     normalizeTag(tag),
		opts:    newOpts,
	}
}

func (b *Builder) Apply(instructions []*instruction) error {
	for _, inst := range instructions {
		if err := b.execute(inst); err != nil {
			return fmt.Errorf("execute %s: %w", inst.command, err)
		}
	}
	return nil
}

func (b *Builder) CreateNewLayer(layerDir string) error {
	hash, err := util.HashDir(layerDir)
	if err != nil {
		return fmt.Errorf("hash layer: %w", err)
	}
	b.layers = append(b.layers, layer{hash: hash, tmpPath: layerDir})
	return nil
}

func (b *Builder) Build() error {
	defer b.cleanup()

	err := b.createImage()
	if err != nil {
		return fmt.Errorf("create image: %w", err)
	}

	slog.Info("Successfully built image", "tag", b.tag)
	return nil
}

func (b *Builder) execute(inst *instruction) error {
	switch inst.command {
	case "FROM":
		// 已在解析阶段处理
	case "WORKDIR":
		b.opts.WorkDir = inst.args[0]
	case "ENV":
		b.opts.Env = append(b.opts.Env, inst.args...)
	case "EXPOSE":
		b.opts.Port = append(b.opts.Port, inst.args...)
	case "CMD":
		b.opts.Cmd = inst.args
	case "COPY":
		return b.execCopy(inst)
	case "RUN":
		return b.execRun(inst)
	}
	return nil
}

func (b *Builder) execCopy(inst *instruction) error {
	if len(inst.args) < 2 {
		return fmt.Errorf("COPY requires src and dest")
	}
	src, dest := inst.args[0], inst.args[1]

	layerDir, err := os.MkdirTemp("", "ducker-layer-")
	if err != nil {
		return fmt.Errorf("create layer dir: %w", err)
	}
	b.tmpDirs = append(b.tmpDirs, layerDir)

	destInLayer := filepath.Join(layerDir, dest)
	if err := util.EnsureDir(filepath.Dir(destInLayer)); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}
	if err := util.CopyDir(src, destInLayer); err != nil {
		return fmt.Errorf("copy files: %w", err)
	}

	return b.CreateNewLayer(layerDir)
}

func (b *Builder) execRun(inst *instruction) error {
	if len(inst.args) < 1 {
		return fmt.Errorf("RUN requires command")
	}

	tmpBase, err := os.MkdirTemp("", "ducker-run-")
	if err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}
	upperDir := filepath.Join(tmpBase, "upper")
	workDir := filepath.Join(tmpBase, "work")
	mergedDir := filepath.Join(tmpBase, "merged")
	b.tmpDirs = append(b.tmpDirs, tmpBase)

	for _, dir := range []string{upperDir, workDir, mergedDir} {
		if err := util.EnsureDir(dir); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		strings.Join(b.currentLowerDirs(), ":"), upperDir, workDir)
	if err := syscall.Mount("overlay", mergedDir, "overlay", 0, options); err != nil {
		return fmt.Errorf("mount overlayfs: %w", err)
	}
	defer syscall.Unmount(mergedDir, syscall.MNT_DETACH)

	cmd := exec.Command("chroot", mergedDir, "/bin/sh", "-c", inst.args[0])
	cmd.Env = b.opts.Env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command %q: %w", inst.args[0], err)
	}

	return b.CreateNewLayer(upperDir)
}

func (b *Builder) currentLowerDirs() []string {
	dirs := append([]string{}, b.baseImg.getLayers()...)
	for _, l := range b.layers {
		dirs = append(dirs, l.tmpPath)
	}
	return dirs
}

func (b *Builder) createImage() error {
	allLayers := append([]string{}, b.baseImg.Layers...)
	for _, l := range b.layers {
		allLayers = append(allLayers, l.hash)
	}

	imageID := util.GenerateID(b.tag)
	img := &Image{
		Tag:        b.tag,
		ID:         imageID,
		CreatedAt:  time.Now(),
		Layers:     allLayers,
		RunOptions: b.opts,
	}

	if err := util.EnsureDir(util.GetImageLayersDir(imageID)); err != nil {
		return fmt.Errorf("ensure layers dir: %w", err)
	}

	for _, srcPath := range b.baseImg.getLayers() {
		hash := filepath.Base(srcPath)
		dstPath := util.GetImageLayerDir(imageID, hash)
		if err := util.CopyDir(srcPath, dstPath); err != nil {
			img.Remove()
			return fmt.Errorf("copy base layer: %w", err)
		}
	}

	for _, l := range b.layers {
		dstPath := util.GetImageLayerDir(imageID, l.hash)
		if err := util.CopyDir(l.tmpPath, dstPath); err != nil {
			img.Remove()
			return fmt.Errorf("copy layer: %w", err)
		}
	}

	if err := img.saveConfig(); err != nil {
		img.Remove()
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}

func (b *Builder) cleanup() {
	for _, dir := range b.tmpDirs {
		os.RemoveAll(dir)
	}
}
