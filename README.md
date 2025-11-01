# Ducker

一个用 Go 实现的轻量级容器运行时，用于学习和理解容器技术的核心原理。

## 特性

### 容器管理
- 创建、启动、停止、删除容器
- 交互式运行或后台运行模式
- 容器退出时自动删除（`--rm`）
- 在运行中的容器内执行命令（`exec`）
- 查看容器日志，支持持续跟踪
- 容器和主机之间复制文件

### 镜像管理
- 从 Duckerfile 构建镜像（支持 `FROM`、`RUN`、`COPY`、`ENV`、`WORKDIR`、`EXPOSE`、`CMD` 指令）
- 从容器创建镜像（`commit`）
- 导入/导出镜像为 tar.gz 归档
- **注意**：本项目不支持从远程仓库拉取镜像，可以编写duckerfile来构建镜像，或者直接使用alpine镜像(内置在项目中)

### 网络管理
- **Bridge 网络驱动**，基于 Linux 网桥实现容器网络
- 自动 IP 分配（IPAM）
- **NAT 网络**，容器可访问外部网络
- **端口映射**（DNAT），支持 TCP/UDP 协议
- 创建自定义网络，支持指定子网、网关、IP 范围
- 容器自动连接默认网络（`ducker`）

### 卷管理
- 创建、挂载、删除数据卷
- **Bind Mount** - 挂载主机目录到容器
- **命名卷** - 持久化数据存储

### 资源限制
- **CPU 限制** - 基于 cgroup v1 的 cpu.cfs_quota_us
- **内存限制** - 基于 cgroup v1 的 memory.limit_in_bytes

### 进程隔离
- **Linux Namespace 隔离**
  - UTS - 主机名隔离
  - PID - 进程 ID 隔离
  - Mount - 挂载点隔离
  - Network - 网络隔离
- **pivot_root** - 切换容器根文件系统
- **OverlayFS** - 分层文件系统，支持写时复制

## 系统要求

- Go 1.21+
- Linux 系统（需要 root 权限）
- cgroup v1（CPU 和内存子系统）
- iptables
- OverlayFS 支持

## 快速开始

### 安装

```bash
go build -o ducker .
```

### 运行第一个容器

**方式1**：直接使用内置的 alpine 镜像

```bash
sudo ./ducker run -it alpine /bin/sh
```

**方式2**：编写 Duckerfile 构建自定义镜像

```dockerfile
# Duckerfile
# 从内置的 alpine 镜像或本地已有镜像构建
FROM alpine:latest

WORKDIR /app

ENV APP_NAME=ducker-test APP_VERSION=1.0

COPY app.sh /app/app.sh

EXPOSE 8080

CMD ["/bin/sh", "/app/app.sh"]
```

```bash
# 构建镜像
sudo ./ducker build -t myapp .

# 运行容器
sudo ./ducker run -it myapp
```

**常用命令**：

```bash
# 交互式运行
sudo ./ducker run -it alpine /bin/sh

# 后台运行
sudo ./ducker run -d --name myapp alpine sleep 3600

# 查看运行中的容器
sudo ./ducker ps
```

## 命令参考

### run - 创建并运行容器

创建一个新容器并运行指定的命令。

```bash
ducker run [OPTIONS] IMAGE [COMMAND] [ARG...]
```

**选项：**

| 选项 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--name` | | 为容器指定名称 | `--name mycontainer` |
| `--interactive` | `-it`, `-i` | 交互模式，保持 STDIN 打开 | `-it` |
| `--detach` | `-d` | 后台运行容器 | `-d` |
| `--rm` | | 容器退出时自动删除 | `--rm` |
| `--workdir` | `-w` | 设置容器内的工作目录 | `-w /app` |
| `--env` | `-e` | 设置环境变量 | `-e KEY=value` |
| `--volume` | `-v` | 挂载卷，格式：主机路径:容器路径 | `-v /host:/container` |
| `--network` | | 连接到指定网络 | `--network mynet` |
| `--publish` | `-p` | 端口映射，格式：主机端口:容器端口 | `-p 8080:80` |
| `--cpus` | | CPU 核数限制 (浮点数) | `--cpus 0.5` |
| `--memory` | `-m` | 内存限制，支持 k/m/g 后缀 | `-m 256m` |

**示例：**

```bash
# 交互式运行
ducker run -it alpine /bin/sh

# 后台运行，设置资源限制
ducker run -d --name web --cpus 1.5 -m 512m alpine sleep 3600

# 挂载卷和端口映射
ducker run -d -p 8080:80 -v /data:/app/data --network mynet alpine

