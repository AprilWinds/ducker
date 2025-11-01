package container

import (
	"bufio"
	"ducker/image"
	"ducker/limit"
	"ducker/net"
	"ducker/util"
	"ducker/volume"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Status string

const (
	StatusRunning Status = "running"
	StatusExited  Status = "exited"
	EnvDuckerID          = "DUCKER_ID"
)

// RunOptions 容器运行时配置（镜像默认配置 + 用户运行时参数）
type RunOptions struct {
	// 基本运行选项
	Interactive bool `json:"interactive"`
	AutoRemove  bool `json:"auto_remove"`

	// 网络和存储
	Volume  map[string]string `json:"volumes"`
	Ports   map[string]string `json:"ports"`
	Network string            `json:"network"`

	// 容器命令配置
	WorkDir string   `json:"workdir"`
	Env     []string `json:"env"`
	Cmd     []string `json:"cmd"`

	// 资源限制
	CPUs   float64 `json:"cpus"`
	Memory uint64  `json:"memory"`
}

type container struct {
	ID        string    `json:"cid"`
	Name      string    `json:"name"`
	ImageTag  string    `json:"image_name"`
	CreatedAt time.Time `json:"created_at"`
	PID       int       `json:"pid"`
	Status    Status    `json:"status"`

	RunOptions `json:"run_options"`
}

func newContainer(name, imageTag string, opts *RunOptions) (*container, error) {
	id := util.GenerateID(name)
	c := &container{
		Name:       name,
		ID:         id,
		ImageTag:   imageTag,
		CreatedAt:  time.Now(),
		Status:     StatusExited,
		RunOptions: *opts,
	}

	layers, err := image.GetLayers(c.ImageTag)
	if err != nil {
		return nil, fmt.Errorf("get image layers: %w", err)
	}

	if err := c.setupRootfs(layers); err != nil {
		return nil, fmt.Errorf("setup rootfs: %w", err)
	}

	if err := c.saveConfig(); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return c, nil
}

func (c *container) start() error {
	if c.Status == StatusRunning {
		return fmt.Errorf("container already running")
	}

	// 1. 准备子进程
	cmd, syncRead, syncWrite, err := c.prepareChildProcess()
	if err != nil {
		return err
	}

	// 2. 启动子进程
	if err := cmd.Start(); err != nil {
		syncRead.Close()
		syncWrite.Close()
		return fmt.Errorf("start process: %w", err)
	}
	syncRead.Close() // 父进程关闭读端
	c.PID = cmd.Process.Pid
	c.Status = StatusRunning

	// 3. 配置容器资源（网络、cgroup）
	if err := c.setupResources(); err != nil {
		syncWrite.Close()
		c.killAndReset()
		return fmt.Errorf("setup resources: %w", err)
	}
	c.saveConfig()

	// 4. 通知子进程继续执行
	syncWrite.Write([]byte("GO"))
	syncWrite.Close()

	// 5. 等待子进程退出
	if c.Interactive {
		c.waitAndCleanup(cmd)
	}
	return nil
}

// prepareChildProcess 准备子进程命令，返回 cmd、同步管道读端和写端
func (c *container) prepareChildProcess() (*exec.Cmd, *os.File, *os.File, error) {
	syncRead, syncWrite, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create sync pipe: %w", err)
	}

	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
	}
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%s=%s", EnvDuckerID, c.ID),
		"DUCKER_SYNC_FD=3",
	)
	cmd.Env = append(cmd.Env, c.Env...)
	cmd.ExtraFiles = []*os.File{syncRead}

	if err := c.setupIO(cmd); err != nil {
		syncRead.Close()
		syncWrite.Close()
		return nil, nil, nil, err
	}

	return cmd, syncRead, syncWrite, nil
}

// setupIO 配置子进程的输入输出
func (c *container) setupIO(cmd *exec.Cmd) error {
	if c.Interactive {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return nil
	}

	logPath := util.GetContainerLogPath(c.ID)
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	return nil
}

// killAndReset 终止进程并重置状态
func (c *container) killAndReset() {
	syscall.Kill(c.PID, syscall.SIGKILL)
	c.Status = StatusExited
	c.PID = 0
}

