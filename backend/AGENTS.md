# Go-Wind-Admin 后端项目开发指南

本文件为 AI Agent 提供项目级开发规范和约定，所有代码生成和修改必须遵循以下规则。

## 项目概览

Go-Wind-Admin 是基于 **Go + Kratos** 框架的后台管理系统。当前 admin 服务为**单体架构**，但采用了微服务框架（Kratos）搭建，便于后续按需拆分为独立微服务。项目采用经典的三层架构。

Go 模块路径: `go-wind-admin`

### 技术栈

| 层面     | 技术                            |
|--------|-------------------------------|
| 框架     | go-kratos/kratos v2           |
| ORM    | entgo.io/ent (Ent)            |
| DB     | MySQL / PostgreSQL / SQLite   |
| 缓存     | Redis (go-redis/v9)           |
| 对象存储   | MinIO                         |
| API 定义 | Protocol Buffers 3 (buf 工具链)  |
| 依赖注入   | 手写构造注入 (cmd/server/wiring_ent.go) |
| 认证     | JWT (kratos-authn)            |
| 授权     | Casbin / OPA (kratos-authz)   |
| 异步任务   | Asynq                         |
| 实时推送   | SSE                           |
| 脚本引擎   | go-scripts (Lua + JavaScript) |
| 可观测性   | OpenTelemetry                 |

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
│       ├── cmd/server/           # 入口 (main.go, wiring_ent.go / wiring_gorm.go 依赖装配)
│       ├── configs/              # 配置文件 (YAML)
│       └── internal/
│           ├── data/             # 数据层 (Repository)
│           │   ├── ent/          # Ent 生成代码 & schema [禁止手动修改]
│           │   ├── gorm/         # GORM 相关
│           │   └── *_repo.go     # 各资源 Repository
│           ├── server/           # 传输层 (HTTP/Asynq/SSE)
│           └── service/          # 业务逻辑层 (Service)
│               └── *_service.go  # 各资源 Service
├── pkg/                          # 公共包
│   ├── authorizer/               # 授权引擎
│   ├── middleware/auth/          # 认证中间件
│   ├── jwt/                      # JWT 工具
│   ├── oss/                      # 对象存储
│   ├── eventbus/                 # 事件总线
│   ├── scripting/                # 多语言脚本引擎 (Lua + JavaScript)
│   │   ├── api/                  # 业务 API 模块 (cache/eventbus/oss/crypto...)
│   │   ├── hook/                 # Hook 注册表
│   │   └── internal/convert/     # Go ↔ 脚本值转换
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
- 可裁剪 API 面积，只暴露前端需要的 RPC（源领域可能 8 个 RPC，BFF 只暴露 6 个）
- 使用 `buf generate` 生成 Go 代码到 `api/gen/go/`

### 2. Service 层 - 业务逻辑

- 位置: `app/admin/service/internal/service/*_service.go`
- 实现 protobuf 生成的接口
- 通过 `auth.FromContext(ctx)` 获取操作人信息
- 调用 Repo 层进行数据操作
- 注入依赖: authorizer, repo, log 等

### 3. Data 层 - 数据访问

- 位置: `app/admin/service/internal/data/*_repo.go`
- 使用自封装的 `go-crud` 库，**同时支持 Ent 和 GORM 两种 ORM**:
  - **Ent** (主要): `go-crud/entgo` 泛型 Repository，用于所有 CRUD 操作
  - **GORM** (辅助): `go-crud/gorm` Client，当前主要用于自动迁移 (`gorm/models/`)
- 通过 `go-utils/mapper.CopierMapper` 做 Entity ↔ DTO 自动转换（注册 copierutil 转换器处理类型差异）
- Repository 泛型签名包含 Ent 的 9 种类型（Query, Select, Create, CreateBulk, Update, UpdateOne, Delete, Predicate, Entity）
- 必须注册时间转换器: `TimeStringConvertedPair` + `TimeTimestamppbConverterPair`
- enum 字段需用 `mapper.NewEnumTypeConverter` 注册
- `ListWithPaging` 传入 builder 和 builder.Clone()，自动处理分页/排序/搜索
- `UpdateX` 支持 FieldMask 部分更新

### 4. Server 层 - 传输层

- 位置: `app/admin/service/internal/server/rest_server.go`
- 注册所有 Service 到 HTTP Server
- 配置认证/授权中间件 (白名单机制)
- 支持 Swagger UI

## 依赖装配 (手写 wiring)

依赖注入不使用框架,由 `cmd/server/wiring_ent.go`(!gorm_backend)的 `initApp` 手写构造注入,
自上而下单向分层:基础设施 → 仓储层(data) → 认证与鉴权 → 服务层(service) → 传输层(server)。
GORM 后端为平行文件 `cmd/server/wiring_gorm.go`(gorm_backend):其仓储层(`newGormRepos`)已是
可编译的真实代码,服务层待 repo 接口抽取(ORM 切换 Phase 4)后接通,当前 tag 构建仅剩预期错误
`undefined: initApp`。

