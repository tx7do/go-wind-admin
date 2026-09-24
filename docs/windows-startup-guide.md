# GoWind Admin Windows 本地开发启动指南

本文档介绍如何在 Windows 系统上从零搭建开发环境并启动 GoWind Admin 项目。

基于实际在 Windows 11 上完整跑通后编写，包含所有已知的坑和解决方案。

> **阅读指引**：循序渐进的主路径见教程层
> [教程 02 · 从零跑起来](./tutorial/02-get-it-running.md)（浓缩版主链路）；
> 本指南是**逐步全流程与 FAQ 的权威参考**——遇到教程未覆盖的报错时查本文档「常见问题」节。
> 文档体系总入口见 [docs/README.md](./README.md)。

---

## 前置要求

| 软件 | 版本要求 | 用途 |
|------|---------|------|
| Go | 1.26+（`backend/go.mod` 当前钉 `go 1.26.4`，低于该版本无法构建） | 后端编译运行 |
| Docker Desktop | 最新版 | 运行 PostgreSQL、Redis、MinIO |
| Node.js | 以各前端 `package.json` 的 `engines` 为准：react **未声明** engines，vue-element `^20.19.0 \|\| >=22.12.0`，vue-vben `>=20.10.0`（另要求 pnpm `>=9.12.0`）——三端同时满足即取 vue-element 的约束 `^20.19.0 \|\| >=22.12.0`（21.x 不在内） | 前端编译运行 |
| pnpm | 仅 vue-vben 钉定（其 `packageManager` 字段为 `pnpm@11.18.0`，corepack 自动路由）；react / vue-element 未钉定版本 | Vue 前端包管理 |
| Git | 最新版 | 版本控制 |

### 一键安装开发环境

项目提供了自动化脚本，以管理员身份打开 PowerShell 执行：

```powershell
# 在项目根目录执行
powershell -ExecutionPolicy Bypass -File backend\scripts\env\install_windows_dev.ps1
```

在 `backend/` 目录下执行 `.\scripts\env\install_windows_dev.ps1` 等价——脚本用 `$MyInvocation` 解析自身目录来加载
`scripts/env/lib/*.ps1`（`install_windows_dev.ps1:13`），**工作目录不影响行为**。

该脚本通过 Scoop 自动安装 Go、Docker Desktop、Node.js、Git、Make 等，并安装所有 Go 代码生成工具（buf、ent、gow 等）。

> **注意**：非管理员运行也可以，但**需要管理员权限的两步会被跳过**：Docker 服务自动启动配置，
> 以及 hosts 写入（`install_windows_dev.ps1:44-45` 的提示原文即
> "Docker auto-start and hosts configuration will be skipped"，52-54 行另有一条跳过告警）。

### 手动启用 corepack

一键安装脚本不覆盖 pnpm。启用 corepack（Node.js 自带）即可：

```bash
corepack enable
```

vue-vben 的 `package.json` 中 `packageManager` 字段钉死了 `pnpm@11.18.0`——corepack
在该 monorepo 目录下会自动使用该版本，**无需也不应手动 `corepack prepare`**。
react / vue-element 未钉定版本，使用 corepack 默认提供的 pnpm。

> **不要用** `npm install -g pnpm`：绕过 corepack 的版本路由，可能与钉定版本不一致导致安装失败。

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

# 首次搭建：安装代码生成工具（buf、ent、gow 等）
# make init = make plugin + make cli，cli 目标里已含 gow（backend/Makefile:48），无需单独再装
make init

# （可选）只想装 gow 时用；已跑过 make init 则这条是重复
# go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest

# 生成全部代码（Ent ORM + Protobuf API）
gow ent && gow api

# OpenAPI 文档（gow 未覆盖，走 make）
make openapi
```

如果没有安装 `make`，生成代码可全部分步用 gow / buf 手动执行：

```bash
# 1. 生成 ENT ORM 代码（在 backend/ 下）
gow ent

# 2. 生成 Protobuf Go 代码（在 backend/ 下）
gow api

# 3. 生成 OpenAPI 文档
cd api
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

## 第六步：启动前端

前端统一存放于 `frontend/admin` 目录，有三个版本可选：

| 版本 | 目录 | 启动命令 | 本地端口 |
|------|------|---------|--------|
| Vue Vben | `frontend/admin/vue-vben` | `pnpm dev:antd` | 5666 |
| Vue Element | `frontend/admin/vue-element` | `pnpm dev` | 5777 |
| React | `frontend/admin/react` | `pnpm dev` | 5888 |

以 Vue Vben 版本为例：

```bash
cd frontend/admin/vue-vben

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev:antd
```

启动成功后访问 http://localhost:5666

默认登录账号：`admin`，密码：`Abcd@1234`

### 首次 `pnpm install` 后 `pnpm dev` 报错 `turbo-run` 找不到？

这是 Windows 上的已知问题。`scripts/turbo-run/bin/turbo-run.mjs` 和 `scripts/vsh/bin/vsh.mjs` 这两个 bin shim
可能缺失——它们各自是 `scripts/turbo-run/package.json` / `scripts/vsh/package.json` 里 `bin` 字段指向的入口，
而仓库根 `.gitignore` 的 `bin/` 规则（第 29 行）把这两个目录整个忽略了，**所以它们不在 git 索引里，
新克隆的仓库天然缺这两个文件**（`git ls-files frontend/admin/vue-vben/scripts` 查不到 `bin/*.mjs` 即为佐证）。

