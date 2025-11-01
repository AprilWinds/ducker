package util

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// ========== 通用配置加载 ==========

type ResourceType string

const (
	TypeImage     ResourceType = "image"
	TypeContainer ResourceType = "container"
	TypeVolume    ResourceType = "volume"
	TypeNet       ResourceType = "net"
)

// FindBy 根据类型和ID加载配置，返回具体类型
func FindBy[T any](resType ResourceType, id string) (*T, error) {
	var configPath string
	switch resType {
	case TypeImage:
		configPath = GetImageConfigPath(id)
	case TypeContainer:
		configPath = GetContainerConfigPath(id)
	case TypeVolume:
		configPath = GetVolumeConfigPath(id)
	case TypeNet:
		configPath = GetNetConfigPath(id)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resType)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("%s %s not found", resType, id)
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &v, nil
}

// GenerateID 生成12位hex ID
// 如果传入name则基于name的hash生成，否则随机生成
func GenerateID(name string) string {
	if len(name) > 0 {
		hash := md5.Sum([]byte(name))
		return fmt.Sprintf("%012x", hash[:6])
	}
	timestamp := time.Now().UnixNano()
	randomValue := rand.Int63()
	combinedHash := timestamp ^ randomValue
	return fmt.Sprintf("%012x", combinedHash&0xffffffffffff)
}

// IsValidID 判断是否为有效的12位hex ID
func IsValidID(s string) bool {
	if len(s) != 12 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
