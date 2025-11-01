package util

import (
	"path/filepath"
)

const (
	baseDir      = "/var/lib/ducker"
	imageDir     = baseDir + "/images"
	containerDir = baseDir + "/containers"
	volumeDir    = baseDir + "/volumes"
	netDir       = baseDir + "/nets"

	cgroupCPUDir    = "/sys/fs/cgroup/cpu"
	cgroupMemoryDir = "/sys/fs/cgroup/memory"
)

// ========== 容器相关路径 ==========
func GetContainerRootDir() string {
	return containerDir
}

func GetContainerDir(containerID string) string {
	return filepath.Join(GetContainerRootDir(), containerID)
}

func GetContainerMergedDir(containerID string) string {
	return filepath.Join(GetContainerDir(containerID), "merged")
}

func GetContainerUpperDir(containerID string) string {
	return filepath.Join(GetContainerDir(containerID), "upper")
}

func GetContainerWorkDir(containerID string) string {
	return filepath.Join(GetContainerDir(containerID), "work")
}

func GetContainerConfigPath(containerID string) string {
	return filepath.Join(GetContainerDir(containerID), "config.json")
}

func GetContainerLogPath(containerID string) string {
	return filepath.Join(GetContainerMergedDir(containerID), "var/log/container.log")
}

// ========== 镜像相关路径 ==========
func GetImageRootDir() string {
	return imageDir
}

func GetImageDir(imageID string) string {
	return filepath.Join(GetImageRootDir(), imageID)
}

func GetImageConfigPath(imageID string) string {
	return filepath.Join(GetImageDir(imageID), "config.json")
}

func GetImageLayersDir(imageID string) string {
	return filepath.Join(GetImageDir(imageID), "layers")
}

func GetImageLayerDir(imageID, layerHash string) string {
	return filepath.Join(GetImageLayersDir(imageID), layerHash)
}

// ========== 卷相关路径 ==========

func GetVolumeRootDir() string {
	return volumeDir
}

func GetVolumeDir(name string) string {
	return filepath.Join(GetVolumeRootDir(), name)
}

func GetVolumeDataDir(name string) string {
	return filepath.Join(GetVolumeDir(name), "data")
}

func GetVolumeConfigPath(name string) string {
	return filepath.Join(GetVolumeDir(name), "config.json")
}

// ========== cgroup 相关路径 ==========

func GetCgroupCPUPath(containerID string) string {
	return filepath.Join(cgroupCPUDir, containerID)
}

func GetCgroupMemoryPath(containerID string) string {
	return filepath.Join(cgroupMemoryDir, containerID)
}

func GetCPUQuotaPath(containerID string) string {
	return filepath.Join(GetCgroupCPUPath(containerID), "cpu.cfs_quota_us")
}

func GetMemoryLimitPath(containerID string) string {
	return filepath.Join(GetCgroupMemoryPath(containerID), "memory.limit_in_bytes")
}

func GetCPUTasksPath(containerID string) string {
	return filepath.Join(GetCgroupCPUPath(containerID), "tasks")
}

func GetMemoryTasksPath(containerID string) string {
	return filepath.Join(GetCgroupMemoryPath(containerID), "tasks")
}

// ========== 网络相关路径 ==========

func GetNetRootDir() string {
	return netDir
}

func GetNetDir(netID string) string {
	return filepath.Join(netDir, netID)
}

func GetNetConfigPath(netID string) string {
	return filepath.Join(GetNetRootDir(), netID, "config.json")
}
