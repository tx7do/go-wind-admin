# 文件上传架构与部署说明

本文档说明 GoWind Admin 文件上传链路的架构、部署依赖、元数据落库机制、安全校验，以及预签名上传路径的当前状态与启用条件。面向部署运维与后端开发。

---

## 部署依赖

上传功能依赖 SeaweedFS 对象存储（S3 兼容网关）。MinIO 社区版已于 2026-04 停止维护并归档仓库，本仓已切换；代码侧仍走 minio-go SDK 的标准 S3 语义，与具体服务端解耦。

### 对象存储服务（SeaweedFS）

SeaweedFS 通过 `backend/docker-compose.libs.yaml` 部署，与 PostgreSQL、Redis 一同由该 compose 文件管理。
**容器服务名/主机名沿用 `minio`**——`oss.yaml` 与 hosts 映射零改动。

```yaml
minio:
  image: docker.io/chrislusf/seaweedfs:4.51
  ports:
    - "9000:9000"   # S3 兼容 API
    - "9333:9333"   # Master Web UI
  volumes:
    - ./scripts/docker/seaweedfs/s3.json:/etc/seaweedfs/s3.json:ro
  command: server -dir=/data -filer -s3 -s3.port=9000 -s3.config=/etc/seaweedfs/s3.json
```

| 项 | 说明 |
|---|---|
| S3 API 端口 | 9000（`-s3.port` 指定，容器内外一致） |
| Web UI 端口 | 9333（Master UI，替代原 MinIO Console :9001） |
| 访问凭证 | `root` / `*Abcd123456`，定义于 `scripts/docker/seaweedfs/s3.json` 的 identities（生产环境务必修改，并与 `oss.yaml` 同步） |
| bucket 创建 | 无需预建——应用侧 `EnsureBucketExists` 在上传/签名时自动创建 |
| 存量迁移 | 旧 MinIO 数据用任意 S3 客户端按桶镜像即可（如 `mc mirror src dst`） |

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
    endpoint: "minio:9000"        # 对象存储 S3 API 地址（SeaweedFS，服务名沿用 minio）
    upload_host: "minio:9000"     # 上传请求实际访问的主机
    download_host: "minio:9000"   # 下载请求实际访问的主机
    access_key: "root"
    secret_key: "*Abcd123456"
    token: ""
    use_ssl: false
```

> **本地开发注意**：compose 里的服务名 `minio` 仅在容器网络内有效。本地直接运行后端时，需将 `endpoint`/`upload_host`/`download_host` 从 `minio` 改为 `localhost`。详见 `windows-startup-guide.md` 第三步。

`upload_host` 与 `download_host` 用于对对象存储返回的预签名/下载 URL 做主机替换——把容器内部地址替换成客户端可访问的外部地址。生产部署时这三个字段都应指向客户端可达的对象存储入口。

---

## 上传链路架构

上传入口：`POST /admin/v1/file/upload`、`PUT /admin/v1/file/upload`（在 `internal/server/i_file_transfer_http.pb.go` 手动注册，因为 multipart 表单无法由 proto 生成器处理）。

**请求体形态（2026-09-17 起，三端一致）**：前端以 **JSON** 提交——`file` 字段为 base64 编码的文件字节（kratos 未注册 form-data codec，multipart 表单对生成的服务桩不可用）。handler 保留双模式兼容：先尝试 multipart `FormFile("file")`，取不到再 `Bind` JSON 请求体；新调用方一律走 JSON。前端封装：`frontend/admin/react/src/api/hooks/file-transfer.ts`（vue-element / vue-vben 为 `src/api/composables/file-transfer.ts` 同型实现）。

请求进入 `FileTransferService.UploadFile`（`internal/service/file_transfer_service.go`），该函数按 `req.Source` 的 oneof 类型分流：

```
UploadFileRequest.Source
├── UploadFileRequest_File   → directUploadFile   ✅ 当前唯一可用路径
└── UploadFileRequest_Presign → presignedUploadFile ❌ 已禁用（返回未实现错误）
```

### directUploadFile（服务端中转，当前生效）

流程（`internal/service/file_transfer_service.go` 的 `directUploadFile`，146-245 行）：

1. **参数校验**：storageObject / file / mime / sourceFileName 非空。
2. **大小校验**：`len(file) > MaxUploadSize`（50 MiB）则拒绝，防 DoS。
3. **MIME 嗅探**：`oss.DetectFileType` 按文件内容（非客户端声明）判定真实 MIME，命中白名单才放行；用真实类型覆盖客户端声明，防 bucket 路由被绕过。
4. **目录安全校验**：`oss.IsFileDirectorySafe` 拒绝 `..`、绝对路径、非法字符，防穿越。
5. **操作者身份**：`auth.FromContext(ctx)` 从认证上下文取 tenantId / userId。
6. **bucket / objectName 兜底**：未指定时按 MIME 推断 bucket、按目录+文件名+UUID 生成 objectName。
7. **上传到对象存储**：`s.mc.UploadFile(...)` 返回 `UploadInfo`（含 Bucket/Key/Size）与 `downloadUrl`。
8. **元数据落库**：`recordFile(...)` 写入 `storage.File` 表。
9. **返回**：下载 URL。

关键点：**服务端持有完整文件字节**，因此 tenantId/userId/sourceFileName/sha256 都可在服务端取得，元数据完整。

### presignedUploadFile（预签名上传，已禁用）

预签名上传的工作方式与服务端中转根本不同：服务端只签发一个预签名 URL 交客户端，**实际文件上传由客户端稍后直接 PUT/POST 到对象存储，服务端不参与、不知道是否上传成功、拿不到文件字节**。

该路径当前在入口直接返回错误：

```go
return nil, storageV1.ErrorUploadFailed(
    "presigned upload is not implemented, use direct upload instead")
