#!/bin/bash
# Ducker 全量测试脚本

set -e

cd "$(dirname "$0")/.."

DUCKER="./ducker"
TEST_DIR="./test"
PASSED=0
FAILED=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() {
    echo -e "${GREEN}✓ $1${NC}"
    ((PASSED++)) || true
}

fail() {
    echo -e "${RED}✗ $1${NC}"
    ((FAILED++)) || true
}

section() {
    echo -e "\n${YELLOW}=== $1 ===${NC}"
}

cleanup() {
    echo "清理环境..."
    $DUCKER stop test-bg test-cpu test-mem test-net test-port 2>/dev/null || true
    $DUCKER rm test-bg test-cpu test-mem test-net test-port 2>/dev/null || true
    $DUCKER volume rm test-vol 2>/dev/null || true
    $DUCKER network rm test-network 2>/dev/null || true
    $DUCKER rmi test-app:v1 2>/dev/null || true
    $DUCKER rmi loaded-alpine:latest 2>/dev/null || true
    $DUCKER rmi committed-image:v1 2>/dev/null || true
    rm -f /tmp/test-alpine.tar.gz /tmp/copied-example.txt 2>/dev/null || true
}

# 开始
echo ""
echo "=========================================="
echo "       Ducker 全量功能测试"
echo "=========================================="

cleanup

# 1. 镜像管理
section "1. 镜像管理"

if $DUCKER load -i $TEST_DIR/alpine.tar.gz alpine:latest 2>&1; then
    pass "load"
else
    fail "load"
fi

if $DUCKER images 2>&1 | grep -q alpine; then
    pass "images"
else
    fail "images"
fi

if $DUCKER save -o /tmp/test-alpine.tar.gz alpine:latest 2>&1; then
    pass "save"
else
    fail "save"
fi

if $DUCKER load -i /tmp/test-alpine.tar.gz loaded-alpine:latest 2>&1; then
    pass "load (from save)"
else
    fail "load (from save)"
fi

# 2. Volume 管理
section "2. Volume 管理"

if $DUCKER volume create test-vol 2>&1; then
    pass "volume create"
else
    fail "volume create"
fi

if $DUCKER volume ls 2>&1 | grep -q test-vol; then
    pass "volume ls"
else
    fail "volume ls"
fi

if $DUCKER volume inspect test-vol 2>&1 | grep -q test-vol; then
    pass "volume inspect"
else
    fail "volume inspect"
fi

# 3. 网络管理
section "3. 网络管理"

# 检查默认网络
if $DUCKER network ls 2>&1 | grep -q ducker; then
    pass "default network (ducker)"
else
    fail "default network (ducker)"
fi

# 创建自定义网络
if $DUCKER network create --subnet 192.168.100.0/24 --gateway 192.168.100.1/24 test-network 2>&1; then
    pass "network create"
else
    fail "network create"
fi

if $DUCKER network ls 2>&1 | grep -q test-network; then
    pass "network ls"
else
    fail "network ls"
fi

# 不能删除默认网络
if ! $DUCKER network rm ducker 2>&1; then
    pass "cannot remove default network"
else
    fail "cannot remove default network"
fi

# 4. 容器运行
section "4. 容器运行"

if $DUCKER run -d --name test-bg alpine:latest /bin/sh -c "while true; do sleep 10; done" 2>&1; then
    pass "run -d"
else
    fail "run -d"
fi

sleep 1

if $DUCKER ps 2>&1 | grep -q test-bg; then
    pass "ps"
else
    fail "ps"
fi

if $DUCKER exec test-bg /bin/sh -c "echo hello" 2>&1 | grep -q hello; then
    pass "exec"
else
    fail "exec"
fi

if $DUCKER logs test-bg 2>&1; then
    pass "logs"
else
    fail "logs"
fi

if $DUCKER stop test-bg 2>&1; then
    pass "stop"
else
    fail "stop"
fi

if $DUCKER start test-bg 2>&1; then
    pass "start"
else
    fail "start"
fi

# 5. 容器参数
section "5. 容器参数"

if $DUCKER run --rm --name test-w -w /tmp alpine:latest /bin/sh -c "pwd" 2>&1 | grep -q /tmp; then
    pass "run -w"
else
    fail "run -w"
fi

if $DUCKER run --rm --name test-e -e MY_VAR=hello alpine:latest /bin/sh -c 'echo $MY_VAR' 2>&1 | grep -q hello; then
    pass "run -e"
else
    fail "run -e"
fi

if $DUCKER run -d --name test-cpu --cpus 0.5 alpine:latest /bin/sh -c "sleep 2" 2>&1; then
    pass "run --cpus"
else
    fail "run --cpus"
