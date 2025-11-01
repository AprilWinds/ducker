package net

import (
	"ducker/util"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// BridgeDriver 网络驱动，管理网桥和容器网络连接
type BridgeDriver struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	IPM  *IPManager `json:"ipm"`

	ContainerIPs map[string]string `json:"container_ips"`

	bridge *netlink.Bridge
	veths  map[string]*netlink.Veth
}

func newBridgeDriver(name, subnet, gateway, ipRange string) (*BridgeDriver, error) {
	id := util.GenerateID(name)
	if err := util.EnsureDir(util.GetNetDir(id)); err != nil {
		return nil, fmt.Errorf("ensure dir: %w", err)
	}
	ipm, err := newIPManager(subnet, ipRange, gateway)
	if err != nil {
		return nil, fmt.Errorf("new ipm: %w", err)
	}
	return &BridgeDriver{
		ID:           id,
		Name:         name,
		IPM:          ipm,
		ContainerIPs: make(map[string]string),
		bridge:       &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridgeName(id)}},
		veths:        make(map[string]*netlink.Veth),
	}, nil
}

func bridgeName(id string) string {
	return fmt.Sprintf("br-%s", id[:6])
}

func (d *BridgeDriver) setUp() error {
	if err := d.createBridge(); err != nil {
		return fmt.Errorf("create bridge: %w", err)
	}
	if err := d.setupNAT(); err != nil {
		netlink.LinkDel(d.bridge)
		return fmt.Errorf("setup nat: %w", err)
	}
	return d.saveConfig()
}

// restore 恢复网络资源（网桥和 iptables 规则）
func (d *BridgeDriver) restore() error {
	if _, err := netlink.LinkByName(d.bridge.Attrs().Name); err != nil {
		if err := d.createBridge(); err != nil {
			return fmt.Errorf("restore bridge: %w", err)
		}
	}
	d.cleanNAT()
	if err := d.setupNAT(); err != nil {
		return fmt.Errorf("restore nat: %w", err)
	}
	return nil
}

func (d *BridgeDriver) tearDown() error {
	d.cleanNAT()
	if err := netlink.LinkDel(d.bridge); err != nil {
		return fmt.Errorf("del bridge: %w", err)
	}
	return os.RemoveAll(util.GetNetDir(d.ID))
}

func (d *BridgeDriver) setupNAT() error {
	return SetupBridgeNAT(d.bridge.Attrs().Name, d.IPM.CIDR)
}

func (d *BridgeDriver) cleanNAT() {
	CleanBridgeNAT(d.bridge.Attrs().Name, d.IPM.CIDR)
}

func (d *BridgeDriver) connect(containerID string, pid int) error {
	veth, err := d.createVethPair(containerID)
	if err != nil {
		return fmt.Errorf("create veth: %w", err)
	}

	if err := d.attachToBridge(veth); err != nil {
		return fmt.Errorf("attach to bridge: %w", err)
	}

	if err := d.configureContainer(veth.PeerName, pid, containerID); err != nil {
		return fmt.Errorf("configure container: %w", err)
	}

	return d.saveConfig()
}

func (d *BridgeDriver) disconnect(containerID string) error {
	// 根据容器 ID 生成 veth 名称并删除
	short := containerID[:6]
	vethName := "veth-" + short
	if link, err := netlink.LinkByName(vethName); err == nil {
		netlink.LinkDel(link)
	}
	delete(d.veths, containerID)

	// 释放容器 IP
	if ipCIDR, ok := d.ContainerIPs[containerID]; ok {
		d.IPM.Release(ipCIDR)
		delete(d.ContainerIPs, containerID)
	}

	return d.saveConfig()
}

// getContainerIP 获取容器的 IP 地址（不含掩码）
func (d *BridgeDriver) getContainerIP(containerID string) (string, error) {
	ipCIDR, ok := d.ContainerIPs[containerID]
	if !ok {
		return "", fmt.Errorf("container %s not connected to network %s", containerID, d.Name)
	}
	// 从 CIDR 中提取 IP
	ip, _, err := net.ParseCIDR(ipCIDR)
	if err != nil {
		return "", fmt.Errorf("parse ip: %w", err)
	}
	return ip.String(), nil
}

