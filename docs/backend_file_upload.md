# 文件上传架构与部署说明

本文档说明 GoWind Admin 文件上传链路的架构、部署依赖、元数据落库机制、安全校验，以及预签名上传路径的当前状态与启用条件。面向部署运维与后端开发。

---

## 部署依赖

上传功能依赖 MinIO 对象存储服务。

### MinIO 服务

MinIO 通过 `backend/docker-compose.libs.yaml` 部署，与 PostgreSQL、Redis 一同由该 compose 文件管理。

```yaml
minio:
  image: docker.io/minio/minio:latest
  ports:
    - "9000:9000"   # S3 API
    - "9001:9001"   # 管理控制台
  environment:
    - MINIO_ROOT_USER=root
    - MINIO_ROOT_PASSWORD=*Abcd123456
    - MINIO_DEFAULT_BUCKETS=images
  command: server /data --console-address ':9001'
```

| 项 | 说明 |
|---|---|
| S3 API 端口 | 9000（容器内外一致） |
| 管理控制台端口 | 9001，浏览器访问可查看 bucket/对象 |
| 默认 bucket | `images`（由 `MINIO_DEFAULT_BUCKETS` 自动创建） |
| 根凭证 | `root` / `*Abcd123456`（生产环境务必修改） |

启动服务：

```bash
cd backend
docker compose -f docker-compose.libs.yaml up -d
```

### 后端配置文件

上传相关配置在 `backend/app/admin/service/configs/oss.yaml`：

```yaml
oss:
  minio:
    endpoint: "minio:9000"        # MinIO API 地址
    upload_host: "minio:9000"     # 上传请求实际访问的主机
    download_host: "minio:9000"   # 下载请求实际访问的主机
    access_key: "root"
    secret_key: "*Abcd123456"
    token: ""
    use_ssl: false
```

> **本地开发注意**：compose 里的服务名 `minio` 仅在容器网络内有效。本地直接运行后端时，需将 `endpoint`/`upload_host`/`download_host` 从 `minio` 改为 `localhost`。详见 `windows-startup-guide.md` 第三步。

`upload_host` 与 `download_host` 用于对 MinIO 返回的预签名/下载 URL 做主机替换——把容器内部地址替换成客户端可访问的外部地址。生产部署时这三个字段都应指向客户端可达的 MinIO 入口。

---

## 上传链路架构

上传入口：`POST /admin/v1/file/upload`、`PUT /admin/v1/file/upload`（在 `internal/server/i_file_transfer_http.pb.go` 手动注册，因为 multipart 表单无法由 proto 生成器处理）。

请求进入 `FileTransferService.UploadFile`（`internal/service/file_transfer_service.go`），该函数按 `req.Source` 的 oneof 类型分流：

```
UploadFileRequest.Source
├── UploadFileRequest_File   → directUploadFile   ✅ 当前唯一可用路径
└── UploadFileRequest_Presign → presignedUploadFile ❌ 已禁用（返回未实现错误）
```

### directUploadFile（服务端中转，当前生效）

流程（`file_transfer_service.go:142-227`）：

1. **参数校验**：storageObject / file / mime / sourceFileName 非空。
2. **大小校验**：`len(file) > MaxUploadSize`（50 MiB）则拒绝，防 DoS。
3. **MIME 嗅探**：`oss.DetectFileType` 按文件内容（非客户端声明）判定真实 MIME，命中白名单才放行；用真实类型覆盖客户端声明，防 bucket 路由被绕过。
4. **目录安全校验**：`oss.IsFileDirectorySafe` 拒绝 `..`、绝对路径、非法字符，防穿越。
5. **操作者身份**：`auth.FromContext(ctx)` 从认证上下文取 tenantId / userId。
6. **bucket / objectName 兜底**：未指定时按 MIME 推断 bucket、按目录+文件名+UUID 生成 objectName。
7. **上传到 MinIO**：`s.mc.UploadFile(...)` 返回 `UploadInfo`（含 Bucket/Key/Size）与 `downloadUrl`。
8. **元数据落库**：`recordFile(...)` 写入 `storage.File` 表。
9. **返回**：下载 URL。

关键点：**服务端持有完整文件字节**，因此 tenantId/userId/sourceFileName/sha256 都可在服务端取得，元数据完整。

### presignedUploadFile（预签名上传，已禁用）

预签名上传的工作方式与服务端中转根本不同：服务端只签发一个预签名 URL 交客户端，**实际文件上传由客户端稍后直接 PUT/POST 到 MinIO，服务端不参与、不知道是否上传成功、拿不到文件字节**。

该路径当前在入口直接返回错误：

```go
return nil, storageV1.ErrorUploadFailed(
    "presigned upload is not implemented, use direct upload instead")
```

**禁用原因见下文「预签名上传路径」一节。**

---

## 元数据落库机制

`recordFile`（`file_transfer_service.go:104-139`）在 directUploadFile 成功后，将文件元数据写入 `storage.File` 表（经 `FileRepo.Create`）。

写入字段及其来源：