# 设置环境变量和工作目录
ducker run -it -e DB_HOST=localhost -w /app alpine /bin/sh
```

---

### ps - 列出容器

显示容器列表。

```bash
ducker ps [OPTIONS]
```

**选项：**

| 选项 | 简写 | 说明 |
|------|------|------|
| `--all` | `-a` | 显示所有容器（默认只显示运行中的） |
| `--quiet` | `-q` | 只显示容器 ID |

**示例：**

```bash
ducker ps           # 显示运行中的容器
ducker ps -a        # 显示所有容器
ducker ps -q        # 只显示容器 ID
```

---

### exec - 在容器中执行命令

在运行中的容器内执行命令。

```bash
ducker exec [OPTIONS] CONTAINER COMMAND [ARG...]
```

**选项：**

| 选项 | 简写 | 说明 |
|------|------|------|
| `--interactive` | `-it`, `-i` | 交互模式，保持 STDIN 打开 |
| `--detach` | `-d` | 后台执行命令 |
| `--env` | `-e` | 设置环境变量 |
| `--workdir` | `-w` | 设置工作目录 |

**示例：**

```bash
# 进入容器 shell
ducker exec -it mycontainer /bin/sh

# 执行单条命令
ducker exec mycontainer ls -la /app

# 设置环境变量执行
ducker exec -e DEBUG=1 mycontainer ./script.sh
```

---

### start - 启动容器

启动一个或多个已停止的容器。

```bash
ducker start [OPTIONS] CONTAINER [CONTAINER...]
```

**选项：**

| 选项 | 简写 | 说明 |
|------|------|------|
| `--attach` | `-a` | 附加 STDOUT/STDERR 并转发信号 |
| `--interactive` | `-i` | 与 --attach 一起使用时附加 STDIN |

**示例：**

```bash
ducker start mycontainer
ducker start -ai mycontainer    # 交互式启动
ducker start c1 c2 c3           # 同时启动多个容器
```

---

### stop - 停止容器

停止一个或多个运行中的容器。

```bash
ducker stop [OPTIONS] CONTAINER [CONTAINER...]
```

**选项：**

| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--time` | `-t` | 等待容器停止的秒数，超时后强制杀死 | 10 |

**示例：**

```bash
ducker stop mycontainer
ducker stop -t 30 mycontainer   # 等待 30 秒
ducker stop c1 c2 c3            # 同时停止多个容器
```

---

### rm - 删除容器

删除一个或多个容器。

```bash
ducker rm [OPTIONS] CONTAINER [CONTAINER...]
```

**选项：**

| 选项 | 简写 | 说明 |
|------|------|------|
| `--force` | `-f` | 强制删除运行中的容器 |
| `--volumes` | `-v` | 同时删除容器关联的匿名卷 |

**示例：**

```bash
ducker rm mycontainer
ducker rm -f running_container  # 强制删除
ducker rm -v mycontainer        # 同时删除关联卷
```

---

### logs - 查看容器日志

获取容器的日志输出。

```bash
ducker logs [OPTIONS] CONTAINER
```

**选项：**

| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--follow` | `-f` | 持续跟踪日志输出 | - |
| `--tail` | | 显示日志末尾的行数 | 100 |

**示例：**

```bash
ducker logs mycontainer
ducker logs -f mycontainer      # 持续跟踪
ducker logs --tail 50 mycontainer
```

---

### cp - 复制文件

在容器和本地文件系统之间复制文件或目录。

```bash
ducker cp CONTAINER:SRC_PATH DEST_PATH
ducker cp SRC_PATH CONTAINER:DEST_PATH
```

**示例：**

```bash
# 从容器复制到主机
ducker cp mycontainer:/app/config.json ./config.json

# 从主机复制到容器
ducker cp ./data mycontainer:/app/data
```

---

### images - 列出镜像

显示本地镜像列表。

```bash
ducker images [OPTIONS]
```

**选项：**

| 选项 | 简写 | 说明 |
|------|------|------|
| `--all` | `-a` | 显示所有镜像（包括中间层） |
| `--quiet` | `-q` | 只显示镜像名称 |

**示例：**

```bash
ducker images
ducker images -a
ducker images -q
```

---

### build - 构建镜像

从 Duckerfile 构建镜像。

```bash
ducker build [OPTIONS] PATH
```

**选项：**

| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--tag` | `-t` | 镜像名称和标签，格式：name:tag | - |
| `--file` | `-f` | Duckerfile 的路径 | PATH/Duckerfile |

**示例：**

```bash
ducker build -t myapp:v1 .
ducker build -t myapp:latest -f Duckerfile.dev .
```

**Duckerfile 指令：**