**新增 CRUD 模块的登记(推荐自动化):**

```bash
make register ENTITY=product    # 在 backend/ 根目录执行
```

一条命令完成全部五处登记: `wiring_ent.go` 的仓储构造行、服务构造行、`NewRestServer` 实参,以及
`rest_server.go` 的服务形参与路由注册调用。注入位置由文件内的 `register:*` 锚点注释标记,
工具可重复执行(幂等)。仅覆盖标准 CRUD 形态(`New<Ent>Repo(ctx, entClient)` /
`New<Ent>Service(ctx, <ent>Repo)`),依赖更多的模块需手工调整注入行。

**手工登记(等价):** 在 `wiring_ent.go` 对应分层小节追加构造行并传给下游消费者;
漏接由编译器在调用处报错,无需任何代码生成步骤。

## 添加新 CRUD 功能 (以 Product 为例)

### 完整流程概览

```
1. 源领域: 定义消息 + gRPC Service
2. BFF 层: 定义 REST Service (带 HTTP 路由)
3. 生成 Go 代码
4. 创建 Ent Schema → 5. 生成 Ent 代码
6. 创建 Repository → 7. 创建 Service → 8. 注册到 Server
9. 在 wiring_ent.go 注册依赖 → 10. 验证
```

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
message GetProductRequest {
  oneof query_by { uint32 id = 1; }
  optional google.protobuf.FieldMask view_mask = 100 [json_name = "viewMask"];
}
message CreateProductRequest { Product data = 1; }
message UpdateProductRequest {
  uint32 id = 1;
  Product data = 2;
  google.protobuf.FieldMask update_mask = 3 [json_name = "updateMask"];
  optional bool allow_missing = 4 [json_name = "allowMissing"];
}
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
  // 注意: 源领域有 Count RPC，但 BFF 层可选择性不暴露
}
```

**裁剪原则**: BFF 层只暴露前端需要的 RPC，源领域中的 `Count`、`BatchCreate` 等方法可不在 REST 层暴露。

### Step 3: 生成 Go 代码

```bash
gow api        # 推荐，可在任意位置执行
# 或
cd backend && make api && make openapi
```

### Step 4-5: 创建 Ent Schema 并生成

在 `app/admin/service/internal/data/ent/schema/` 下创建 schema（字段用 `.Optional()` 对应 proto `optional`，可用 `entsql.Annotation{Table: "..."}` 指定表名），然后:

```bash
gow ent        # 推荐
# 或
cd app/admin/service && make ent
```

### Step 6: 创建 Repository

在 `app/admin/service/internal/data/` 下创建 `*_repo.go`。**关键**: 使用自封装的 `go-crud/entgo` 泛型 Repository，泛型签名包含 Ent 生成的 9 种类型。骨架（参照现有 `api_repo.go`）:

```go
type ProductRepo struct {
    entClient *entCrud.EntClient[*ent.Client]
    log       *log.Helper
    mapper     *mapper.CopierMapper[pb.Product, ent.Product]
    repository *entCrud.Repository[
        // Ent 生成的 9 种泛型类型，顺序固定不可调换
        ent.ProductQuery, ent.ProductSelect,       // Query, Select
        ent.ProductCreate, ent.ProductCreateBulk,   // Create, CreateBulk
        ent.ProductUpdate, ent.ProductUpdateOne,    // Update, UpdateOne
        ent.ProductDelete,                          // Delete
        predicate.Product,                          // Predicate
        pb.Product, ent.Product,                    // DTO, Entity
    ]
}

func NewProductRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ProductRepo {
    r := &ProductRepo{
        entClient: entClient,
        log:       ctx.NewLoggerHelper("repo/product"),
        mapper:    mapper.NewCopierMapper[pb.Product, ent.Product](),
    }
    r.repository = entCrud.NewRepository[ /* 9 个类型... */ ](r.mapper)
    // 注册时间类型转换器 (必须)
    r.mapper.AppendConverters(copierutil.NewTimeStringConvertedPair())
    r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
    // 有 enum 字段时: r.mapper.AppendConverters(mapper.NewEnumTypeConverter(...))
    return r
}

// List: r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
// Get:  r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
// Create: entClient.Client().Product.Create().SetNillable*().Exec(ctx)
// Update: r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(), setCallback, whereCallback)
// Delete: r.repository.Delete(ctx, builder, whereCallback)
```

**go-crud 关键概念**: 泛型签名 9 参数顺序固定；`CopierMapper` 经 `ConverterPair` 处理 Ent Entity ↔ Protobuf DTO 类型差异；`ListWithPaging` 自动处理分页/排序/搜索；`UpdateX` 第二个回调设置 WHERE 条件，支持 FieldMask。

### Step 7: 创建 Service

在 `app/admin/service/internal/service/` 下创建 `*_service.go`，实现 protobuf 接口，通过 `auth.FromContext(ctx)` 获取操作人:

```go
type ProductService struct {
    adminV1.UnimplementedProductServiceServer
    repo *data.ProductRepo
    log  *log.Helper
}

