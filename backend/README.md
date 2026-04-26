# GO后端

本项目是基于 Go + `gow` 脚手架 构建的**微服务后端架构**，支持：

- 多协议：RESTful API + gRPC
- 多存储：MySQL/PostgreSQL、Redis、MinIO
- 服务治理：Etcd 注册发现、Jaeger 链路追踪
- 代码生成：Ent ORM、Wire 依赖注入、Protobuf 自动生成、TypeScript 客户端
- 容器化：Docker + Docker Compose 一键部署
- 文档自动生成：Swagger UI、OpenAPI v3
- CRUD自动生成：基于 SQL的DDL语句 一键生成 CRUD 接口和实现

## 前置环境要求

### 2.1 基础环境

| 工具             | 版本要求  | 说明          |
|----------------|-------|-------------|
| Go             | 1.21+ | 核心开发语言      |
| Docker         | 20.0+ | 容器化部署       |
| Docker Compose | v2+   | 服务编排        |
| Make           | 3.8+  | 构建工具        |
| PM2            | 可选    | 进程管理（物理机部署） |

### 2.2 中间件（自动部署）

| 中间件        | 版本要求          | 说明                      |
|------------|---------------|-------------------------|
| Redis      | 6.0+          | 必须 ≥6.0，项目使用 HExpire 命令 |
| Postgresql | 14+           | 关系型数据库，支持 JSON / 事务     |
| MinIO      | RELEASE.2024+ | 兼容 S3 协议的对象存储服务         |
| Jaeger     | 1.40+         | 分布式链路追踪，可视化请求耗时         |

## 内置API文档

### Swagger UI

- [Admin Swagger UI](http://localhost:7788/docs/)

### openapi.yaml

- [Admin openapi.yaml](http://localhost:7788/docs/openapi.yaml)

## 项目目录结构

```text
myproject/
├── api/                    # API定义/Protobuf
├── app/                    # 微服务主目录
│   └── admin/              # 单个服务
│       └── service/        # 服务启动目录
│              ├── cmd/         # 启动入口
│              ├── configs/     # 配置文件
│              └── internal/    # 业务核心代码
│                     ├── data/     # 数据访问层（Ent ORM）
│                     ├── server/   # gRPC/HTTP服务层
│                     └── service/  # 业务逻辑层
├── scripts/             # 部署脚本
├── go.mod               # Go模块依赖
├── go.sum               # 依赖校验
└── Makefile             # 构建命令
```

## gow 脚手架工具使用

### 安装

```bash
# 安装最新版
go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest

# 验证安装
gow version  # 查看版本
gow help     # 查看帮助
```

### 环境变量配置（可选）

将 Go 安装目录加入环境变量，确保全局可调用`gow`：

```bash
# Linux/Mac
export PATH=$PATH:$(go env GOPATH)/bin

# Windows（PowerShell）
$env:PATH += ";$(go env GOPATH)\bin"
```

### 核心命令

#### 创建新项目

```bash
# 基础创建（默认模块名）
gow new myproject
cd myproject
go mod tidy  # 安装依赖

# 指定自定义模块名（推荐）
gow new myproject -m github.com/yourname/myproject
```

#### 添加微服务

```shell
# 基础服务（默认REST）
gow add service admin
gow add service user

# 高级用法
gow add service order -s grpc          # 纯gRPC服务
gow add service pay -s rest -s grpc    # 双协议服务
gow add svc goods -d gorm -d redis     # 带数据库+缓存
go mod tidy                            # 同步依赖
```

#### 运行服务

```shell
# 方式1：进入服务目录运行
cd app/admin/service
gow run

# 方式2：根目录指定服务运行
gow run admin
```

#### 代码自动生成

```bash
# 生成Ent ORM代码
gow ent          # 所有服务
gow ent admin    # 指定服务

# 生成Wire依赖注入代码
gow wire
gow wire admin

# 生成Protobuf/API代码
gow api
```

## Makefile 构建命令详解

### 环境初始化

```bash
make init     # 安装所有依赖、工具、插件
make cli      # 更新命令行工具
make plugin   # 更新项目插件
```

### 代码生成（开发常用）

```bash
# 进入服务目录
cd app/admin/service

make api      # 生成API/Protobuf代码
make ent      # 生成Ent ORM代码
make wire     # 生成依赖注入代码
make openapi  # 生成OpenAPI文档
make ts       # 生成前端TS代码
```

### 运行与构建

```bash
make run      # 调试运行服务
make build    # 编译二进制文件
```

### Docker 容器管理

```bash
make compose-up      # 启动所有服务+中间件
make compose-down    # 停止所有服务+中间件
make docker         # 构建服务Docker镜像
```

## 一键部署整个项目

部署项目有两种方法：

1. 三方中间件和微服务都运行在Docker之下；
2. 三方中间件运行在Docker下，微服务通过PM2管理运行在OS下。

#### 1. 三方中间件和微服务都运行在Docker之下

```bash
cd ./scripts/docker
chmod +x *.sh
./full_deploy.sh
```

#### 2. 三方中间件运行在Docker下，微服务运行在OS下

先部署三方中间件：

```bash
cd ./scripts/docker
chmod +x *.sh
./libs_only.sh
```

接着部署PM2管理的微服务：

```bash
cd ./scripts/deploy
chmod +x *.sh
./pm2_service.sh
```

##  常见问题 FAQ

### go.sum 依赖验证失败

有时候，有的依赖包在`go mod tidy`之后会出现`go.sum`验证不通过的问题，解决方法：

```bash
go clean -modcache
go mod tidy
```

如果还不行的话，可以删除`go.sum`文件，然后重新执行`go mod tidy`。

### 中间件连接失败（postgres/redis/minio）

**原因**：容器内域名解析需要配置 hosts

我们需要修改`hosts`文件，修改需要管理员权限，其配置文件路径在：

- Linux：`/etc/hosts`
- MacOS: `/private/etc/hosts`
- Windows: `C:\Windows\System32\drivers\etc\hosts`

增加以下内容：

```ini
127.0.0.1 postgres
127.0.0.1 redis
127.0.0.1 minio
127.0.0.1 consul
127.0.0.1 jaeger
```

> 注意：如果注册中心使用Consul，consul的地址填写为`consul`会返回`502`，使用`localhost`或者`127.0.0.1`都可以。
> ```yaml
> registry:
> type: "consul"
>
> consul:
> address: "localhost:8500"
> ```

# Redis 认证失败 / 命令报错

- **原因**：项目使用`HExpire`命令，仅支持 Redis 6.0+
- **解决**：升级 Redis 到 6.0 及以上版本
