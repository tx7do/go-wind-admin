# 02 · 从零跑起来：本地开发环境

> 前置：读完[第 1 章](./01-architecture-overview.md)。读完你会得到：一个可调试的本地环境——中间件跑在 Docker，后端与前端跑在本地，浏览器登录进管理后台。

本文是**主线路径**的浓缩版。逐项细节、坑的完整解法在权威参考
[windows-startup-guide.md](../windows-startup-guide.md)（Windows）与两篇环境准备文档（见文末深读）。

## 1. 拓扑：依赖模式

本地开发用**依赖模式（libs_only）**：只有三方中间件（PostgreSQL / Redis / MinIO）跑在 Docker 里，
后端服务与前端都在本地 IDE/终端运行调试。这与生产部署（第 9 章）是两回事。

## 2. 工具前置

| 工具 | 版本 | 用途 |
|---|---|---|
| Go | 以 `backend/go.mod` 为准（当前钉 `1.26.4`） | 后端 |
| Docker Desktop | 最新 | 中间件容器 |
| Node.js | 以各前端 `package.json` 的 `engines` 为准（react 未声明；交集为 `^20.19.0 \|\| >=22.12.0`，即 21.x 不满足） | 前端 |
| pnpm | 仅 vue-vben 由其 `packageManager` 钉死 `pnpm@11.18.0`；react / vue-element 未钉定 | 前端包管理 |

一键安装脚本（含 Go 代码生成工具链，**在项目根目录执行**）：

```powershell
# Windows 管理员 PowerShell
powershell -ExecutionPolicy Bypass -File backend\scripts\env\install_windows_dev.ps1
```

```bash
# Linux / macOS
backend/scripts/env/install_unix_dev.sh
```

启用 corepack（Node 自带）。vue-vben 的 `packageManager` 字段钉死了 pnpm 版本，
corepack 会在该目录自动路由到钉定版本，无需手动 prepare；react / vue-element 未钉定，
使用 corepack 默认提供的 pnpm：

```bash
corepack enable
```

## 3. 启动中间件（Docker）

Windows 上先确认 Docker Desktop 处于 **Linux 容器模式**（`docker info | grep OSType` 应为 `linux`，
否则见 windows-startup-guide 第一步）。

```bash
cd backend
docker compose -f docker-compose.libs.yaml up -d
docker compose -f docker-compose.libs.yaml ps   # 应看到 postgres / redis / minio 三个 Running
```

## 4. 改后端配置：容器服务名 → localhost

`docker-compose.libs.yaml` 里的服务名（`postgres`/`redis`/`minio`）只在容器网络内有效。
后端在本地跑，所以 `backend/app/admin/service/configs/` 下三个文件的地址**必须改成 `localhost`**：

| 文件 | 改什么 |
|---|---|
| `data.yaml` | `database.source`、`redis.addr` 的主机名 |
| `server.yaml` | `asynq.uri` 的主机名 |
| `oss.yaml` | `minio.endpoint` / `upload_host` / `download_host` 的主机名 |

> 这些改动只用于本地，别提交。建议 `git stash` 或本地 ignore。

## 5. 生成代码 & 启动后端

```bash
cd backend
go mod download
gow ent admin && gow api  # ent ORM + proto Go 代码（首次必跑；gow 的安装见 windows-startup-guide）
```

```bash
# 任选其一：
cd backend && gow run admin            # gow CLI（推荐）
cd backend/app/admin/service && go run ./cmd/server -c ./configs
```

启动成功的标志（日志）：

```
[HTTP] server listening on: [::]:7788
[sse] server listening on: [::]:7789
```

验证：<http://localhost:7788/docs/openapi.yaml> 返回 YAML 即正常。
首次启动会播种系统默认数据（admin 用户、角色、菜单、权限、语言、Api 表——全部由 Go 侧
`count == 0` 守卫播种，**没有 SQL 种子脚本**）。

> 首次启动的一条 ERROR `failed to list apis by ids: ... BAD_REQUEST` 是时序正常现象，可忽略。

## 6. 启动前端（三选一，或多个）