fi

if $DUCKER run -d --name test-mem -m 64m alpine:latest /bin/sh -c "sleep 2" 2>&1; then
    pass "run -m"
else
    fail "run -m"
fi

sleep 3

# 清理 cpu/mem 测试容器
$DUCKER stop test-cpu test-mem 2>/dev/null || true
$DUCKER rm test-cpu test-mem 2>/dev/null || true

# 6. 容器网络
section "6. 容器网络"

# 容器应该自动连接到默认网络，可以 ping 网关
if $DUCKER run --rm --name test-net alpine:latest /bin/sh -c "ping -c 1 172.18.0.1" 2>&1 | grep -q "1 packets"; then
    pass "container network (ping gateway)"
else
    fail "container network (ping gateway)"
fi

# 容器应该有 eth0 网卡
if $DUCKER run --rm --name test-ifconfig alpine:latest /bin/sh -c "ip addr show eth0" 2>&1 | grep -q "inet 172.18"; then
    pass "container eth0"
else
    fail "container eth0"
fi

# 测试端口映射
if $DUCKER run -d --name test-port -p 18080:80 alpine:latest /bin/sh -c "while true; do echo -e 'HTTP/1.1 200 OK\r\n\r\nHello' | nc -l -p 80; done" 2>&1; then
    pass "run -p (port mapping)"
else
    fail "run -p (port mapping)"
fi

sleep 1

# 检查 iptables DNAT 规则
if iptables -t nat -L PREROUTING -n 2>&1 | grep -q "18080"; then
    pass "iptables DNAT rule"
else
    fail "iptables DNAT rule"
fi

$DUCKER stop test-port 2>/dev/null || true
$DUCKER rm test-port 2>/dev/null || true

# 7. 文件操作
section "7. 文件操作"

# cp 需要容器停止状态
$DUCKER stop test-bg 2>/dev/null || true

if $DUCKER cp $TEST_DIR/example.txt test-bg:/tmp/ 2>&1; then
    pass "cp to container"
else
    fail "cp to container"
fi

if $DUCKER cp test-bg:/tmp/example.txt /tmp/copied-example.txt 2>&1; then
    pass "cp from container"
else
    fail "cp from container"
fi

if cat /tmp/copied-example.txt 2>&1 | grep -q "example"; then
    pass "verify cp"
else
    fail "verify cp"
fi

# 重新启动容器
$DUCKER start test-bg 2>/dev/null || true

# 8. Volume 挂载
section "8. Volume 挂载"

if $DUCKER run --rm --name test-bind -v /tmp:/mnt alpine:latest /bin/sh -c "ls /mnt" 2>&1; then
    pass "bind mount"
else
    fail "bind mount"
fi

if $DUCKER run --rm --name test-volmnt -v test-vol:/data alpine:latest /bin/sh -c "echo 'test' > /data/t.txt && cat /data/t.txt" 2>&1 | grep -q test; then
    pass "volume mount"
else
    fail "volume mount"
fi

# 9. 镜像构建
section "9. 镜像构建"

# commit 需要容器停止状态
$DUCKER stop test-bg 2>/dev/null || true

if $DUCKER commit test-bg committed-image:v1 2>&1; then
    pass "commit"
else
    fail "commit"
fi

if $DUCKER build -t test-app:v1 -f Duckerfile $TEST_DIR 2>&1; then
    pass "build"
else
    fail "build"
fi

if $DUCKER images 2>&1 | grep -q test-app; then
    pass "verify build"
else
    fail "verify build"
fi

# 10. 清理
section "10. 清理"

if $DUCKER stop test-bg 2>/dev/null; $DUCKER rm test-bg 2>&1; then
    pass "rm container"
else
    fail "rm container"
fi

if $DUCKER volume rm test-vol 2>&1; then
    pass "volume rm"
else
    fail "volume rm"
fi

if $DUCKER network rm test-network 2>&1; then
    pass "network rm"
else
    fail "network rm"
fi

if $DUCKER rmi test-app:v1 2>&1; then
    pass "rmi"
else
    fail "rmi"
fi

# 结果
echo ""
echo "=========================================="
echo "             测试结果"
echo "=========================================="
echo -e "${GREEN}通过: $PASSED${NC}"
echo -e "${RED}失败: $FAILED${NC}"
echo "=========================================="

rm -f /tmp/test-alpine.tar.gz /tmp/copied-example.txt
$DUCKER rmi loaded-alpine:latest committed-image:v1 2>/dev/null || true

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}所有测试通过!${NC}\n"
    exit 0
else
    echo -e "\n${RED}有 $FAILED 个测试失败!${NC}\n"
    exit 1
fi