func (d *BridgeDriver) createBridge() error {
	if err := netlink.LinkAdd(d.bridge); err != nil {
		return fmt.Errorf("add bridge: %w", err)
	}

	addr, err := netlink.ParseAddr(d.IPM.Gateway)
	if err != nil {
		netlink.LinkDel(d.bridge)
		return fmt.Errorf("parse gateway: %w", err)
	}

	if err := netlink.AddrAdd(d.bridge, addr); err != nil {
		netlink.LinkDel(d.bridge)
		return fmt.Errorf("add addr: %w", err)
	}

	if err := netlink.LinkSetUp(d.bridge); err != nil {
		netlink.LinkDel(d.bridge)
		return fmt.Errorf("set up: %w", err)
	}

	return nil
}

func (d *BridgeDriver) createVethPair(containerID string) (*netlink.Veth, error) {
	short := containerID[:6]
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: "veth-" + short},
		PeerName:  "ceth-" + short,
	}
	if err := netlink.LinkAdd(veth); err != nil {
		return nil, fmt.Errorf("add veth: %w", err)
	}
	d.veths[containerID] = veth
	return veth, nil
}

func (d *BridgeDriver) attachToBridge(veth *netlink.Veth) error {
	link, err := netlink.LinkByName(veth.Name)
	if err != nil {
		return fmt.Errorf("get veth: %w", err)
	}
	if err := netlink.LinkSetMaster(link, d.bridge); err != nil {
		return fmt.Errorf("set master: %w", err)
	}
	return netlink.LinkSetUp(link)
}

func (d *BridgeDriver) configureContainer(peerName string, pid int, containerID string) error {
	peer, err := netlink.LinkByName(peerName)
	if err != nil {
		return fmt.Errorf("get peer: %w", err)
	}

	ipCIDR, err := d.IPM.Allocate()
	if err != nil {
		return fmt.Errorf("allocate ip: %w", err)
	}

	d.ContainerIPs[containerID] = ipCIDR

	if err := netlink.LinkSetNsPid(peer, pid); err != nil {
		return fmt.Errorf("move to ns: %w", err)
	}

	return d.setupContainerNs(peerName, ipCIDR, pid)
}

func (d *BridgeDriver) setupContainerNs(peerName, ipCIDR string, pid int) error {
	targetNs, err := netns.GetFromPid(pid)
	if err != nil {
		return fmt.Errorf("get ns: %w", err)
	}
	defer targetNs.Close()

	nlh, err := netlink.NewHandleAt(targetNs)
	if err != nil {
		return fmt.Errorf("create handle: %w", err)
	}
	defer nlh.Close()

	if lo, err := nlh.LinkByName("lo"); err == nil {
		nlh.LinkSetUp(lo)
	}

	link, err := nlh.LinkByName(peerName)
	if err != nil {
		return fmt.Errorf("get peer: %w", err)
	}

	if err := nlh.LinkSetName(link, "eth0"); err != nil {
		return fmt.Errorf("rename to eth0: %w", err)
	}
	link, err = nlh.LinkByName("eth0")
	if err != nil {
		return fmt.Errorf("get eth0: %w", err)
	}

	addr, err := netlink.ParseAddr(ipCIDR)
	if err != nil {
		return fmt.Errorf("parse addr: %w", err)
	}
	if err := nlh.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("add addr: %w", err)
	}
	if err := nlh.LinkSetUp(link); err != nil {
		return fmt.Errorf("set up: %w", err)
	}

	// 添加默认路由
	route := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Gw:        d.IPM.GatewayIP(),
	}
	if err := nlh.RouteAdd(route); err != nil {
		return fmt.Errorf("add route: %w", err)
	}

	return nil
}

func (d *BridgeDriver) saveConfig() error {
	f, err := os.Create(util.GetNetConfigPath(d.ID))
	if err != nil {
		return err
	}
	defer f.Close()

	data, _ := json.MarshalIndent(d, "", "  ")
	_, err = f.Write(data)
	return err
}