| 字段 | 来源 |
|---|---|
| `Provider` | 常量 `MINIO` |
| `BucketName` | `UploadInfo.Bucket`（MinIO 返回） |
| `SaveFileName` / `FileDirectory` / `Extension` | `parseKey(UploadInfo.Key)` 解析得出 |
| `ContentHash` | `sha256.Sum256(fileData)` —— **需原始字节** |
| `FileName` | 客户端请求的 `sourceFileName` —— **需客户端提供** |
| `Size` | `UploadInfo.Size`（MinIO 返回） |
| `LinkUrl` | MinIO 返回的 `downloadUrl` |
| `FileGuid` | `id.NewGUIDv7()` 新生成 |
| `CreatedBy` | 认证上下文的 `userId` —— **需认证上下文** |
| `TenantId` | 认证上下文的 `tenantId` —— **需认证上下文** |

> 标 **需...** 的四项（sourceFileName / tenantId / userId / sha256）只能在上传时由服务端取得，无法从 MinIO 事件通知或对象 key 反推。这是预签名路径无法简单复用 recordFile 的根本原因。

元数据落库失败的处理（`file_transfer_service.go:217-222`）：对象已入 OSS 但 DB 写入失败时（孤儿对象），记录 error 日志并返回错误，不掩盖问题，便于上层感知与后续清理。

---

## 安全校验

directUploadFile 在上传前施加多层校验，定义见 `pkg/oss/constants.go`：