| 指令 | 说明 | 示例 |
|------|------|------|
| `FROM` | 指定基础镜像（仅支持 `alpine` 或本地已有镜像） | `FROM alpine` |
| `RUN` | 执行命令（构建时） | `RUN apk add --no-cache curl` |
| `COPY` | 复制文件到镜像 | `COPY app.sh /app/app.sh` |
| `WORKDIR` | 设置工作目录 | `WORKDIR /app` |
| `ENV` | 设置环境变量 | `ENV APP_NAME=myapp APP_VERSION=1.0` |
| `EXPOSE` | 声明暴露端口 | `EXPOSE 8080` |
| `CMD` | 设置默认启动命令（exec 格式） | `CMD ["/bin/sh", "/app/app.sh"]` |

**Duckerfile 示例：**

```dockerfile
# 基础镜像（仅支持 alpine 或本地已有镜像）
FROM alpine

# 设置工作目录
WORKDIR /app

# 设置环境变量
ENV APP_NAME=myapp APP_VERSION=1.0

# 复制应用文件
COPY app.sh /app/app.sh

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["/bin/sh", "/app/app.sh"]
```

---

### commit - 从容器创建镜像

将容器的当前状态保存为新镜像。

```bash
ducker commit CONTAINER TAG
```

**示例：**

```bash
ducker commit mycontainer myimage:v1
```

---

### save - 导出镜像

将一个或多个镜像导出为 tar 归档文件。

```bash
ducker save [OPTIONS] IMAGE [IMAGE...]
```

**选项：**

| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--output` | `-o` | 输出文件路径 | image.tar.gz |

**示例：**

```bash
ducker save -o backup.tar.gz myimage:v1
ducker save -o images.tar.gz image1:v1 image2:v2
```

---

### load - 导入镜像

从 tar 归档文件导入镜像。

```bash
ducker load [OPTIONS]
```

**选项：**

| 选项 | 简写 | 说明 |
|------|------|------|
| `--input` | `-i` | 输入文件路径（必需） |

**示例：**

```bash
ducker load -i backup.tar.gz
```

---

### rmi - 删除镜像

删除一个或多个镜像。

```bash
ducker rmi [OPTIONS] IMAGE [IMAGE...]
```

**选项：**

| 选项 | 简写 | 说明 |
|------|------|------|
| `--force` | `-f` | 强制删除镜像 |

**示例：**

```bash
ducker rmi myimage:v1
ducker rmi -f myimage:v1 oldimage:v2
```

---

### network - 网络管理

管理容器网络。

#### network create - 创建网络

```bash
ducker network create [OPTIONS] NAME
```

**选项：**

| 选项 | 说明 | 示例 |
|------|------|------|
| `--subnet` | 子网，CIDR 格式 | `--subnet 172.18.0.0/16` |
| `--gateway` | 网关地址 | `--gateway 172.18.0.1` |
| `--ip-range` | IP 分配范围 | `--ip-range 172.18.1.0/24` |

**示例：**

```bash
ducker network create mynet
ducker network create --subnet 172.20.0.0/16 --gateway 172.20.0.1 mynet
```

#### network ls - 列出网络

```bash
ducker network ls [OPTIONS]
```

**选项：**

| 选项 | 简写 | 说明 |
|------|------|------|
| `--quiet` | `-q` | 只显示网络名称 |

#### network rm - 删除网络

```bash
ducker network rm NAME [NAME...]
```

#### network connect - 连接容器到网络

```bash
ducker network connect NETWORK CONTAINER
```

#### network disconnect - 断开容器与网络的连接

```bash
ducker network disconnect NETWORK CONTAINER
```

**示例：**

```bash
ducker network create mynet
ducker network connect mynet mycontainer
ducker network disconnect mynet mycontainer
ducker network rm mynet
```

---

### volume - 卷管理

管理数据卷。

#### volume create - 创建卷

```bash
ducker volume create [NAME]
```

如果不指定名称，将自动生成。

#### volume ls - 列出卷

```bash
ducker volume ls
```

#### volume inspect - 查看卷详情

```bash
ducker volume inspect VOLUME
```

#### volume rm - 删除卷

```bash
ducker volume rm VOLUME [VOLUME...]
```

**示例：**

```bash
ducker volume create mydata
ducker volume ls
ducker volume inspect mydata
ducker volume rm mydata
```

---

## 数据存储

所有数据存储在 `/var/lib/ducker/` 目录下：

```
/var/lib/ducker/
├── containers/     # 容器数据
│   └── <id>/
│       ├── config.json   # 容器配置
│       ├── merged/       # OverlayFS 合并层
│       ├── upper/        # OverlayFS 上层（可写层）
│       └── work/         # OverlayFS 工作目录
├── images/         # 镜像数据
│   └── <id>/
│       ├── config.json   # 镜像配置
│       └── layers/       # 镜像层
├── volumes/        # 卷数据
│   └── <name>/
│       ├── config.json   # 卷配置
│       └── data/         # 卷数据
└── nets/           # 网络配置
    └── <name>/
        └── config.json   # 网络配置
```


## 许可证

MIT
