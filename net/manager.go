package net

import (
	"ducker/util"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/vishvananda/netlink"
)

const (
	DefaultNetworkName = "ducker"
	DefaultSubnet      = "172.18.0.0/16"
	DefaultGateway     = "172.18.0.1/16"
)

// Init 初始化默认网络（创建或恢复）
func Init() error {
	driver, err := get(DefaultNetworkName)
	if err != nil {
		return Create(DefaultNetworkName, DefaultSubnet, DefaultGateway, "")
	}
	return driver.restore()
}

func Create(name, subnet, gateway, ipRange string) error {
	if _, err := get(name); err == nil {
		return fmt.Errorf("network %s already exists", name)
	}
	driver, err := newBridgeDriver(name, subnet, gateway, ipRange)
	if err != nil {
		return fmt.Errorf("create driver: %w", err)
	}
	if err := driver.setUp(); err != nil {
		return fmt.Errorf("set up driver: %w", err)
	}
	return nil
}

func List(quiet bool) error {
	entries, err := os.ReadDir(util.GetNetRootDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var networks []*BridgeDriver
	for _, entry := range entries {
		if entry.IsDir() {
			if driver, err := util.FindBy[BridgeDriver](util.TypeNet, entry.Name()); err == nil {
				networks = append(networks, driver)
			}
		}
	}

	if quiet {
		for _, n := range networks {
			fmt.Println(n.ID[:12])
		}
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "NETWORK ID\tNAME\tSUBNET\tGATEWAY")
	for _, n := range networks {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", n.ID[:12], n.Name, n.IPM.CIDR, n.IPM.Gateway)
	}
	return nil
}

func Remove(name string) error {
	if name == DefaultNetworkName {
		return fmt.Errorf("cannot remove default network %s", DefaultNetworkName)
	}
	driver, err := get(name)
	if err != nil {
		return err
	}
	return driver.tearDown()
}

func Connect(networkName, containerID string, pid int) error {
	driver, err := get(networkName)
	if err != nil {
		return err
	}
	return driver.connect(containerID, pid)
}

func Disconnect(networkName, containerID string) error {
	driver, err := get(networkName)
	if err != nil {
		return err
	}
	return driver.disconnect(containerID)
}

// GetContainerIP 获取指定网络中容器的 IP 地址
func GetContainerIP(networkName, containerID string) (string, error) {
	driver, err := get(networkName)
	if err != nil {
		return "", err
	}
	return driver.getContainerIP(containerID)
}

// SetupPortMappings 设置容器端口映射
// network: 网络名称或 ID
// containerID: 容器 ID
// ports: 端口映射 map[hostPort]containerPort
func SetupPortMappings(network, containerID string, ports map[string]string) error {
	if len(ports) == 0 {
		return nil
	}

	containerIP, err := GetContainerIP(network, containerID)
	if err != nil {
		return fmt.Errorf("get container ip: %w", err)
	}

	return SetupPortMapping(containerIP, ports)
}

// CleanPortMappings 清理容器端口映射
func CleanPortMappings(network, containerID string, ports map[string]string) error {
	if len(ports) == 0 {
		return nil
	}

	containerIP, err := GetContainerIP(network, containerID)
	if err != nil {
		// 如果获取不到 IP，可能容器已断开，跳过清理
		return nil
	}

	CleanPortMapping(containerIP, ports)
	return nil
}

func get(nameOrID string) (*BridgeDriver, error) {
	if nameOrID == "" {
		return nil, fmt.Errorf("network name is empty")
	}

	var networkID string
	if util.IsValidID(nameOrID) {
		networkID = nameOrID
	} else {
		networkID = util.GenerateID(nameOrID)
	}

	driver, err := util.FindBy[BridgeDriver](util.TypeNet, networkID)
	if err != nil {
		return nil, fmt.Errorf("network %s not found", nameOrID)
	}

	driver.bridge = &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridgeName(driver.ID)}}
	if driver.veths == nil {
		driver.veths = make(map[string]*netlink.Veth)
	}
	if driver.ContainerIPs == nil {
		driver.ContainerIPs = make(map[string]string)
	}
	if err := driver.IPM.init(); err != nil {
		return nil, fmt.Errorf("init ipm: %w", err)
	}
	return driver, nil
}
