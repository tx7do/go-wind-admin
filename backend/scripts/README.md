# Script 脚本指南

本目录包含项目的各种自动化脚本，按功能分为三个主要模块：环境配置、Docker 部署和应用服务管理。

---

## 📁 目录结构

```
scripts/
├── env/                           # 环境配置脚本
│   ├── install_unix_prod.sh      # Unix/Linux 生产环境准备
│   ├── install_unix_dev.sh       # Unix/Linux 开发环境准备
│   ├── install_windows_dev.ps1   # Windows 开发环境配置
│   └── lib/                      # Unix/PowerShell 共享函数库
│       ├── common-utils.sh       # Unix 通用日志/权限工具
│       ├── host-utils.sh         # Unix hosts 文件管理工具
│       ├── common-utils.ps1      # PowerShell 通用日志工具
│       └── host-utils.ps1        # PowerShell hosts 文件管理工具
│
├── docker/                        # Docker 部署脚本
│   ├── full_deploy.sh            # 完整部署（应用+依赖）- Bash
│   ├── full_deploy.ps1           # 完整部署（应用+依赖）- PowerShell
│   ├── libs_only.sh              # 仅依赖部署 - Bash
│   └── libs_only.ps1             # 仅依赖部署 - PowerShell
│
└── deploy/                        # 应用服务管理脚本
    └── pm2_service.sh            # PM2 服务部署
```

---

## 🔧 环境配置脚本 (`env/`)

### install_unix_prod.sh ⭐ 推荐（生产）

**用途：** 生产环境的统一 Unix/Linux 环境准备脚本

**自动安装：**

- 基础工具（git, wget, jq 等）
- Node.js & npm (22.x LTS)
- pm2 应用管理器
- Docker（含 Docker Compose）
- Golang

**支持系统：**

- 🍎 macOS (Intel & Apple Silicon)
- 🐧 Ubuntu / Debian
- 🎯 CentOS / RHEL
- 🏔️ Rocky Linux / AlmaLinux
- 👻 Fedora

**使用方式：**

```bash
# 自动检测系统并安装
./scripts/env/install_unix_prod.sh
```

**特点：**

- ✅ 自动系统检测
- ✅ 智能包管理器选择
- ✅ 完整的错误处理
- ✅ 向后兼容旧脚本

**可选：初始化 hosts 记录（默认关闭）**

脚本支持通过环境变量自动写入 `/etc/hosts`（例如 `postgres.local`）。默认不会改动 hosts，需显式开启：

```bash
AUTO_INIT_HOSTS=true \
HOSTS_IP=127.0.0.1 \
HOSTS_DOMAIN_SUFFIX=.local \
HOSTS_SERVICES="postgres mysql redis" \
./scripts/env/install_unix_prod.sh
```

---

### install_unix_dev.sh

**用途：** 开发环境的 Unix/Linux 环境准备脚本

**自动安装：** （基于 install_unix_prod.sh）

- 所有生产环境工具
- **Go 开发工具：**
    - Protobuf 编译器插件（protoc-gen-go, protoc-gen-go-grpc 等）
    - 项目 CLI 工具（可在脚本中自定义）

**使用方式：**

```bash
./scripts/env/install_unix_dev.sh
```

**开发特性：**

- ✅ 包括所有生产工具
- ✅ Go 代码生成插件
- ✅ CLI 脚手架工具（可扩展）
- ✅ 开发友好的配置

**可选：初始化 hosts 记录（默认关闭）**

支持与生产脚本相同的 hosts 初始化参数：

```bash
AUTO_INIT_HOSTS=true \
HOSTS_IP=127.0.0.1 \
HOSTS_DOMAIN_SUFFIX=.local \
HOSTS_SERVICES="postgres mysql redis" \
./scripts/env/install_unix_dev.sh
```

---

### install_golang.sh

**用途：** 独立的 Go 运行时安装脚本

**功能：**

- 检测当前 Go 版本
- 下载最新稳定版
- 解压到指定位置
- 配置 PATH

**使用方式：**

```bash
./scripts/env/install_golang.sh
```

**支持：** macOS 和 Linux（自动检测 ARM64/AMD64）

---

### install_windows_dev.ps1

**用途：** Windows 开发环境完整配置脚本

**自动安装：**

1. **Scoop 包管理器** - 4 个函数封装
2. **Docker Desktop** - 4 个函数封装
3. **Go 环境** - 6 个函数封装

**使用方式：**

```powershell
# 完整安装
.\scripts\env\install_windows_dev.ps1

# 跳过 Docker
.\scripts\env\install_windows_dev.ps1 -SkipDocker

# 自动确认
.\scripts\env\install_windows_dev.ps1 -AutoConfirm
```

**特点：**

