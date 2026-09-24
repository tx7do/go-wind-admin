# 09 · 部署上线：从依赖模式到生产

> 前置：[02 章](./02-get-it-running.md)跑过本地环境。读完你会知道：部署形态怎么选、SSE 网关为什么独立、生产密钥怎么注入、备份怎么转——以及每一项的权威文档在哪。

## 1. 部署形态决策

| 形态 | 中间件 | 应用 | 适用 |
|---|---|---|---|
| 依赖模式（libs_only） | Docker | 本地 IDE | **开发调试**（第 02 章用的就是它） |
| 完整模式（full_deploy） | Docker | Docker | 一键演示、容器化生产 |
| PM2 物理机 | 物理机 | PM2 进程托管 | 非容器的生产物理机 |

统一入口是 `backend/scripts/`（完整用法见 [`scripts/README.md`](../../backend/scripts/README.md)）：

```bash
# 完整模式
./scripts/docker/full_deploy.sh          # Linux/macOS
.\scripts\docker\full_deploy.ps1         # Windows

# 依赖模式
./scripts/docker/libs_only.sh

# PM2 物理机
./scripts/deploy/pm2_service.sh
```

注意：**Shell 脚本要先赋执行权限**，且 `scripts/` 下有三层子目录，`chmod +x ./scripts/**/*.sh`
会漏掉 `env/lib` 与 `deploy/sse`——用 `find ./scripts -name '*.sh' -exec chmod +x {} +` 一次到位。
完整模式下宿主机访问容器服务要改 hosts（`postgres` / `redis` / `minio` 三个主机名映射 127.0.0.1，
清单见 [backend_deploy.md](../backend_deploy.md)）。

## 2. 网络拓扑与职责边界

```
Internet ─ TLS 终止（外层负载均衡/nginx）
             │
   ┌─────────┴──────────┐
   │  均衡/反代          │  ← TLS、WAF、限流在这里，不在应用
   └──┬──────────┬──────┘
      │          │
  静态前端站点   admin-service（:7788 REST）
                SSE 网关（:8013 → 容器内 8080 → 7789/events）
```

**SSE 网关是独立 nginx 容器**，不随 docker-compose 启动，需单独构建运行（它关闭了 gzip 与全部
代理缓冲、启用 HTTP/1.1 长连接保活——这些是 SSE 实时性的必要配置，勿回退）。构建与运行命令、
网络（必须与 admin-service 同处 `app-tier`）、upstream 修改：全部见
[backend_deploy.md](../backend_deploy.md)「SSE 反向代理网关」节。

## 3. 生产密钥与配置

JWT 签名密钥**环境变量优先于 `configs/auth.yaml`**（内置的是开发示例密钥，生产必须替换，否则启动告警）：

| 环境变量 | 用途 |
|---|---|
| `GWA_AUTH_JWT_PRIVATE_KEY` / `GWA_AUTH_JWT_PUBLIC_KEY` | 非对称（RS256/ES256/Ed25519）PEM 对 |
| `GWA_AUTH_JWT_KEY` | 对称（HS256）共享密钥 |

```bash
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out jwt_private_key.pem
openssl pkey -in jwt_private_key.pem -pubout -out jwt_public_key.pem
```

其他生产前核对项：

- **MinIO**：`configs/oss.yaml` 的 `endpoint`/`upload_host`/`download_host` 指向客户端可达入口，
  并替换 root 凭证（见 [backend_file_upload.md](../backend_file_upload.md)）；
- **authz 引擎**：`configs/auth.yaml` 的 `authz.type` 从 `noop` 切到 casbin/opa（第 05 章）；
- **审计留存**：`AUDIT_RETENTION_DAYS` 按合规要求核对（第 07 章）；
- **脚本出站**：`SCRIPT_HTTP_ALLOWED_DOMAINS` 按需配置（第 08 章）。

## 4. 数据备份

[`scripts/backup/pg_backup.sh`](../../backend/scripts/backup/pg_backup.sh) 定时全量备份
（pg_dump，默认保留 30 份自动轮换），支持 Docker 容器 / 本地直连双模式，附恢复操作文档。
用 cron/计划任务挂上，并**演练一次恢复**——没验证过的备份等于没有。

## 5. 上线检查单

1. 密钥注入（JWT、MinIO 凭证、DB 口令）；
2. TLS 在外层负载均衡启用，后端只听内网；
3. authz 引擎切非 noop；审计留存天数核对；
4. 新端点做了「接口同步」、新菜单进了菜单管理（第 04/06 章）；
5. 备份任务挂上且恢复演练过；
6. SSE 网关与后端同网络、缓冲配置未被改动；
7. 演示数据清理（`backend/sql/` 是演示数据，生产库别导）。

## 深读

- [backend_deploy.md](../backend_deploy.md) —— 部署唯一权威：两模式、PM2、SSE 网关、密钥、hosts
- [`backend/scripts/README.md`](../../backend/scripts/README.md) —— 脚本逐个用法
- [backend_file_upload.md](../backend_file_upload.md) —— 对象存储的生产配置
- 根 [README.md](../../README.md)「安全与等保合规」—— 安全能力矩阵（对外口径）

---

**系列完**。回到 [docs/README.md](../README.md) 看参考层清单——接下来你改哪个子系统，就先读哪篇权威文档。
