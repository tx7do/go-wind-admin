# Go-Wind-Admin 后端项目开发指南

## 项目概览

Go-Wind-Admin 是基于 **Go + Kratos** 框架的后台管理系统。当前 admin 服务为**单体架构**，但采用了微服务框架（Kratos）搭建，便于后续按需拆分为独立微服务。项目采用经典的三层架构。

Go 模块路径: `go-wind-admin`

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

## 项目目录结构

```
backend/
├── api/                          # Protobuf API 定义与生成代码
│   ├── protos/                   # .proto 源文件
│   │   ├── admin/service/v1/     # 管理后台 REST 接口 (BFF 层)
│   │   ├── permission/service/v1/ # 权限相关 (源领域层)
│   │   ├── identity/service/v1/  # 身份相关 (源领域层)
│   │   └── ...
│   └── gen/go/                   # buf 生成的 Go 代码
├── app/
│   └── admin/service/            # Admin 服务应用
│       ├── cmd/server/           # 入口 (main.go, wire.go)
│       ├── configs/              # 配置文件 (YAML)
│       └── internal/
│           ├── data/             # 数据层 (Repository)
│           │   ├── ent/          # Ent 生成代码 & schema [禁止手动修改]
│           │   ├── gorm/         # GORM 相关
│           │   ├── providers/    # Wire provider set
│           │   └── *_repo.go     # 各资源 Repository
│           ├── server/           # 传输层 (HTTP/Asynq/SSE)
│           │   └── providers/    # Wire provider set
│           └── service/          # 业务逻辑层 (Service)
│               ├── providers/    # Wire provider set
│               └── *_service.go  # 各资源 Service
├── pkg/                          # 公共包
│   ├── scripting/                # 多语言脚本引擎 (Lua + JavaScript)
│   │   ├── api/                  # 业务 API 模块 (cache/eventbus/oss/crypto...)
│   │   ├── hook/                 # Hook 注册表
│   │   └── internal/convert/     # Go ↔ 脚本值转换
│   ├── oss/                      # 对象存储
│   ├── eventbus/                 # 事件总线
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
- 使用 `buf generate` 生成 Go 代码到 `api/gen/go/`

### 2. Service 层 - 业务逻辑

- 位置: `app/admin/service/internal/service/*_service.go`
- 实现 protobuf 生成的接口
- 通过 `auth.FromContext(ctx)` 获取操作人信息
- 调用 Repo 层进行数据操作

### 3. Data 层 - 数据访问

- 位置: `app/admin/service/internal/data/*_repo.go`
- 使用自封装的 `go-crud` 库，**同时支持 Ent 和 GORM 两种 ORM**:
  - **Ent** (主要): `go-crud/entgo` 泛型 Repository，用于所有 CRUD 操作
  - **GORM** (辅助): `go-crud/gorm` Client，当前主要用于自动迁移 (`gorm/models/`)
- 通过 `go-utils/mapper.CopierMapper` 做 Entity ↔ DTO 自动转换（注册 copierutil 转换器处理类型差异）
- Repository 泛型签名包含 Ent 的 9 种类型（Query, Select, Create, CreateBulk, Update, UpdateOne, Delete, Predicate, Entity）
- 必须注册时间转换器: `TimeStringConverterPair` + `TimeTimestamppbConverterPair`
- enum 字段需用 `mapper.NewEnumTypeConverter` 注册
- `ListWithPaging` 传入 builder 和 builder.Clone()，自动处理分页/排序/搜索
- `UpdateX` 支持 FieldMask 部分更新

### 4. Server 层 - 传输层

- 位置: `app/admin/service/internal/server/rest_server.go`
- 注册所有 Service 到 HTTP Server
- 配置认证/授权中间件 (白名单机制)
- 支持 Swagger UI

## Wire 依赖注入

每一层都有 `providers/wire_set.go` 定义 ProviderSet。入口在 `cmd/server/wire.go`。

```bash
# 重新生成 Wire 代码
gow wire          # 推荐
# 或
cd app/admin/service && make wire
```

## 构建与运行

### 前置条件

- Go 1.22+
- Docker & Docker Compose (可选)
- buf CLI (Protobuf 代码生成)
- gow CLI (推荐): `go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest`

### 常用命令

**代码生成 (推荐使用 gow):**

| 命令         | 说明                                  |
|------------|-------------------------------------|
| `gow api`  | 生成 Protobuf & API 代码 (自动遍历 buf 配置) |
| `gow ent`  | 生成 Ent ORM 代码                      |
| `gow wire` | 生成 Wire 依赖注入代码                     |
| `gow run`  | 运行服务                                |

**Make 命令 (需在 backend/ 根目录执行):**

| 命令                | 说明                                |
|-------------------|-----------------------------------|
| `make api`        | 生成 Protobuf Go 代码                 |
| `make openapi`    | 生成 OpenAPI 文档                     |
| `make ent`        | 生成 Ent ORM 代码                     |
| `make wire`       | 生成 Wire 依赖注入代码                    |
| `make gen`        | 一键生成 (ent + wire + api + openapi) |
| `make build`      | 构建 (含 api + openapi 生成)           |
| `make test`       | 运行测试                              |

### 配置文件

位于 `app/admin/service/configs/`: `server.yaml`, `data.yaml`, `auth.yaml`, `logger.yaml`, `oss.yaml`, `client.yaml`

## 编码约定

1. **错误处理**: 使用 protobuf 定义的错误码 (`adminV1.ErrorBadRequest`, `permissionV1.ErrorInternalServerError` 等)
2. **参数校验**: Service 层入口校验 `req == nil` 和 `req.Data == nil`
3. **操作人记录**: Create/Update 操作通过 `auth.FromContext(ctx)` 获取 userId
4. **可选字段**: 使用 `trans.Ptr()` 将标量转为指针，Ent 使用 `SetNillable*` 方法
5. **注释风格**: 中英双语注释 `// 中文说明 / English description`
6. **日志**: 通过 `ctx.NewLoggerHelper("module/name")` 创建命名日志器
7. **禁止手动修改**: `wire_gen.go`、`api/gen/go/` 和 `internal/data/ent/` 下的生成代码

## 添加新 CRUD 功能 (以 Product 为例)

### Step 1: 源领域层 - 定义消息 + gRPC Service

在 `api/protos/<domain>/service/v1/` 下定义，**不带** `google.api.http` 注解:

```protobuf
// api/protos/catalog/service/v1/product.proto
syntax = "proto3";
package catalog.service.v1;

import "gnostic/openapi/v3/annotations.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/field_mask.proto";
import "pagination/v1/pagination.proto";

service ProductService {
  rpc List (pagination.PagingRequest) returns (ListProductResponse) {}
  rpc Count (pagination.PagingRequest) returns (CountProductResponse) {}
  rpc Get (GetProductRequest) returns (Product) {}
  rpc Create (CreateProductRequest) returns (google.protobuf.Empty) {}
  rpc Update (UpdateProductRequest) returns (google.protobuf.Empty) {}
  rpc Delete (DeleteProductRequest) returns (google.protobuf.Empty) {}
}

message Product {
  optional uint32 id = 1 [json_name = "id"];
  optional string name = 2 [json_name = "name"];
  optional string description = 3 [json_name = "description"];
  optional uint32 status = 4 [json_name = "status"];
  optional uint32 created_by = 100 [json_name = "createdBy"];
  optional uint32 updated_by = 101 [json_name = "updatedBy"];
  optional google.protobuf.Timestamp created_at = 200 [json_name = "createdAt"];
  optional google.protobuf.Timestamp updated_at = 201 [json_name = "updatedAt"];
}

message ListProductResponse { repeated Product items = 1; uint64 total = 2; }
message GetProductRequest { oneof query_by { uint32 id = 1; } optional google.protobuf.FieldMask view_mask = 100; }
message CreateProductRequest { Product data = 1; }
message UpdateProductRequest { uint32 id = 1; Product data = 2; google.protobuf.FieldMask update_mask = 3; optional bool allow_missing = 4; }
message DeleteProductRequest { oneof query_by { uint32 id = 1; } }
message CountProductResponse { uint64 count = 1; }
```

### Step 2: BFF 层 - 定义 REST Service

在 `api/protos/admin/service/v1/` 下创建，import 源领域消息，按需裁剪:

```protobuf
// api/protos/admin/service/v1/i_product.proto
syntax = "proto3";
package admin.service.v1;

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";
import "pagination/v1/pagination.proto";
import "catalog/service/v1/product.proto";

service ProductService {
  rpc List (pagination.PagingRequest) returns (catalog.service.v1.ListProductResponse) {
    option (google.api.http) = { get: "/admin/v1/products" };
  }
  rpc Get (catalog.service.v1.GetProductRequest) returns (catalog.service.v1.Product) {
    option (google.api.http) = { get: "/admin/v1/products/{id}" };
  }
  rpc Create (catalog.service.v1.CreateProductRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = { post: "/admin/v1/products" body: "*" };
  }
  rpc Update (catalog.service.v1.UpdateProductRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = { put: "/admin/v1/products/{id}" body: "*" };
  }
  rpc Delete (catalog.service.v1.DeleteProductRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = { delete: "/admin/v1/products/{id}" };
  }
}
```

### Step 3: 生成 Go 代码

```bash
gow api        # 推荐，可在任意位置执行
# 或
make api && make openapi
```

### Step 4-5: 创建 Ent Schema 并生成

在 `app/admin/service/internal/data/ent/schema/` 下创建 schema，然后:

```bash
gow ent        # 推荐
# 或
cd app/admin/service && make ent
```

### Step 6: 创建 Repository

在 `app/admin/service/internal/data/` 下创建 `*_repo.go`。

**关键**: 使用自封装的 `go-crud/entgo` 泛型 Repository，泛型签名包含 Ent 生成的 9 种类型:
`Query, Select, Create, CreateBulk, Update, UpdateOne, Delete, Predicate, DTO, Entity`

使用 `mapper.CopierMapper` 做 Entity ↔ DTO 转换，必须注册时间转换器。参照现有 `api_repo.go`。

### Step 7: 创建 Service

在 `app/admin/service/internal/service/` 下创建 `*_service.go`，实现 protobuf 接口，通过 `auth.FromContext(ctx)` 获取操作人。

### Step 8: 注册到 Server

编辑 `rest_server.go`，添加 Service 参数并注册 HTTP handler。

### Step 9: 更新 Wire ProviderSet

更新 `data/providers/wire_set.go` 和 `service/providers/wire_set.go`。

### Step 10: 重新生成 Wire

```bash
gow wire       # 推荐
# 或
cd app/admin/service && make wire
```

### Step 11: 验证

```bash
gow run        # 无需先构建
# 访问 http://localhost:7788/docs 验证
```
