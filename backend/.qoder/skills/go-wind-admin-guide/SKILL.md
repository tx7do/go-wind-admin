---
name: go-wind-admin-guide
description: Go-Wind-Admin 后端项目的架构指南和开发规范。当前为单体架构，采用微服务框架（Kratos）搭建，可按需拆分。提供项目结构、技术栈、代码生成、CRUD开发流程、Wire依赖注入等指导。适用于在此Go项目中开发新功能、理解架构或排查问题时。
---

# Go-Wind-Admin 后端项目开发指南

## 项目概览

Go-Wind-Admin 是基于 **Go + Kratos** 框架的后台管理系统。当前 admin 服务为**单体架构**，但采用了微服务框架（Kratos）搭建，便于后续按需拆分为独立微服务。项目采用经典的三层架构。

### 技术栈

| 层面     | 技术                           |
|--------|------------------------------|
| 框架     | go-kratos/kratos v2          |
| ORM    | entgo.io/ent (Ent)           |
| DB     | MySQL / PostgreSQL / SQLite  |
| 缓存     | Redis (go-redis/v9)          |
| 对象存储   | MinIO                        |
| API 定义 | Protocol Buffers 3 (buf 工具链) |
| 依赖注入   | google/wire                  |
| 认证     | JWT (kratos-authn)           |
| 授权     | Casbin / OPA (kratos-authz)  |
| 异步任务   | Asynq                        |
| 实时推送   | SSE                          |
| 脚本引擎   | go-scripts (Lua + JavaScript) |
| 可观测性   | OpenTelemetry                |

### Go 模块路径

```
go-wind-admin
```

## 项目目录结构

```
backend/
├── api/                          # Protobuf API 定义与生成代码
│   ├── protos/                   # .proto 源文件
│   │   ├── admin/service/v1/     # 管理后台服务接口
│   │   ├── permission/service/v1/ # 权限相关消息定义
│   │   ├── identity/service/v1/  # 身份相关消息定义
│   │   └── ...
│   └── gen/go/                   # buf 生成的 Go 代码
├── app/
│   └── admin/service/            # Admin 微服务应用
│       ├── cmd/server/           # 入口 (main.go, wire.go)
│       ├── configs/              # 配置文件 (YAML)
│       └── internal/
│           ├── data/             # 数据层 (Repository)
│           │   ├── ent/          # Ent 生成代码 & schema
│           │   ├── gorm/         # GORM 相关
│           │   ├── providers/    # Wire provider set
│           │   └── *_repo.go     # 各资源 Repository
│           ├── server/           # 传输层 (HTTP/gRPC/Asynq/SSE)
│           │   └── providers/    # Wire provider set
│           └── service/          # 业务逻辑层 (Service)
│               ├── providers/    # Wire provider set
│               └── *_service.go  # 各资源 Service
├── pkg/                          # 公共包
│   ├── authorizer/               # 授权引擎
│   ├── middleware/auth/          # 认证中间件
│   ├── jwt/                      # JWT 工具
│   ├── oss/                      # 对象存储
│   ├── eventbus/                 # 事件总线
│   ├── scripting/                # 多语言脚本引擎 (Lua + JavaScript)
│   └── ...
└── scripts/                      # 部署/安装脚本
```

## 三层架构

```
Proto (API 定义) → Service (业务逻辑) → Data/Repo (数据访问)
```

### 1. Proto 层 - API 定义 (两层架构)

本项目采用 **源领域 + BFF 层** 的 Proto 两层架构:

**源领域层** (如 `api/protos/permission/service/v1/`):
- 定义消息类型 (message)
- 定义完整的 gRPC Service（**不带** `google.api.http` 注解）
- 提供全部 RPC 方法 (List, Count, Get, Create, Update, Delete + 业务方法)

**BFF 层** (如 `api/protos/admin/service/v1/`):
- 定义 REST Service（**带** `google.api.http` 路由注解）
- import 源领域的消息类型，不重复定义
- 可裁剪 API 面积，只暴露前端需要的 RPC
- 例如源领域有 8 个 RPC，BFF 层可能只暴露 6 个

- 使用 `buf generate` 生成 Go 代码到 `api/gen/go/`

### 2. Service 层 - 业务逻辑

- 位置: `app/admin/service/internal/service/*_service.go`
- 实现 protobuf 生成的接口
- 通过 `auth.FromContext(ctx)` 获取操作人信息
- 调用 Repo 层进行数据操作
- 注入依赖: authorizer, repo, log 等

### 3. Data 层 - 数据访问

- 位置: `app/admin/service/internal/data/*_repo.go`
- 使用 Ent ORM 进行数据库操作
- 使用自封装的 `go-crud` 库，**同时支持 Ent 和 GORM 两种 ORM**:
  - **Ent** (主要使用): `go-crud/entgo` 泛型 Repository，用于所有 CRUD 操作
  - **GORM** (辅助): `go-crud/gorm` Client，当前主要用于自动迁移 (`gorm/models/`)
- 通过 `go-utils/mapper.CopierMapper` 做 Entity ↔ DTO 自动转换（基于 copierutil 转换器）
- Repository 泛型签名包含 Ent 的 9 种类型（Query, Select, Create, CreateBulk, Update, UpdateOne, Delete, Predicate, Entity）

### 4. Server 层 - 传输层

- 位置: `app/admin/service/internal/server/rest_server.go`
- 注册所有 Service 到 HTTP Server
- 配置认证/授权中间件 (白名单机制)
- 支持 Swagger UI

## Wire 依赖注入

每一层都有 `providers/wire_set.go` 定义 ProviderSet:

```go
// service/providers/wire_set.go
var ProviderSet = wire.NewSet(
    service.NewXxxService,
    // ...
)
```

入口在 `cmd/server/wire.go`，合并三层 ProviderSet 后由 Wire 生成 `wire_gen.go`。

**修改依赖后需重新生成:**
```bash
make wire
# 或
cd app/admin/service && make wire
```

## 添加新 CRUD 功能

详细流程参见 [add-crud-feature.md](add-crud-feature.md)

## 构建与运行

详细指令参见 [build-and-run.md](build-and-run.md)

## 编码约定

1. **错误处理**: 使用 protobuf 定义的错误码 (`adminV1.ErrorBadRequest`, `permissionV1.ErrorInternalServerError` 等)
2. **参数校验**: Service 层入口校验 `req == nil` 和 `req.Data == nil`
3. **操作人记录**: Create/Update 操作通过 `auth.FromContext(ctx)` 获取 userId
4. **可选字段**: 使用 `trans.Ptr()` 将标量转为指针，Ent 使用 `SetNillable*` 方法
5. **注释风格**: 中英双语注释 `// 中文说明 / English description`
6. **日志**: 通过 `ctx.NewLoggerHelper("module/name")` 创建命名日志器