| 端 | 目录 | 启动 | 端口 |
|---|---|---|---|
| Vue Vben | `frontend/admin/vue-vben` | `pnpm install && pnpm dev:antd` | 5666 |
| Vue Element | `frontend/admin/vue-element` | `pnpm install && pnpm dev` | 5777 |
| React | `frontend/admin/react` | `pnpm install && pnpm dev` | 5888 |

端口写在 `.env.development` 里但键名三端各异（react `VITE_SERVER_PORT` / vue-element `VITE_APP_PORT`
/ vue-vben `VITE_PORT`，见第 01 章）；被占用时 vite 会自动顺延一个端口，**启动日志里有实际端口**，
别按记忆硬连。

> **只打算长期维护其中一端？** 三端互不依赖、可以整目录删掉另外两端，但有三处会"悄悄回流"的耦合点
> （`make ts` 重建已删目录、CORS 白名单、菜单表没有"端"这个维度）——动手前读
> [只保留一个前端：裁剪步骤与代价](../adopt-one-frontend.md)。

登录账号：`admin` / `Abcd@1234`（首启播种的默认口令，来源
`backend/pkg/constants/default_data.go` 的 `DefaultUserPassword`）。

## 7. 本地端口一览

| 服务 | 地址 | 说明 |
|---|---|---|
| 后端 REST API | <http://localhost:7788> | 所有管理接口 + Swagger（`/docs`） |
| 后端 SSE | <http://localhost:7789/events> | 服务端推送 |
| React / Vue Element / Vue Vben | 5888 / 5777 / 5666 | Vite dev server |
| MinIO Console | <http://localhost:9001> | 对象存储管理界面（root / \*Abcd123456） |
| PostgreSQL / Redis | 5432 / 6379 | 仅本机后端使用 |

## 8. 常见坑速查（细节见 windows-startup-guide「常见问题」）

| 症状 | 方向 |
|---|---|
| Docker 拉镜像 `TLS handshake timeout` | 配置 Docker 镜像加速并重启 Docker Desktop |
| 后端连不上 DB | 配置里主机名没改 `localhost`（第 4 节） |
| `gow`/`buf` 命令找不到 | `%USERPROFILE%\go\bin` 不在 PATH；手动 `go install`（见参考文档命令） |
| vben `pnpm dev` 报 `turbo-run` 找不到 | Windows 已知问题，按参考文档补两个 bin shim |
| 页面白屏 / 路由塌空（vue-element） | 已修复的历史问题：真因是 `<transition mode="out-in">` 与 vue-router 5 懒加载路由的竞态（**不是** vite 依赖优化，dev/prod 都能复现）。别把 `mode="out-in"` 改回去；判回归先重启 dev server，细节见该端 AGENTS.md「白屏（router-view 塌空）真因与处置」 |
| 前端请求全部 401/无法登录 | 确认后端 7788 在跑；验证码答案可在 Redis 按 `gowind:captcha:<captchaId>` 读（自动化调试用） |

## 9. 日常开发工作流

环境就绪后，每天的循环只有三条命令：

```
终端 1：cd backend && docker compose -f docker-compose.libs.yaml up -d
终端 2：cd backend && gow run admin
终端 3：cd frontend/admin/<选择端> && pnpm dev[:antd]
```

改了 proto → `gow api`（前端 TS：后端根目录 `make ts`）；改了 ent schema → `gow ent admin`。
完整命令速查见 windows-startup-guide「日常开发工作流」。

## 深读

- [windows-startup-guide.md](../windows-startup-guide.md) —— Windows 全流程逐步版 + 完整 FAQ（本文的权威来源）
- [backend_development_environment_preparation.md](../backend_development_environment_preparation.md) —— 后端工具链与 Go 模块代理
- [frontend_development_environment_preparation.md](../frontend_development_environment_preparation.md) —— 前端工具链与 npm 镜像
- [backend_deploy.md](../backend_deploy.md) —— 依赖模式 vs 完整模式的定义（第 9 章展开生产侧）

下一步：[03 · 代码生成链路](./03-codegen-chain.md)。