// waitAndCleanup 等待进程退出并清理资源
func (c *container) waitAndCleanup(cmd *exec.Cmd) {
	cmd.Wait()
	c.Status = StatusExited
	c.PID = 0
	c.cleanupNetwork()
	c.saveConfig()
	if c.AutoRemove {
		c.remove()
	}
}

// cleanupNetwork 清理网络资源（端口映射 + 断开连接）
func (c *container) cleanupNetwork() {
	network := c.RunOptions.Network
	if network == "" {
		network = net.DefaultNetworkName
	}
	if len(c.RunOptions.Ports) > 0 {
		net.CleanPortMappings(network, c.ID, c.RunOptions.Ports)
	}
	net.Disconnect(network, c.ID)
}

func (c *container) stop(timeoutSec int) error {
	if c.Status != StatusRunning {
		return fmt.Errorf("container not running")
	}

	c.cleanupNetwork()

	if c.PID > 0 && syscall.Kill(c.PID, 0) == nil {
		syscall.Kill(c.PID, syscall.SIGTERM)
		if !c.waitProcessExit(timeoutSec) {
			syscall.Kill(c.PID, syscall.SIGKILL)
		}
	}

	c.Status = StatusExited
	c.PID = 0
	return c.saveConfig()
}

// waitProcessExit 等待进程退出，返回是否在超时前退出
func (c *container) waitProcessExit(timeoutSec int) bool {
	if timeoutSec <= 0 {
		return true
	}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(c.PID, 0) != nil {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func (c *container) exec(interactive bool, envVars, cmdArgs []string, workDir string) error {
	if c.Status != StatusRunning {
		return fmt.Errorf("container not running")
	}
	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	mergedDir := util.GetContainerMergedDir(c.ID)
	args := []string{"-t", fmt.Sprintf("%d", c.PID), "-m", "-p", "-u", "-i", "-n", "--root=" + mergedDir}
	if workDir != "" {
		args = append(args, "--wd="+workDir)
	}
	args = append(args, "--")
	args = append(args, cmdArgs...)

	task := exec.Command("nsenter", args...)
	task.Env = append(os.Environ(), envVars...)
	task.Stdout, task.Stderr = os.Stdout, os.Stderr
	if interactive {
		task.Stdin = os.Stdin
	}
	return task.Run()
}

func (c *container) logs(follow bool, tailLines int) error {
	logPath := util.GetContainerLogPath(c.ID)
	logFile, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	// 读取所有行
	var lines []string
	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan log file: %w", err)
	}

	// 输出尾部行
	if tailLines > 0 && len(lines) > tailLines {
		lines = lines[len(lines)-tailLines:]
	}
	for _, line := range lines {
		fmt.Println(line)
	}

	// 持续跟踪
	if !follow {
		return nil
	}
	reader := bufio.NewReader(logFile)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err != nil {
			return fmt.Errorf("read log: %w", err)
		}
		fmt.Print(line)
	}
}

func (c *container) copy(srcPath, destPath string, srcInContainer bool) error {
	if c.Status == StatusRunning {
		return fmt.Errorf("cannot copy from running container")
	}
	if srcInContainer {
		srcPath = filepath.Join(util.GetContainerMergedDir(c.ID), srcPath)
	} else {
		destPath = filepath.Join(util.GetContainerMergedDir(c.ID), destPath)
	}
	if err := util.EnsureDir(filepath.Dir(destPath)); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	if err := util.CopyDir(srcPath, destPath); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

func (c *container) remove() error {
	if c.Status == StatusRunning {
		return fmt.Errorf("cannot remove running container")
	}

	containerDir := util.GetContainerDir(c.ID)
	mergedDir := util.GetContainerMergedDir(c.ID)

	if err := syscall.Unmount(mergedDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount merged dir: %w", err)
	}

	if err := os.RemoveAll(containerDir); err != nil {
		return fmt.Errorf("remove container dir: %w", err)
	}

	limit.Remove(c.ID)

	return nil
}

func (c *container) commit(newImageTag string) error {
	if c.Status == StatusRunning {
		return fmt.Errorf("cannot commit running container")
	}
	return image.Create(c.ImageTag, newImageTag, util.GetContainerUpperDir(c.ID), &image.RunOptions{
		Env:     c.Env,
		Cmd:     c.Cmd,
		WorkDir: c.WorkDir,
	})
}

func (c *container) setupRootfs(lowerLayerPaths []string) error {
	upperDir := util.GetContainerUpperDir(c.ID)
	workDir := util.GetContainerWorkDir(c.ID)
	mergedDir := util.GetContainerMergedDir(c.ID)

	for _, dir := range []string{upperDir, workDir, mergedDir} {
		if err := util.EnsureDir(dir); err != nil {
			return fmt.Errorf("ensure dir %s: %w", dir, err)
		}
	}

	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		strings.Join(lowerLayerPaths, ":"), upperDir, workDir)
	if err := syscall.Mount("overlay", mergedDir, "overlay", 0, options); err != nil {
		return fmt.Errorf("mount overlay: %w", err)
	}
	return nil
}