- ✅ 15 个专业函数，职责清晰
- ✅ 完整参数化设计
- ✅ 灵活定制（Scoop 包、Go 插件、CLI 工具）
- ✅ 详细文档（2000+ 行）

详见：Windows 脚本目录下的 `QUICK_START.md` 和 `GO_ENVIRONMENT_GUIDE.md`

---

## 🐳 Docker 部署脚本 (`docker/`)

### full_deploy.sh & full_deploy.ps1

**用途：** 启动完整的 Docker Compose 应用（应用 + 依赖）

**启动的服务：**

- 主应用服务（根据 docker-compose.yml 定义）
- PostgreSQL 数据库
- Redis 缓存
- Etcd 服务发现
- MinIO 对象存储
- Jaeger 分布式追踪

**使用方式：**

*Linux/macOS：*

```bash
# 使用默认配置（APP_ROOT=/root/app）
./scripts/docker/full_deploy.sh

# 自定义数据目录
APP_ROOT=/opt/app ./scripts/docker/full_deploy.sh
```

*Windows (PowerShell)：*

```powershell
# 使用默认配置（AppRoot=C:\app）
.\scripts\docker\full_deploy.ps1

# 自定义数据目录
.\scripts\docker\full_deploy.ps1 -AppRoot "D:\app"

# 自定义 Compose 文件
.\scripts\docker\full_deploy.ps1 -ComposeFile "docker-compose.yaml"
```

**使用场景：**

- ✅ 完整的本地开发环境
- ✅ 快速验收测试
- ✅ 生产环境部署
- ✅ 一键启动所有服务

**环境变量（Bash）/ 参数（PowerShell）：**

- `APP_ROOT` - 数据卷根目录（默认：/root/app 或 C:\app）
- `COMPOSE_FILE` - Compose 文件路径（可选，默认自动检测）

---

### libs_only.sh & libs_only.ps1

**用途：** 启动仅包含依赖的 Docker Compose（不启动应用）

**启动的服务：**

- PostgreSQL 数据库
- Redis 缓存
- MinIO 对象存储
- Jaeger 分布式追踪

**不启动：** 主应用服务（应在本地运行）

**使用方式：**

*Linux/macOS：*

```bash
# 启动依赖服务
./scripts/docker/libs_only.sh

# 然后在另一个终端本地运行应用
gow run admin

# 自定义配置
APP_ROOT=/opt/app COMPOSE_FILE=custom.yaml ./scripts/docker/libs_only.sh
```

*Windows (PowerShell)：*

```powershell
# 启动依赖服务
.\scripts\docker\libs_only.ps1

# 然后在另一个 PowerShell 本地运行应用
gow run admin

# 自定义配置
.\scripts\docker\libs_only.ps1 -AppRoot "D:\app" -ComposeFile "docker-compose.libs.yaml"
```

**使用场景：**

- ✅ 本地开发（Docker 依赖 + 本地应用代码）
- ✅ IDE 调试（在编辑器中运行应用）
- ✅ 快速迭代（无需重启应用容器）
- ✅ 多人协作开发

**环境变量（Bash）/ 参数（PowerShell）：**

- `APP_ROOT` - 数据卷根目录（默认：/root/app 或 C:\app）
- `COMPOSE_FILE` - Compose 文件路径（默认：docker-compose.libs.yaml）

---

## 🚀 应用服务脚本 (`deploy/`)

### pm2_service.sh

**用途：** 通过 PM2 部署和管理应用服务

**功能：**

1. 检查环境（Go 版本、PM2 版本）
2. 加载 `.env` 环境变量
3. 构建项目
4. 通过 PM2 启动/重启服务
5. 生成启动日志

**使用方式：**

```bash
./scripts/deploy/pm2_service.sh
```

**前置条件：**

- ✅ Go 已安装
- ✅ PM2 已安装（全局）
- ✅ `.env` 文件存在（可选）
- ✅ 项目根目录有 `Makefile`

**生成的输出：**

- PM2 应用启动状态
- 构建日志
- 启动脚本

---

## 💡 使用场景和工作流

### 场景 1：完整本地开发环境（Linux/macOS）

```bash
# 1. 准备开发环境
./scripts/env/install_unix_dev.sh

# 2. 启动所有服务
./scripts/docker/full_deploy.sh

# 3. 开始开发
gow run admin
```

---

### 场景 2：本地开发 + 远程依赖

```bash
# 1. 准备开发环境
./scripts/env/install_unix_dev.sh

# 2. 启动本地依赖
./scripts/docker/libs_only.sh

# 3. 本地运行应用代码
gow run admin

# 4. 在 IDE 中调试应用（推荐）
```

---

### 场景 3：生产环境部署

```bash
# 1. 准备生产环境
./scripts/env/install_unix_prod.sh

# 2. 启动完整应用
./scripts/docker/full_deploy.sh

# 3. 通过 PM2 管理（可选）
./scripts/deploy/pm2_service.sh
```

