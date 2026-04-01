# GoWind Admin Windows 本地开发启动指南

本文档介绍如何在 Windows 系统上从零搭建开发环境并启动 GoWind Admin 项目。

基于实际在 Windows 11 上完整跑通后编写，包含所有已知的坑和解决方案。

---

## 前置要求

| 软件 | 版本要求 | 用途 |
|------|---------|------|
| Go | 1.25.7+ | 后端编译运行 |
| Docker Desktop | 最新版 | 运行 PostgreSQL、Redis、MinIO |
| Node.js | >= 20.10.0 | 前端编译运行 |
| pnpm | >= 10.19.0 | Vue 前端包管理（`packageManager` 锁定版本） |
| Git | 最新版 | 版本控制 |

### 一键安装开发环境

项目提供了自动化脚本，以管理员身份打开 PowerShell 执行：

```powershell
# 在项目根目录执行
powershell -ExecutionPolicy Bypass -File backend\scripts\env\install_windows_dev.ps1
```

该脚本通过 Scoop 自动安装 Go、Docker Desktop、Node.js、Git、Make 等，并安装所有 Go 代码生成工具（buf、ent、wire 插件等）。

> **注意**：非管理员运行也可以，但 Docker 服务自动启动配置会被跳过。

### 手动安装 pnpm

脚本不会安装 pnpm，需要手动安装。推荐使用 corepack（Node.js 自带）：

```bash
corepack enable
corepack prepare pnpm@10.19.0 --activate
```

验证版本：

```bash
pnpm -v
# 应输出 10.19.0
```

> **不要用** `npm install -g pnpm`，版本可能不匹配。项目 `package.json` 中 `packageManager` 字段锁定了 `pnpm@10.19.0`，版本不对会导致安装失败。

---

## 第一步：确认 Docker Desktop 处于 Linux 容器模式

这一步**非常重要**。Docker Desktop 默认可能处于 Windows 容器模式，项目依赖的 bitnami 镜像只能在 Linux 容器模式下运行。

检查当前模式：

```bash
docker info | grep OSType
```

- 如果输出 `OSType: linux`，说明已经是 Linux 模式，跳过此步
- 如果输出 `OSType: windows`，需要切换：

```powershell
# PowerShell 中执行
& "C:\Program Files\Docker\Docker\DockerCli.exe" -SwitchLinuxEngine
```

或者在系统托盘右键 Docker Desktop 图标，选择 **Switch to Linux containers**。

切换后等待约 15 秒，再次验证：

```bash
docker info | grep OSType
# 应输出 OSType: linux
```

---

## 第二步：启动依赖服务（PostgreSQL、Redis、MinIO）

```bash
cd backend
docker compose -f docker-compose.libs.yaml up -d
```

> **国内网络问题**：如果拉取镜像超时（TLS handshake timeout），需要配置 Docker 镜像加速。
>
> 编辑 `~/.docker/daemon.json`（Linux 容器模式下），添加：
>
> ```json
> {
>   "registry-mirrors": [
>     "https://docker.1ms.run",
>     "https://docker.xuanyuan.me"
>   ]
> }
> ```
>
> 保存后重启 Docker Desktop。

### 验证服务状态

```bash
docker compose -f docker-compose.libs.yaml ps
```

应看到三个服务在运行：

| 服务 | 端口 | 账号 | 密码 |
|------|------|------|------|
| PostgreSQL | 5432 | postgres | *Abcd123456 |
| Redis | 6379 | - | *Abcd123456 |
| MinIO | 9000 (API) / 9001 (Console) | root | *Abcd123456 |

---

## 第三步：修改后端配置（本地开发必须）

默认配置中的主机名（`postgres`、`redis`、`minio`）是 Docker Compose 的容器服务名，仅在容器网络内部有效。本地运行后端时**必须改为 `localhost`**。

需要修改 `backend/app/admin/service/configs/` 下的三个文件：

### data.yaml

```yaml
data:
  database:
    source: "host=localhost port=5432 user=postgres password=*Abcd123456 dbname=gwa sslmode=disable"
    #          ^^^^^^^^^ 原值为 postgres

  redis:
    addr: "localhost:6379"
    #      ^^^^^^^^^ 原值为 redis
```

### server.yaml

```yaml
server:
  asynq:
    uri: "redis://:*Abcd123456@localhost:6379/1"
    #                           ^^^^^^^^^ 原值为 redis
```

### oss.yaml

```yaml
oss:
  minio:
    endpoint: "localhost:9000"
    upload_host: "localhost:9000"
    download_host: "localhost:9000"
    #               ^^^^^^^^^ 原值均为 minio
```

> **提示**：这些改动仅用于本地开发，建议不要提交到 Git。可以用 `git stash` 暂存。

---

## 第四步：安装后端依赖 & 代码生成

```bash
cd backend

# 下载 Go 模块依赖
go mod download

# 首次搭建：安装代码生成工具（buf、ent、wire 插件等）
make init

# 生成全部代码（ENT ORM + Wire DI + Protobuf API + OpenAPI）
make gen
```

如果没有安装 `make`，可以分步手动执行：

