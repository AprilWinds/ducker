package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ensure dir %s: %w", dir, err)
	}
	return nil
}

func GetDirSize(dir string) int64 {
	var size int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0
	}
	return size
}

func CopyDir(sourcePath, destPath string) error {
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("source directory %s: %w", sourcePath, err)
	}

	cmd := exec.Command("cp", "-raf", sourcePath, destPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copy directory from %s to %s: %w", sourcePath, destPath, err)
	}

	return nil
}

// HashDir 计算目录内容的 hash
func HashDir(dir string) (string, error) {
	hasher := sha256.New()

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(dir, path)
			files = append(files, relPath)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk dir %s: %w", dir, err)
	}

	sort.Strings(files)

	for _, relPath := range files {
		hasher.Write([]byte(relPath))

		file, err := os.Open(filepath.Join(dir, relPath))
		if err != nil {
			return "", fmt.Errorf("open file %s: %w", relPath, err)
		}
		if _, err := io.Copy(hasher, file); err != nil {
			file.Close()
			return "", fmt.Errorf("read file %s: %w", relPath, err)
		}
		file.Close()
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))[:12], nil
}
