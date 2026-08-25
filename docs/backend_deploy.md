# 后端项目部署

- 所有的Docker配置文件都在`backend`目录下。
- 所有的部署脚本都在`backend/scripts`目录下。

> 详细的脚本使用说明请参考：[scripts/README.md](../backend/scripts/README.md)

Shell脚本需要赋予执行权限：

```bash
chmod +x ./scripts/**/*.sh
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
docker run -d -p 8013:8080 --network app-tier --name sse-gateway-local sse-gateway-local
```

说明：

- 网关必须与 admin-service 共处同一 Docker 网络（如 `app-tier`）才能访问后端。
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

## 本地开发配置 hosts

如果使用完整模式部署后需要从宿主机访问服务，需修改`hosts`文件（需要管理员权限）：

- Linux：`/etc/hosts`
- MacOS：`/private/etc/hosts`
- Windows：`C:\Windows\System32\drivers\etc\hosts`

增加以下内容：

```ini
127.0.0.1 postgres
127.0.0.1 redis
127.0.0.1 minio
127.0.0.1 consul
127.0.0.1 jaeger
```

> **注意**：如果注册中心使用Consul，consul的地址填写为`consul`会返回`502`，使用`localhost`或者`127.0.0.1`都可以。
> ```yaml
> registry:
>   type: "consul"
>
>   consul:
>     address: "localhost:8500"
> ```
>
> **推荐做法**：本地开发使用依赖模式（libs_only），配置文件中直接使用`localhost`即可，无需修改 hosts。
