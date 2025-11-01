package net

import (
	"encoding/binary"
	"fmt"
	"net"
)

// IPManager 管理网络的 IP 地址分配
type IPManager struct {
	CIDR      string   `json:"cidr"`      // 网络 CIDR，如 10.88.0.0/24
	Range     string   `json:"range"`     // IP 分配范围（可选）
	Gateway   string   `json:"gateway"`   // 网关地址
	Allocated []string `json:"allocated"` // 已分配的 IP 列表

	mask     net.IPMask
	rangeMin uint32
	rangeMax uint32
}

func newIPManager(cidr, ipRange, gateway string) (*IPManager, error) {
	m := &IPManager{CIDR: cidr, Range: ipRange, Gateway: gateway}
	if err := m.init(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *IPManager) init() error {
	_, network, err := net.ParseCIDR(m.CIDR)
	if err != nil {
		return fmt.Errorf("invalid CIDR %s: %w", m.CIDR, err)
	}
	m.mask = network.Mask
	m.setRange(network)

	if m.Range != "" {
		_, rangeNet, err := net.ParseCIDR(m.Range)
		if err != nil {
			return fmt.Errorf("invalid range CIDR %s: %w", m.Range, err)
		}
		m.setRange(rangeNet)
	}
	return nil
}

func (m *IPManager) setRange(network *net.IPNet) {
	networkIP := binary.BigEndian.Uint32(network.IP.To4())
	ones, bits := network.Mask.Size()
	broadcast := networkIP | (1<<(bits-ones) - 1)
	m.rangeMin = networkIP + 1
	m.rangeMax = broadcast - 1
}

// Allocate 分配一个可用 IP，返回 CIDR 格式（如 10.88.0.2/24）
func (m *IPManager) Allocate() (string, error) {
	used := m.usedIPs()
	for i := m.rangeMin; i <= m.rangeMax; i++ {
		ip := uint32ToIP(i)
		if !used[ip.String()] {
			cidr := m.toCIDR(ip)
			m.Allocated = append(m.Allocated, cidr)
			return cidr, nil
		}
	}
	return "", fmt.Errorf("no available IP in range")
}

func (m *IPManager) Release(cidr string) {
	for i, v := range m.Allocated {
		if v == cidr {
			m.Allocated = append(m.Allocated[:i], m.Allocated[i+1:]...)
			return
		}
	}
}

func (m *IPManager) GatewayIP() net.IP {
	if m.Gateway != "" {
		ip, _, _ := net.ParseCIDR(m.Gateway)
		return ip
	}
	// 默认使用网段第一个可用 IP
	_, network, _ := net.ParseCIDR(m.CIDR)
	return uint32ToIP(binary.BigEndian.Uint32(network.IP.To4()) + 1)
}

func (m *IPManager) usedIPs() map[string]bool {
	used := make(map[string]bool)
	for _, cidr := range m.Allocated {
		if ip, _, _ := net.ParseCIDR(cidr); ip != nil {
			used[ip.String()] = true
		}
	}
	if gw := m.GatewayIP(); gw != nil {
		used[gw.String()] = true
	}
	return used
}

func (m *IPManager) toCIDR(ip net.IP) string {
	ones, _ := m.mask.Size()
	return fmt.Sprintf("%s/%d", ip.String(), ones)
}

func uint32ToIP(ipValue uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipValue)
	return ip
}