```

**禁用原因见下文「预签名上传路径」一节。**

---

## 元数据落库机制

`recordFile`（同文件 108-143 行）在 directUploadFile 成功后，将文件元数据写入 `storage.File` 表（经 `FileRepo.Create`）。

写入字段及其来源：

| 字段 | 来源 |
|---|---|
| `Provider` | 常量 `MINIO` |
| `BucketName` | `UploadInfo.Bucket`（SDK 返回） |
| `SaveFileName` / `FileDirectory` / `Extension` | `parseKey(UploadInfo.Key)` 解析得出 |
| `ContentHash` | `sha256.Sum256(fileData)` —— **需原始字节** |
| `FileName` | 客户端请求的 `sourceFileName` —— **需客户端提供** |
| `Size` | `UploadInfo.Size`（SDK 返回） |
| `LinkUrl` | SDK 返回的 `downloadUrl` |
| `FileGuid` | `id.NewGUIDv7()` 新生成 |
| `CreatedBy` | 认证上下文的 `userId` —— **需认证上下文** |
| `TenantId` | 认证上下文的 `tenantId` —— **需认证上下文** |

> 标 **需...** 的四项（sourceFileName / tenantId / userId / sha256）只能在上传时由服务端取得，无法从对象存储事件通知或对象 key 反推。这是预签名路径无法简单复用 recordFile 的根本原因。

元数据落库失败的处理（同文件 215-226 行的孤儿对象分支）：对象已入 OSS 但 DB 写入失败时（孤儿对象），记录 error 日志并返回错误，不掩盖问题，便于上层感知与后续清理。

---

## 安全校验

directUploadFile 在上传前施加多层校验，定义见 `pkg/oss/constants.go`：

| 校验 | 位置 | 说明 |
|---|---|---|
| 文件大小上限 | `file_transfer_service.go:164-165` | `MaxUploadSize = 50 MiB`，超限拒绝 |
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

- **密钥**：`GOWIND_CRYPTO_KEY` 环境变量（`go-utils/crypto` 的全局加密器与 HMAC 签名）。
  **未配置时 SignData 报错，PublicUrl 返回空串**——上传与下载不受影响，仅富文本内嵌预览不可用（前端编辑器拿不到公开 URL）。
- **有效期**：`mediaURLTTL = 365 天`（常量，`file_transfer_service.go`）。已嵌入历史富文本的图片链接在该期限内可访问。
- **验签**：`crypto.VerifyData`，内部 `hmac.Equal` **恒定时间比较**（防时序侧信道）。

### 代理端点（`ServeImageHandler`）

- 路由 `GET /admin/v1/file/image` 注册于 `i_file_transfer_http.pb.go` 的**手动注册块**——该路由不设 `http.SetOperation`，
  而 auth 中间件经 selector 按 **Operation** 匹配，故天然不适用鉴权（**免鉴权是结构性的，不在 `AddWhiteList` 里**）。
  安全性完全由 handler 内的三重校验承担：参数齐全 → `expires` 未过 → 签名验证（均失败即 403/400）。
- 通过后拆分 `{bucket}/{object}`，经 `GetObjectReader` 从对象存储**流式回源**（Content-Type/Content-Length 按对象元数据回填，
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
4. **当前无调用方**：前端上传实际走 JSON+base64（对应 directUploadFile），presign 分支无人调用。

### 启用条件

若未来确需预签名大文件直传（减轻后端带宽），启用前需引入完整闭环。注意 SeaweedFS **没有 S3 bucket notification**（不支持 MinIO 的 `PutBucketNotificationConfiguration` 事件订阅），事件源推荐改用**存储无关的回执 + 对账**方案：

1. **客户端回执**：客户端 PUT 完成后调用服务端 confirm 接口（携带 objectKey 与 PUT 响应的 ETag）；服务端 HeadObject 校验对象存在 + 大小一致 +（非分片时）ETag 一致后才落库——伪造回执会被 HeadObject 挡住。sha256 无法在不读全量字节的前提下服务端校验，接受降级（该字段在预签名路径下留空或事后补算）。
2. **待确认表**：presignedUploadFile 签发 URL 前，把 sourceFileName / tenantId / userId 存入一张「待确认」表，关联 objectKey；**配额预检在这一步做**（直传绕过后端，不预检就拦不住超大上传）。
3. **定时对账兜底**：定时任务扫描到期未确认的行——对象真在（HeadObject/ListObjects 核实）则补全落库，不在则清掉；顺带清孤儿对象。与站内信「台账 + DeliverySweep」同一定式，不依赖任何事件机制，换存储后端也不用改。
4. **定时清理**：待确认表加 `created_at`，由定时任务清理 24 小时未完成（拿了 URL 却未上传）的过期记录，防表无限膨胀。

（备选事件源：SeaweedFS filer 的 `[notification]` webhook——`filer.toml` 配置，S3 写入底层经 filer 也会触发，支持 Kafka/PubSub/SQS/Webhook。但载荷是 SeaweedFS 自有格式需适配、送达保障弱于真消息队列，且用了仍需第 3 步对账兜底，一般不值得引入。）

这是一个跨新增表、新增端点、依赖装配、定时任务的中等规模改动，启用前应整体规划。

---

## 配置项速查

影响上传行为的配置项及其位置：

| 配置项 | 位置 | 作用 |
|---|---|---|
| `oss.minio.endpoint` | `configs/oss.yaml` | 对象存储 S3 API 地址（服务端为 SeaweedFS；容器内为 `minio:9000`，本地开发改为 `localhost:9000`） |
| `oss.minio.upload_host` / `download_host` | `configs/oss.yaml` | 对对象存储返回的 URL 做主机替换，指向客户端可达地址 |
| `oss.minio.access_key` / `secret_key` | `configs/oss.yaml` | 对象存储凭证（生产环境务必修改，并与 `s3.json` 的 identities 一致） |
| `MaxUploadSize` | `pkg/oss/constants.go:13` | 单次直传上限（50 MiB），改常量需重新编译 |
| `AllowedMimePrefixes` / `AllowedExactMimeTypes` | `pkg/oss/constants.go:21,29` | 上传 MIME 白名单 |
| `s3.json` identities | `backend/scripts/docker/seaweedfs/s3.json`（compose 挂载） | SeaweedFS S3 凭证与权限；须与 `oss.yaml` 的 `access_key`/`secret_key` 一致 |
| `GOWIND_CRYPTO_KEY` | 环境变量 | 签名公开 URL 的 HMAC 密钥（`go-utils/crypto`）；未配置则 `PublicUrl` 恒为空串，富文本内嵌预览不可用；轮换=全部存量公开 URL 作废 |

---

## 相关文件索引

| 文件 | 说明 |
|---|---|
| `backend/app/admin/service/internal/service/file_transfer_service.go` | 上传/下载业务逻辑（directUploadFile / presignedUploadFile / recordFile / DownloadFile） |
| `backend/app/admin/service/internal/server/i_file_transfer_http.pb.go` | 手动注册的上传/下载 HTTP 端点（multipart `FormFile` 与 JSON `Bind` 双模式兼容，前端实际走 JSON+base64）；含签名图片代理路由的免鉴权手动注册块 |
| `go-utils/crypto` | 签名公开 URL 的 HMAC-SHA256 签发/恒定时间验签（`GOWIND_CRYPTO_KEY`） |
| `backend/pkg/oss/minio.go` | 对象存储客户端封装（minio-go SDK，S3 兼容服务端通用；UploadFile / DownloadFile / GetUploadPresignedUrl 等） |
| `go-utils/ossutil` | 安全常量与校验函数（大小上限、MIME 白名单、目录校验）及 objectName 生成、MIME 嗅探、bucket 路由等工具 |
| `backend/app/admin/service/configs/oss.yaml` | 对象存储连接配置（键名沿用 `minio` 前缀） |
| `backend/scripts/docker/seaweedfs/s3.json` | SeaweedFS S3 凭证配置（compose 挂载） |
| `backend/docker-compose.libs.yaml` | SeaweedFS 容器定义（服务名沿用 minio） |
| `api/protos/storage/service/v1/file.proto` | `storage.File` 元数据 message 定义 |
| `api/protos/storage/service/v1/file_transfer.proto` | 上传/下载 RPC 定义 |