---

### 场景 4：Windows 开发环境

```powershell
# 1. 以管理员身份打开 PowerShell
# 2. 运行完整配置
.\scripts\env\install_windows_dev.ps1

# 3. 启动 Docker 依赖（方式 A - 推荐本地开发）
.\scripts\docker\libs_only.ps1

# 或启动完整应用（方式 B - 一键启动）
.\scripts\docker\full_deploy.ps1

# 4. 在 IDE 中开发（推荐方式 A）
# 打开项目，按 F5 开始调试

# 或本地运行应用（如使用方式 B）
gow run admin
```

**Windows PowerShell 脚本特性：**

- ✅ 完整的参数支持（-AppRoot, -ComposeFile）
- ✅ 自动检测 Docker Compose 版本
- ✅ 彩色化的日志输出
- ✅ 完整的帮助系统（Get-Help）

---

## 📊 脚本功能对比

| 脚本             | 应用服务     | 依赖服务 | 用途   |
|----------------|----------|------|------|
| full_deploy.sh | ✅ 启动     | ✅ 启动 | 完整部署 |
| libs_only.sh   | ❌ 不启动    | ✅ 启动 | 本地开发 |
| pm2_service.sh | ✅ PM2 管理 | -    | 生产服务 |

---

## ✅ 常见问题

### Q: 应该用哪个环境脚本？

**A:**

- **生产环境** → `install_unix_prod.sh`
- **开发环境** → `install_unix_dev.sh`
- **Windows** → `install_windows_dev.ps1`

### Q: full_deploy.sh 和 libs_only.sh 有什么区别？

**A:**

- `full_deploy.sh` - 启动应用 + 依赖（适合一键启动）
- `libs_only.sh` - 仅启动依赖（适合本地开发）

### Q: 脚本失败了怎么办？

**A:** 脚本具有幂等性，直接重新运行即可。多次运行不会出现问题。

### Q: 如何自定义脚本行为？

**A:** 通过环境变量控制：

```bash
APP_ROOT=/custom/path bash scripts/docker/full_deploy.sh
COMPOSE_FILE=custom.yaml bash scripts/docker/libs_only.sh

# Unix 环境脚本可选初始化 hosts（默认 false）
AUTO_INIT_HOSTS=true HOSTS_SERVICES="postgres mysql redis" bash scripts/env/install_unix_dev.sh
```

Unix hosts 初始化相关变量：

- `AUTO_INIT_HOSTS` - 是否启用 hosts 初始化（默认：`false`）
- `HOSTS_IP` - hosts 记录 IP（默认：`127.0.0.1`）
- `HOSTS_DOMAIN_SUFFIX` - 域名后缀（默认：`.local`）
- `HOSTS_SERVICES` - 服务名列表，空格分隔（默认：`postgres mysql redis`）

### Q: 如何添加新的 Go 工具或插件？

**A:** 编辑对应的 `install_unix_dev.sh` 或 `install_windows_dev.ps1` 脚本中的工具列表。

---

## 🚀 快速开始

### 第一次使用（Linux/macOS）

```bash
# 1. 准备环境
./scripts/env/install_unix_dev.sh

# 2. 启动依赖
./scripts/docker/libs_only.sh

# 3. 本地开发运行
gow run admin

# 4. 查看日志
docker logs -f postgres  # 查看数据库日志
```

### 第一次使用（Windows）

```powershell
# 1. 以管理员打开 PowerShell
# 2. 准备环境
.\scripts\env\install_windows_dev.ps1

# 3. 启动依赖
.\scripts\docker\libs_only.ps1

# 4. 本地开发
gow run admin
```

---

## 📚 详细文档

| 文档                       | 说明             | 位置           |
|--------------------------|----------------|--------------|
| PREPARE_UNIX.md          | Unix 环境准备详解    | scripts/     |
| DOCKER_COMPOSE_NAMING.md | Docker 脚本说明    | scripts/     |
| GO_ENVIRONMENT_GUIDE.md  | Go 环境配置详解      | scripts/env/ |
| QUICK_START.md           | Windows 脚本快速参考 | scripts/env/ |

---

## 🔍 脚本验证

检查脚本是否正确安装：

```bash
# 检查环境
go version
docker --version
pm2 -v

# 检查 Go 插件
protoc-gen-go --version

# 列出 PM2 应用
pm2 list
```

---

## 📞 技术支持

遇到问题？

1. **查看脚本注释** - 每个脚本头部都有详细说明
2. **查看相关文档** - 参考"详细文档"部分
3. **检查环境变量** - 确保环境变量设置正确
4. **查看日志** - 脚本有详细的日志输出
5. **重新运行** - 脚本具有幂等性，多次运行安全

---