| 校验 | 位置 | 说明 |
|---|---|---|
| 文件大小上限 | `file_transfer_service.go:160` | `MaxUploadSize = 50 MiB`，超限拒绝 |
| MIME 白名单 | `oss.IsAllowedMimeType` | 仅允许 image/* / video/* / audio/* 前缀及一批精确文档类型（pdf/zip/office 等） |
| 真实 MIME 嗅探 | `oss.DetectFileType` | 按文件内容（非客户端声明）判定，防止伪装扩展名绕过 bucket 路由 |
| 目录穿越校验 | `oss.IsFileDirectorySafe` | 拒绝 `..`、绝对路径、非法字符；仅允许 `[a-zA-Z0-9_/-]` |

下载路径 `downloadFileFromURL`（从外部 URL 拉取文件）另有完整 SSRF 防护：scheme 白名单、逐 IP 内网校验、DialContext 钉死 IP 防 DNS rebinding、重定向二次校验、响应体大小上限 `MaxDownloadSize = 50 MiB`、超时。

---

## 签名公开访问 URL 与图片代理

服务端中转上传成功后，除走鉴权 API 的下载地址外，另返回一个**签名公开访问 URL**（`UploadFileResponse.PublicUrl`），供富文本编辑器等场景直接以 `<img>` 内嵌引用。

### URL 形态与签名机制

```
GET /admin/v1/file/image?path={bucket}/{object}&expires={unix}&sig={hmac}
sig = HMAC-SHA256(GOWIND_CRYPTO_KEY, "{path}|{expires}")   // hex
```

- **密钥**：`GOWIND_CRYPTO_KEY` 环境变量（`pkg/crypto/hmac.go` 的全局加密器）。
  **未配置时 SignData 报错，PublicUrl 返回空串**——上传与下载不受影响，仅富文本内嵌预览不可用（前端编辑器拿不到公开 URL）。
- **有效期**：`mediaURLTTL = 365 天`（常量，`file_transfer_service.go`）。已嵌入历史富文本的图片链接在该期限内可访问。
- **验签**：`crypto.VerifyData`，内部 `hmac.Equal` **恒定时间比较**（防时序侧信道）。

### 代理端点（`ServeImageHandler`）

- 路由 `GET /admin/v1/file/image` 注册于 `i_file_transfer_http.pb.go` 的**手动注册块**——该路由不设 `http.SetOperation`，
  而 auth 中间件经 selector 按 **Operation** 匹配，故天然不适用鉴权（**免鉴权是结构性的，不在 `AddWhiteList` 里**）。
  安全性完全由 handler 内的三重校验承担：参数齐全 → `expires` 未过 → 签名验证（均失败即 403/400）。
- 通过后拆分 `{bucket}/{object}`，经 `GetObjectReader` 从 MinIO **流式回源**（Content-Type/Content-Length 按对象元数据回填，
  响应头 `Cache-Control: public, max-age=31536000, immutable`）。
- 该端点不做租户/用户级鉴别：**签名 URL 是不记名凭证**——持有者即可在有效期内访问对象，无接收方绑定、不可撤销轮换（除非换 `GOWIND_CRYPTO_KEY` 全量作废）。

### 边界与注意

- **对象路径不自动带租户维度**（[tenant_isolation.md](./tenant_isolation.md) 第 6 节覆盖边界）：签名覆盖的是完整
  `{bucket}/{object}` 串，路径本身不可篡改；但 URL 的分发面（富文本内容、转发）即访问面。
- 需要撤销/短期访问时，走鉴权下载端点（`GET /admin/v1/file/download`）而非公开 URL。
- 换 `GOWIND_CRYPTO_KEY` 会使全部存量公开 URL 立即失效（含历史富文本内嵌图）——轮换前评估。
- 上传侧元数据落库失败时（孤儿对象）按 H3 语义告警并向上抛错，公开 URL 不会签出。

---

## 预签名上传路径

### 为何禁用

预签名上传（presignedUploadFile）当前禁用，原因如下：

1. **元数据无法可靠落库**：该路径服务端不接触文件字节，`recordFile` 所需的 sourceFileName / tenantId / userId / sha256 四个字段无法取得——前三个未编码进对象 key（key 仅由 UUID/SHA/HMAC/时间戳+扩展名生成），sha256 需原始字节。
2. **客户端不可信**：把 tenantId/userId 放进 `x-amz-meta-*` 对象元数据的方案不可行——浏览器请求可被篡改，恶意用户可伪造 `x-amz-meta-user-id` 冒充他人上传，破坏 directUploadFile 从认证上下文取身份的安全语义。
3. **当前无刚需**：业务无大文件直传需求（`MaxUploadSize = 50 MiB`，前端无分片上传），服务端中转零问题且安全语义完整。
4. **当前无调用方**：前端上传实际走 multipart（对应 directUploadFile），presign 分支无人调用。

### 启用条件

若未来确需预签名大文件直传（减轻后端带宽），启用前需引入完整闭环：

1. **MinIO 事件通知**：配置 bucket notification（webhook 或 queue），在对象真正写入时由 MinIO 回调服务端。
2. **待确认表**：presignedUploadFile 签发 URL 前，把 sourceFileName / tenantId / userId 存入一张「待确认」表，关联 objectKey。
3. **回调端点**：服务端新增一个接收 MinIO 事件通知的 HTTP 端点（需加入 `rest_server.go` 的白名单豁免 auth，因 MinIO 回调无法携带 token）。
4. **配对补全**：回调端点按事件 payload 的 objectKey 匹配待确认表，补全元数据字段，并 GetObject 下载对象计算 sha256，最后落库。
5. **定时清理**：待确认表加 `created_at`，由定时任务清理 24 小时未完成（拿了 URL 却未上传）的过期记录，防表无限膨胀。

这是一个跨 MinIO 配置、新增表、新增端点、依赖装配、定时任务的中等规模改动，启用前应整体规划。

---

## 配置项速查

影响上传行为的配置项及其位置：

| 配置项 | 位置 | 作用 |
|---|---|---|
| `oss.minio.endpoint` | `configs/oss.yaml` | MinIO API 地址（容器内为 `minio:9000`，本地开发改为 `localhost:9000`） |
| `oss.minio.upload_host` / `download_host` | `configs/oss.yaml` | 对 MinIO 返回的 URL 做主机替换，指向客户端可达地址 |
| `oss.minio.access_key` / `secret_key` | `configs/oss.yaml` | MinIO 凭证（生产环境务必修改） |
| `MaxUploadSize` | `pkg/oss/constants.go:10` | 单次直传上限（50 MiB），改常量需重新编译 |
| `AllowedMimePrefixes` / `AllowedExactMimeTypes` | `pkg/oss/constants.go:18,26` | 上传 MIME 白名单 |
| `MINIO_DEFAULT_BUCKETS` | `docker-compose.libs.yaml` | 启动时自动创建的 bucket |
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | `docker-compose.libs.yaml` | MinIO 根凭证（生产环境务必修改） |
| `GOWIND_CRYPTO_KEY` | 环境变量 | 签名公开 URL 的 HMAC 密钥（`pkg/crypto/hmac.go`）；未配置则 `PublicUrl` 恒为空串，富文本内嵌预览不可用；轮换=全部存量公开 URL 作废 |

---

## 相关文件索引

| 文件 | 说明 |
|---|---|
| `backend/app/admin/service/internal/service/file_transfer_service.go` | 上传/下载业务逻辑（directUploadFile / presignedUploadFile / recordFile / DownloadFile） |
| `backend/app/admin/service/internal/server/i_file_transfer_http.pb.go` | 手动注册的上传/下载 HTTP 端点（处理 multipart）；含签名图片代理路由的免鉴权手动注册块 |
| `backend/pkg/crypto/hmac.go` | 签名公开 URL 的 HMAC-SHA256 签发/恒定时间验签（`GOWIND_CRYPTO_KEY`） |
| `backend/pkg/oss/minio.go` | MinIO 客户端封装（UploadFile / DownloadFile / GetUploadPresignedUrl 等） |
| `backend/pkg/oss/constants.go` | 安全常量与校验函数（大小上限、MIME 白名单、目录校验） |
| `backend/pkg/oss/utils.go` | objectName 生成、MIME 嗅探、bucket 路由等工具 |
| `backend/app/admin/service/configs/oss.yaml` | MinIO 连接配置 |
| `backend/docker-compose.libs.yaml` | MinIO 容器定义 |
| `api/protos/storage/service/v1/file.proto` | `storage.File` 元数据 message 定义 |
| `api/protos/storage/service/v1/file_transfer.proto` | 上传/下载 RPC 定义 |
