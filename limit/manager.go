package limit

import (
	"ducker/util"
	"fmt"
	"os"
	"strconv"
)

// Apply 应用 cgroup 资源限制
func Apply(containerID string, pid int, cpuLimit float64, memoryLimit uint64) error {
	if containerID == "" {
		return fmt.Errorf("container ID is empty")
	}
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	if cpuLimit > 0 {
		if err := applyCPULimit(containerID, pid, cpuLimit); err != nil {
			return err
		}
	}

	if memoryLimit > 0 {
		if err := applyMemoryLimit(containerID, pid, memoryLimit); err != nil {
			return err
		}
	}
	return nil
}

func applyCPULimit(containerID string, pid int, cpuLimit float64) error {
	cpuPath := util.GetCgroupCPUPath(containerID)
	if err := os.Mkdir(cpuPath, 0755); err != nil {
		return fmt.Errorf("create cpu cgroup: %w", err)
	}
	quota := strconv.Itoa(int(cpuLimit * 100000))
	if err := os.WriteFile(util.GetCPUQuotaPath(containerID), []byte(quota), 0644); err != nil {
		return fmt.Errorf("set cpu quota: %w", err)
	}
	if err := os.WriteFile(util.GetCPUTasksPath(containerID), []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("add pid to cpu cgroup: %w", err)
	}
	return nil
}

func applyMemoryLimit(containerID string, pid int, memoryLimit uint64) error {
	memPath := util.GetCgroupMemoryPath(containerID)
	if err := os.Mkdir(memPath, 0755); err != nil {
		return fmt.Errorf("create memory cgroup: %w", err)
	}
	if err := os.WriteFile(util.GetMemoryLimitPath(containerID), []byte(strconv.FormatUint(memoryLimit, 10)), 0644); err != nil {
		return fmt.Errorf("set memory limit: %w", err)
	}
	if err := os.WriteFile(util.GetMemoryTasksPath(containerID), []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("add pid to memory cgroup: %w", err)
	}
	return nil
}

func Remove(containerID string) {
	os.RemoveAll(util.GetCgroupCPUPath(containerID))
	os.RemoveAll(util.GetCgroupMemoryPath(containerID))
}
