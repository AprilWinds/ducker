package net

import (
	"fmt"
	"os/exec"
	"strings"
)

// SetupPortMapping 设置端口映射规则 (DNAT)
// ports: map[hostPort]containerPort, 如 {"8080": "80", "8443/udp": "443/udp"}
func SetupPortMapping(containerIP string, ports map[string]string) error {
	for hostPort, containerPort := range ports {
		hostP, proto := parsePort(hostPort)
		containerP, _ := parsePort(containerPort)
		dest := fmt.Sprintf("%s:%s", containerIP, containerP)

		// nat 表 PREROUTING 链: 处理外部进入的流量
		if err := iptablesNAT("-A", "PREROUTING", "-p", proto, "--dport", hostP, "-j", "DNAT", "--to-destination", dest); err != nil {
			return fmt.Errorf("add PREROUTING %s->%s: %w", hostPort, containerPort, err)
		}
		// nat 表 OUTPUT 链: 处理本机发起的流量 (localhost)
		if err := iptablesNAT("-A", "OUTPUT", "-p", proto, "--dport", hostP, "-j", "DNAT", "--to-destination", dest); err != nil {
			iptablesNAT("-D", "PREROUTING", "-p", proto, "--dport", hostP, "-j", "DNAT", "--to-destination", dest)
			return fmt.Errorf("add OUTPUT %s->%s: %w", hostPort, containerPort, err)
		}
	}
	return nil
}

// CleanPortMapping 清理端口映射规则
func CleanPortMapping(containerIP string, ports map[string]string) {
	for hostPort, containerPort := range ports {
		hostP, proto := parsePort(hostPort)
		containerP, _ := parsePort(containerPort)
		dest := fmt.Sprintf("%s:%s", containerIP, containerP)

		iptablesNAT("-D", "PREROUTING", "-p", proto, "--dport", hostP, "-j", "DNAT", "--to-destination", dest)
		iptablesNAT("-D", "OUTPUT", "-p", proto, "--dport", hostP, "-j", "DNAT", "--to-destination", dest)
	}
}

// SetupBridgeNAT 配置网桥 NAT 规则，使容器可以访问外网
func SetupBridgeNAT(bridgeName, cidr string) error {
	// filter 表 FORWARD 链: 允许网桥流量转发
	if err := iptablesFilter("-I", "FORWARD", "-i", bridgeName, "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("add forward in: %w", err)
	}
	if err := iptablesFilter("-I", "FORWARD", "-o", bridgeName, "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("add forward out: %w", err)
	}
	// nat 表 POSTROUTING 链: SNAT/MASQUERADE 使容器访问外网
	if err := iptablesNAT("-A", "POSTROUTING", "-s", cidr, "!", "-o", bridgeName, "-j", "MASQUERADE"); err != nil {
		return fmt.Errorf("add masquerade: %w", err)
	}
	return nil
}

// CleanBridgeNAT 清理网桥 NAT 规则
func CleanBridgeNAT(bridgeName, cidr string) {
	iptablesFilter("-D", "FORWARD", "-i", bridgeName, "-j", "ACCEPT")
	iptablesFilter("-D", "FORWARD", "-o", bridgeName, "-j", "ACCEPT")
	iptablesNAT("-D", "POSTROUTING", "-s", cidr, "!", "-o", bridgeName, "-j", "MASQUERADE")
}

// parsePort 解析端口字符串，返回 (端口, 协议)
// 支持格式: "8080" 或 "8080/tcp" 或 "8080/udp"
func parsePort(s string) (port, proto string) {
	if parts := strings.Split(s, "/"); len(parts) == 2 {
		return parts[0], strings.ToLower(parts[1])
	}
	return s, "tcp"
}

// iptablesNAT 操作 nat 表
func iptablesNAT(args ...string) error {
	return exec.Command("iptables", append([]string{"-t", "nat"}, args...)...).Run()
}

// iptablesFilter 操作 filter 表 (默认表)
func iptablesFilter(args ...string) error {
	return exec.Command("iptables", append([]string{"-t", "filter"}, args...)...).Run()
}
