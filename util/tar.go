package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CreateArchive 将源目录打包为 tar 或 tar.gz 文件
func CreateArchive(sourceDir, outputPath string, compress bool) error {
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", sourceDir)
	}

	outputDir := filepath.Dir(outputPath)
	if err := EnsureDir(outputDir); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// 打包目录内容（不包含目录本身）
	var args []string
	if compress {
		args = []string{"-czf", outputPath, "-C", sourceDir, "."}
	} else {
		args = []string{"-cf", outputPath, "-C", sourceDir, "."}
	}

	if err := exec.Command("tar", args...).Run(); err != nil {
		return fmt.Errorf("tar command failed: %w", err)
	}
	return nil
}

// ExtractArchive 将 tar 或 tar.gz 文件解压到指定目录
func ExtractArchive(archivePath, destDir string, compressed bool) error {
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", archivePath)
	}

	if err := EnsureDir(destDir); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	var args []string
	if compressed {
		args = []string{"-xzf", archivePath, "-C", destDir}
	} else {
		args = []string{"-xf", archivePath, "-C", destDir}
	}

	if err := exec.Command("tar", args...).Run(); err != nil {
		return fmt.Errorf("tar extract command failed: %w", err)
	}
	return nil
}