func (c *container) setupResources() error {
	// 1. 设置资源限制
	if err := limit.Apply(c.ID, c.PID, c.RunOptions.CPUs, c.RunOptions.Memory); err != nil {
		return fmt.Errorf("set resource limit: %w", err)
	}

	// 2. 挂载卷
	mergedDir := util.GetContainerMergedDir(c.ID)
	for hostPath, containerPath := range c.RunOptions.Volume {
		if err := volume.Mount(hostPath, containerPath, mergedDir); err != nil {
			return fmt.Errorf("mount volume: %w", err)
		}
	}

	// 3. 连接网络
	networkName := c.RunOptions.Network
	if networkName == "" {
		networkName = net.DefaultNetworkName
		c.RunOptions.Network = networkName
	}
	if err := net.Connect(networkName, c.ID, c.PID); err != nil {
		return fmt.Errorf("connect network: %w", err)
	}

	// 4. 设置端口映射
	if len(c.RunOptions.Ports) > 0 {
		if err := net.SetupPortMappings(networkName, c.ID, c.RunOptions.Ports); err != nil {
			return fmt.Errorf("setup port mapping: %w", err)
		}
	}
	return nil
}

func (c *container) saveConfig() error {
	configPath := util.GetContainerConfigPath(c.ID)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// ========== 子进程 相关方法 ==========

func (c *container) runChildProc() error {
	syncFd := os.NewFile(3, "sync")
	if syncFd != nil {
		buf := make([]byte, 2)
		syncFd.Read(buf)
		syncFd.Close()
	}

	mergedDir := util.GetContainerMergedDir(c.ID)

	if err := c.pivotRoot(mergedDir); err != nil {
		return fmt.Errorf("pivot root: %w", err)
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	if c.WorkDir != "" {
		if err := os.MkdirAll(c.WorkDir, 0755); err != nil {
			return fmt.Errorf("create workdir: %w", err)
		}
		if err := syscall.Chdir(c.WorkDir); err != nil {
			return fmt.Errorf("chdir to workdir: %w", err)
		}
	}

	return c.execTask()
}

func (c *container) pivotRoot(newRoot string) error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("make root private: %w", err)
	}

	if err := syscall.Mount(newRoot, newRoot, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount: %w", err)
	}

	oldRoot := filepath.Join(newRoot, ".old_root")
	if err := os.MkdirAll(oldRoot, 0755); err != nil {
		return fmt.Errorf("create old_root: %w", err)
	}

	if err := syscall.PivotRoot(newRoot, oldRoot); err != nil {
		return fmt.Errorf("pivot_root: %w", err)
	}

	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir: %w", err)
	}

	if err := syscall.Unmount("/.old_root", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old_root: %w", err)
	}

	os.RemoveAll("/.old_root")
	return nil
}

func (c *container) execTask() error {
	if len(c.Cmd) == 0 {
		c.Cmd = []string{"/bin/sh"}
	}

	cmdPath, err := exec.LookPath(c.Cmd[0])
	if err != nil {
		cmdPath = c.Cmd[0]
	}

	if err := syscall.Exec(cmdPath, c.Cmd, os.Environ()); err != nil {
		return fmt.Errorf("exec %s: %w", cmdPath, err)
	}
	return nil
}