在 **`frontend/admin/vue-vben` 目录下**手动创建（内容与实际磁盘上的 shim 一致：shebang + 空行 + 动态 import）：

```bash
cd frontend/admin/vue-vben   # 已在该目录则跳过

# 创建 turbo-run shim
mkdir -p scripts/turbo-run/bin
printf '#!/usr/bin/env node\n\nimport('"'"'../dist/index.mjs'"'"');\n' > scripts/turbo-run/bin/turbo-run.mjs

# 创建 vsh shim
mkdir -p scripts/vsh/bin
printf '#!/usr/bin/env node\n\nimport('"'"'../dist/index.mjs'"'"');\n' > scripts/vsh/bin/vsh.mjs

# 重新安装以链接 bin
pnpm install
```

> `../dist/index.mjs` 是这两个包自己的产物，同样不进 git（`frontend/admin/vue-vben/.gitignore:3` 的 `dist` 规则），
> 由根 `package.json` 的 `postinstall`（`pnpm -r run stub --if-present`，各包 `stub` = `pnpm unbuild --stub`）生成——
> 所以顺序是**先补 `bin/*.mjs`，再 `pnpm install`**，install 会一并补上 dist 与 bin 链接。
> 只想立刻起 dev server 的话，根目录的 `dev:antd`（= `pnpm -F @vben/web-antd run dev`，见 `package.json:21`）
> 不经过 turbo-run / vsh；根目录的 `dev`（`:20` = `turbo-run dev`）才会用到它们。
> PowerShell 下不要用上面的 `printf`/引号转义写法，改用 `Set-Content` 写同样三行内容。

之后再执行 `pnpm dev` 即可。

---

## 完整服务端口一览

| 服务 | URL | 说明 |
|------|-----|------|
| Vue Vben 前端 | http://localhost:5666 | Vite 开发服务器（Vben 版） |
| Vue Element 前端 | http://localhost:5777 | Vite 开发服务器（Element 版） |
| React 前端 | http://localhost:5888 | Vite 开发服务器（React 版） |
| 后端 REST API | http://localhost:7788 | HTTP 接口 |
| 后端 SSE | http://localhost:7789/events | 服务端推送事件 |
| MinIO Console | http://localhost:9001 | 对象存储管理界面 |
| PostgreSQL | localhost:5432 | 数据库 |
| Redis | localhost:6379 | 缓存 |

> 上表为本地开发端口。生产环境中的 SSE 反向代理网关（独立 nginx 容器，端口 `8013`）与 JWT 签名密钥的环境变量注入属于部署配置，不在本地开发范围，详见 [backend_deploy.md](./backend_deploy.md)。

---

## 日常开发工作流

```
终端 1：cd backend && docker compose -f docker-compose.libs.yaml up -d
终端 2：cd backend/app/admin/service && go run ./cmd/server -c ./configs
终端 3：cd frontend/admin/vue-vben && pnpm dev:antd
```

代码生成命令速查（在 `backend/` 目录下执行）：

| 场景 | 命令 |
|------|------|
| 修改了 `.proto` 文件 | `gow api`，如需文档再跑 `make openapi` |
| 修改了 ENT Schema | `gow ent` |
| 全量重新生成 | `gow ent && gow api && make openapi` |
| 生成前端 TypeScript 客户端 | `make ts` |

---

## 常见问题

### Q: Docker 拉取镜像报 `TLS handshake timeout`

国内网络访问 Docker Hub 不稳定，需要配置镜像加速器（见第二步说明）。配置后必须重启 Docker Desktop 才能生效。

### Q: Docker 容器启动后后端连接数据库失败

确认配置文件中的主机名已改为 `localhost`（见第三步）。`postgres` / `redis` / `minio` 是 Docker Compose 内部网络的服务名，宿主机上不能直接使用。

### Q: `make gen` / `make init` 报错找不到 `buf` / `ent`

确保 Go 的 bin 目录已加入系统 PATH（Windows 上默认是 `%USERPROFILE%\go\bin`，PowerShell 与 cmd 用这个写法）。
**下面这段是 Git Bash 语法**（PowerShell 里 `grep` / `$()` 行为不同）：

```bash
# 取生效的 GOPATH（`echo $GOPATH` 可能为空——Go 有默认值但不一定是环境变量）
go env GOPATH
# 应输出类似 C:\Users\你的用户名\go

ls "$(go env GOPATH)/bin/"
# 应能看到 buf.exe、gow.exe 等
```

如果为空，手动安装：

```bash
go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest
go install github.com/bufbuild/buf/cmd/buf@latest
go install entgo.io/ent/cmd/ent@latest
```

### Q: `golangci-lint` 安装失败

v1 版本的上游依赖仓库已不可用，使用 v2：

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

### Q: 前端 `pnpm install` 报版本不匹配

vue-vben 钉定了 `pnpm@11.18.0`（`packageManager` 字段）。确保 corepack 已启用，
由 corepack 在该目录自动路由到钉定版本，不要手动 prepare、也不要 `npm i -g pnpm`：

```bash
corepack enable
```