func (s *ProductService) Create(ctx context.Context, req *pb.CreateProductRequest) (*emptypb.Empty, error) {
    if req.Data == nil { return nil, adminV1.ErrorBadRequest("invalid parameter") }
    operator, err := auth.FromContext(ctx)
    if err != nil { return nil, err }
    req.Data.CreatedBy = trans.Ptr(operator.UserId)
    if err = s.repo.Create(ctx, req); err != nil { return nil, err }
    return &emptypb.Empty{}, nil
}
// List, Get, Update, Delete 方法参照 api_service.go 模式
```

### Step 8: 注册到 Server

编辑 `rest_server.go`:
1. 在 `NewRestServer` 参数中添加 `productService *service.ProductService`
2. 注册 HTTP handler: `adminV1.RegisterProductServiceHTTPServer(srv, productService)`

(这两处也可由 Step 9 的 `make register` 自动注入,手工编辑可省略。)

### Step 9: 注册依赖装配

```bash
make register ENTITY=product    # 自动注入 wiring_ent.go 三处 + rest_server.go 两处(含 Step 8 两项)
```

或手工编辑 `cmd/server/wiring_ent.go` 对应分层小节:仓储行 `productRepo := data.NewProductRepo(ctx, entClient)`、
服务行 `productService := service.NewProductService(ctx, productRepo)`,并把 `productService`
传入 `server.NewRestServer(...)`(工具幂等,与手工编辑混用不会重复注入)。

### Step 10: 验证

```bash
gow run        # 无需先构建
# 访问 http://localhost:7788/docs 验证
```

## 构建与运行

### 前置条件

- Go 1.22+
- Docker & Docker Compose (可选，用于依赖服务)
- buf CLI (Protobuf 代码生成)
- gow CLI (推荐): `go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest`

### 代码生成命令

**使用 gow (推荐，所有命令可在项目任意位置执行):**

| 命令                                                                                            | 说明                                        |
|-----------------------------------------------------------------------------------------------|-------------------------------------------|
| `gow api`                                                                                     | 生成所有服务的 Protobuf & API 代码 (自动遍历所有 buf 配置) |
| `gow ent` / `gow ent admin`                                                                   | 生成所有 / 指定服务的 Ent 代码                      |
| `gow run` / `gow run admin`                                                                   | 运行当前 / 指定服务                              |
| `gow generate --dsn "mysql://..." --service admin --orm ent`                                  | 从数据库一键生成完整 CRUD 代码                       |
| `gow extract admin user -o role[,permission] [--keep-source]`                                 | 微服务演进：从 admin 提取模块到目标服务                 |

**Make 命令 (需在 `backend/` 根目录执行):**

| 命令                | 说明                                |
|-------------------|-----------------------------------|
| `make api`        | 生成 Protobuf Go 代码                 |
| `make openapi`    | 生成 OpenAPI 文档                     |
| `make ent`        | 生成 Ent ORM 代码                     |
| `make gen`        | 一键生成 (ent + api + openapi)        |
| `make build`      | 构建 (含 api + openapi 生成)           |
| `make build_only` | 仅构建，不生成代码                         |
| `make test`       | 运行测试                              |
| `make cover`      | 覆盖率测试                             |
| `make lint`       | golangci-lint                     |
| `make docker-libs`| 启动 MySQL/Redis/MinIO 等基础服务         |

### 配置文件

位于 `app/admin/service/configs/`:

| 文件            | 用途                   |
|---------------|----------------------|
| `server.yaml` | HTTP/Asynq/SSE 服务器配置 |
| `data.yaml`   | 数据库连接配置              |
| `auth.yaml`   | 认证(JWT)配置            |
| `logger.yaml` | 日志配置                 |
| `oss.yaml`    | 对象存储配置               |
| `client.yaml` | 客户端配置                |

## 编码约定

1. **错误处理**: 使用 protobuf 定义的错误码 (`adminV1.ErrorBadRequest`, `permissionV1.ErrorInternalServerError` 等)，不要用标准 `errors.New`
2. **参数校验**: Service 层入口校验 `req == nil` 和 `req.Data == nil`
3. **操作人记录**: Create/Update 操作通过 `auth.FromContext(ctx)` 获取 userId
4. **可选字段**: 使用 `trans.Ptr()` 将标量转为指针，Ent 使用 `SetNillable*` 方法
5. **注释风格**: 中英双语注释 `// 中文说明 / English description`
6. **日志**: 通过 `ctx.NewLoggerHelper("module/name")` 创建命名日志器，命名遵循 `模块/子模块` 格式
7. **禁止手动修改**: `api/gen/go/` 和 `internal/data/ent/` 下的生成代码;依赖装配统一在 `cmd/server/wiring_ent.go` 手写维护
