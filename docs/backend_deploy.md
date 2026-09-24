# 后端项目部署

> **执行目录约定**：本文所有相对路径命令（`./scripts/...`、`make ...`、`docker compose ...`、`gow ...`）
> 一律在 **`backend/` 目录**下执行——仓库根目录没有 Makefile，脚本也不在根目录。

- 所有的Docker配置文件都在`backend`目录下。
- 所有的部署脚本都在`backend/scripts`目录下。

> 详细的脚本使用说明请参考：[scripts/README.md](../backend/scripts/README.md)

Shell脚本需要赋予执行权限（`scripts/` 下含多级子目录，`**` 在多数 shell 里不会递归，用 `find`）：

```bash
cd backend
find ./scripts -name '*.sh' -exec chmod +x {} +
```

## 初始化操作系统环境

在我们拿到服务器后，首先要做的就是初始化操作系统环境。我们需要安装一些必要的工具和软件包。

> 推荐使用项目提供的一键安装脚本，详见 [环境准备脚本](../backend/scripts/env/)

### Linux / macOS

**生产环境：**

```bash
./scripts/env/install_unix_prod.sh
```

**开发环境：**

```bash
./scripts/env/install_unix_dev.sh
```

### Windows（PowerShell 管理员）

```powershell
.\scripts\env\install_windows_dev.ps1
```

> 脚本用 `$MyInvocation` 解析自身所在目录来加载 `scripts/env/lib/*.ps1`（见 `install_windows_dev.ps1:13`），
> 因此**在哪个目录调用都一样**——从 `backend/` 用上面的相对路径，或从仓库根目录用
> `powershell -ExecutionPolicy Bypass -File backend\scripts\env\install_windows_dev.ps1`
> （[windows-startup-guide.md](./windows-startup-guide.md) 的方式）都成立。

## Docker 两种部署模式

部署项目有两种方法：

1. **完整模式**：三方中间件和微服务都运行在Docker之下；
2. **依赖模式（推荐开发）**：三方中间件运行在Docker下，微服务在本地IDE运行调试。

### 1. 完整模式（三方中间件 + 微服务都在 Docker 下）

**Linux / macOS：**

```bash
./scripts/docker/full_deploy.sh
```

**Windows (PowerShell)：**

```powershell
.\scripts\docker\full_deploy.ps1
```

### 2. 依赖模式（仅启动三方中间件，微服务本地运行）

**Linux / macOS：**

```bash
./scripts/docker/libs_only.sh
```

**Windows (PowerShell)：**

```powershell
.\scripts\docker\libs_only.ps1
```

然后本地运行后端服务：

```bash
gow run admin
```

### 3. PM2 进程管理（生产环境物理机部署）

```bash
./scripts/deploy/pm2_service.sh
```

## SSE 反向代理网关（可选）

SSE 推送链路在生产拓扑中由一个独立的 nginx 反向代理网关承载，把 `/events` 透传至后端 admin-service 的 SSE transport（默认 `7789`）。该网关不随 docker-compose 启动，需单独构建与运行。

构建镜像并运行容器：

```bash
bash scripts/deploy/sse/build-local-docker-image.sh

# 先查出网络的真名（见下），再用它启动
docker network ls --filter name=app-tier
docker run -d -p 8013:8080 --network <上面查到的网络名> --name sse-gateway-local sse-gateway-local
```

说明：

- 网关必须与 admin-service 共处同一 Docker 网络才能访问后端。
- **`--network app-tier` 不能照抄**：`backend/docker-compose.yaml:1-3` 只声明了
  `networks: app-tier: {driver: bridge}`，没写 `name:`，所以 Compose 建出的真实网络名带项目名前缀
  （默认项目名取 compose 文件所在目录名，即形如 `<目录名>_app-tier`；用了 `-p` / `COMPOSE_PROJECT_NAME`
  则是另一个前缀）。**具体名字无法凭文档断定，用 `docker network ls --filter name=app-tier` 现场查**。
  想钉死成 `app-tier`，在 compose 的该网络下加一行 `name: app-tier` 后 `docker compose up -d` 重建。
  同样的照抄问题也存在于 `scripts/deploy/sse/build-local-docker-image.sh` 结尾打印的示例命令里
  （脚本只是 echo 一个 sample，不代为执行），以本节为准。