```bash
# 1. 生成 ENT ORM 代码
cd app/admin/service
ent generate --feature privacy --feature entql --feature sql/modifier --feature sql/upsert --feature sql/lock ./internal/data/ent/schema

# 2. 生成 Wire 依赖注入代码
go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server

# 3. 生成 Protobuf Go 代码
cd ../../../api
buf generate

# 4. 生成 OpenAPI 文档
buf generate --template buf.admin.openapi.gen.yaml
```

> 没有 `make`？通过 Scoop 安装：`scoop install make`

---

## 第五步：启动后端服务

### 方式一：命令行

```bash
cd backend/app/admin/service
go run ./cmd/server -c ./configs
```

### 方式二：Make

```bash
cd backend/app/admin/service
make run
```

### 方式三：GoLand IDE

1. 打开 `backend/app/admin/service/cmd/server/main.go`
2. 点击 `main` 函数旁的运行按钮
3. 在 Run Configuration 中设置 Program arguments：`-c ../../configs`

### 验证后端启动成功

看到以下日志即为成功：

```
[HTTP] server listening on: [::]:7788
[sse] server listening on: [::]:7789
[asynq] asynq server started
```

也可以访问 http://localhost:7788/docs/openapi.yaml ，返回 YAML 内容即为正常。

> 首次启动时会看到一条 ERROR 日志 `failed to list apis by ids: ... BAD_REQUEST`，这是因为数据库还没有导入初始数据，属于正常现象。

---

## 第六步：启动 Vue 前端

```bash
cd frontend/admin/vue

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev
```

启动成功后访问 http://localhost:5666

默认登录账号：`admin`，密码：`admin`

### 首次 `pnpm install` 后 `pnpm dev` 报错 `turbo-run` 找不到？

这是 Windows 上的已知问题。`scripts/turbo-run/bin/turbo-run.mjs` 和 `scripts/vsh/bin/vsh.mjs` 这两个 bin shim 文件可能缺失。

手动创建：

```bash
# 创建 turbo-run shim
mkdir -p scripts/turbo-run/bin
echo '#!/usr/bin/env node
import "../dist/index.mjs";' > scripts/turbo-run/bin/turbo-run.mjs

# 创建 vsh shim
mkdir -p scripts/vsh/bin
echo '#!/usr/bin/env node
import "../dist/index.mjs";' > scripts/vsh/bin/vsh.mjs

# 重新安装以链接 bin
pnpm install
```

之后再执行 `pnpm dev` 即可。

---

## 完整服务端口一览

| 服务 | URL | 说明 |
|------|-----|------|
| Vue 前端 | http://localhost:5666 | Vite 开发服务器 |
| 后端 REST API | http://localhost:7788 | HTTP 接口 |
| 后端 SSE | http://localhost:7789/events | 服务端推送事件 |
| MinIO Console | http://localhost:9001 | 对象存储管理界面 |
| PostgreSQL | localhost:5432 | 数据库 |
| Redis | localhost:6379 | 缓存 |

---

## 日常开发工作流

```
终端 1：cd backend && docker compose -f docker-compose.libs.yaml up -d
终端 2：cd backend/app/admin/service && go run ./cmd/server -c ./configs
终端 3：cd frontend/admin/vue && pnpm dev
```

代码生成命令速查（在 `backend/` 目录下执行）：

| 场景 | 命令 |
|------|------|
| 修改了 `.proto` 文件 | `make api && make openapi` |
| 修改了 ENT Schema | `make ent` |
| 修改了 Wire 依赖注入 | `make wire` |
| 全量重新生成 | `make gen` |
| 生成前端 TypeScript 客户端 | `make ts` |

---

## 常见问题

### Q: Docker 拉取镜像报 `TLS handshake timeout`

国内网络访问 Docker Hub 不稳定，需要配置镜像加速器（见第二步说明）。配置后必须重启 Docker Desktop 才能生效。

### Q: Docker 容器启动后后端连接数据库失败

确认配置文件中的主机名已改为 `localhost`（见第三步）。`postgres` / `redis` / `minio` 是 Docker Compose 内部网络的服务名，宿主机上不能直接使用。

### Q: `make gen` / `make init` 报错找不到 `buf` / `ent` / `wire`

确保 `%USERPROFILE%\go\bin` 已加入系统 PATH。验证：

```bash
echo $GOPATH
# 应输出类似 C:\Users\你的用户名\go

ls $GOPATH/bin/
# 应能看到 buf.exe、ent.exe 等
```

如果为空，手动安装：

```bash
go install github.com/bufbuild/buf/cmd/buf@latest
go install entgo.io/ent/cmd/ent@latest
go install github.com/google/wire/cmd/wire@latest
```

### Q: `golangci-lint` 安装失败

v1 版本的上游依赖仓库已不可用，使用 v2：

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

### Q: 前端 `pnpm install` 报版本不匹配

项目锁定了 `pnpm@10.19.0`，通过 corepack 安装精确版本：

```bash
corepack enable
corepack prepare pnpm@10.19.0 --activate
```

### Q: 后端启动报 `wire_gen.go` 不存在

需要先执行代码生成：

```bash
cd backend/app/admin/service
go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server
```

或在 `backend/` 目录执行 `make wire`。