- 配置文件 `scripts/deploy/sse/nginx.conf` 已针对 SSE 关闭 gzip 与一切代理缓冲（`proxy_buffering off` / `proxy_cache off` / `proxy_request_buffering off`），并使用 HTTP/1.1 与长连接保活；这些是 SSE 实时性的必要配置，勿随意回退。
- TLS 终止由外层负载均衡负责，与前端静态站点部署一致，网关自身仅监听 HTTP。
- 独立部署时如后端地址变更，需修改 `nginx.conf` 中的 `upstream`。

## 生产环境密钥配置

后端 JWT 签名密钥支持通过环境变量注入，优先级高于 `configs/auth.yaml`：

| 环境变量 | 用途 |
|---------|------|
| `GWA_AUTH_JWT_PRIVATE_KEY` | 非对称算法（RS256 / ES256 / Ed25519 等）的 PEM 私钥 |
| `GWA_AUTH_JWT_PUBLIC_KEY` | 非对称算法的 PEM 公钥 |
| `GWA_AUTH_JWT_KEY` | 对称算法（HS256 等）的共享密钥 |

`configs/auth.yaml` 内置的是开发示例密钥，生产环境必须通过上述环境变量替换为自有密钥，否则启动时会告警。生成命令：

```bash
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out jwt_private_key.pem
openssl pkey -in jwt_private_key.pem -pubout -out jwt_public_key.pem
```

## 备份与恢复

平台有两套互补的备份机制，**互不替代**：

| 机制 | 形态 | 说明 |
|---|---|---|
| [`scripts/backup/pg_backup.sh`](../backend/scripts/backup/pg_backup.sh) | pg_dump 物理备份（Docker 容器 / 本地直连双模式），定时全量、默认保留 30 份自动轮换，附恢复操作文档 | 部署侧脚本，cron/计划任务挂载 |
| asynq `backup` 任务 | 应用级逻辑备份：核心表导出 JSON → gzip → 上传 MinIO `backups` 桶（日期分层对象） | 调度与机制见 [task_system.md](./task_system.md) 第 5.3 节；当前无自动恢复/演练工具链，桶内对象无生命周期清理（该文档第 10 节） |

**恢复演练**：没有验证过的备份等于没有。任一机制接入后，用一次真实恢复演练闭环
（pg_dump 走其附带恢复文档；JSON 备份目前需手工下载反序列化）。

## 本地开发配置 hosts

如果使用完整模式部署后需要从宿主机访问服务，需修改`hosts`文件（需要管理员权限）：

- Linux：`/etc/hosts`
- MacOS：`/private/etc/hosts`
- Windows：`C:\Windows\System32\drivers\etc\hosts`

增加以下内容——只对应 `docker-compose.yaml` 里**实际存在**的服务（postgres / redis / minio）：

```ini
127.0.0.1 postgres
127.0.0.1 redis
127.0.0.1 minio
```

> **沿革**：此处早期还列过 `consul` 与 `jaeger`，本仓现状没有这两种部署——`docker-compose.yaml` 里
> jaeger 整段（86 行起）是注释掉的，配置与代码里 `grep -i consul` 零命中，注册中心也未接入。
> 若二次开发自行引入 Consul，再按当时的形态补 hosts 与 `registry` 配置（历史坑：`consul` 作主机名会 502，
> 地址要写 `localhost:8500`）。
>
> **推荐做法**：本地开发使用依赖模式（libs_only），配置文件中直接使用`localhost`即可，无需修改 hosts
> （见 [windows-startup-guide.md](./windows-startup-guide.md) 第三步：改的就是 `localhost`）。
>
> **与一键脚本的差异**：`scripts/env/install_windows_dev.ps1` 以管理员运行时写的是**带 `.local` 后缀**的
> 域名（其 50-51 行 `$services = @('postgres','mysql','redis','minio')` + `-DomainSuffix ".local"`，
> 由 `scripts/env/lib/host-utils.ps1:111,119-120` 拼成 `postgres.local` 等），与上面的裸主机名清单不一致；
> 其中 `mysql` 一项当前 compose 也未启用。两种写法各自自洽即可，**不要混用**：走脚本就在配置里写
> `postgres.local`，走本文清单就写 `postgres`，用 `localhost` 则两条都不需要。
