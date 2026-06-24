<div align="center">

# GoWind Admin｜风行

**开箱即用的企业级前后端一体中后台脚手架**

> **让中后台开发如风般自由 — GoWind Admin**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vuedotjs)](https://vuejs.org/)
[![React](https://img.shields.io/badge/React-19.x-61DAFB?logo=react)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

[English](./README.en-US.md) | **中文** | [日本語](./README.ja-JP.md)

</div>

---

## 项目亮点

- **多前端适配**：同时提供 `Vue3 Vben`（Ant Design Vue）、`Vue3 Element Plus`、`React19 Antd` 三套前端，满足不同团队偏好
- **企业级 RBAC**：支持多租户、多角色、多部门、菜单/按钮/数据级权限控制（Casbin / OPA / Zanzibar）
- **微服务 + 单体自由切换**：基于 go-kratos 微服务框架，但支持单体架构模式开发与部署，灵活适配团队规模
- **全栈代码生成**：Protobuf → Go API / TypeScript 客户端，Ent Schema → ORM，一键 CRUD 脚手架
- **生产就绪**：JWT 鉴权、SSE 消息推送、异步任务调度、分布式链路追踪、Swagger 文档、Docker 一键部署

---

## 演示地址

| 前端版本 | 演示地址 |
|---------|--------|
| Vue3 Vben | <https://vben.admin.gowind.cloud> |
| Vue3 Element Plus | <https://ele.admin.gowind.cloud> |
| React | <https://react.admin.gowind.cloud> |

- 后端 Swagger：<https://api.demo.admin.gowind.cloud/docs/>
- 默认账号密码：`admin` / `admin`

---

## 技术栈

<table>
<tr><th>层级</th><th>技术</th></tr>
<tr><td><strong>后端框架</strong></td><td><code>Golang</code> · <code>go-kratos v2</code> · <code>Wire</code> · <code>Protobuf / Buf</code></td></tr>
<tr><td><strong>ORM</strong></td><td><code>Ent</code>（主要） · <code>GORM</code>（辅助） · <code>MySQL</code> · <code>PostgreSQL</code></td></tr>
<tr><td><strong>中间件</strong></td><td><code>Redis 8.0+</code> · <code>MinIO</code>（S3 兼容对象存储） · <code>Jaeger</code>（链路追踪）</td></tr>
<tr><td><strong>认证授权</strong></td><td><code>JWT</code> · <code>Casbin</code> · <code>OPA</code> · <code>Zanzibar</code></td></tr>
<tr><td><strong>实时通信</strong></td><td><code>SSE</code>（服务端推送） · <code>Asynq</code>（异步任务）</td></tr>
<tr><td><strong>脚本引擎</strong></td><td><code>go-scripts</code> · <code>Lua</code>（gopher-lua） · <code>JavaScript</code>（goja） · 多语言 Hook 插件系统</td></tr>
<tr><td><strong>Vue Vben 版</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Ant Design Vue</code> · <code>Vben Admin</code></td></tr>
<tr><td><strong>Vue Element 版</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Element Plus</code>（轻量纯净版）</td></tr>
<tr><td><strong>React 版</strong></td><td><code>React 19</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Zustand</code> · <code>Ant Design V6</code>（无 UMI）</td></tr>
<tr><td><strong>部署运维</strong></td><td><code>Docker</code> · <code>Docker Compose</code> · <code>PM2</code> · <code>Swagger UI</code></td></tr>
</table>

---

## 快速开始

### 环境要求

| 工具 | 版本 |
|------|------|
| Go | 1.22+ |
| Node.js | >= 20.10.0 |
| pnpm | >= 10.0.0 |
| Docker | 20.0+ |

### 环境脚本选型

- Linux / macOS 开发环境：`scripts/env/install_unix_dev.sh`
- Linux / macOS 生产环境：`scripts/env/install_unix_prod.sh`
- Windows 开发环境：`scripts/env/install_windows_dev.ps1`

### Docker 两种部署模式

- **full_deploy 完整模式**：同步启动中间件+后端应用，适用于一键演示、生产部署
- **libs_only 依赖模式（推荐开发）**：仅启动中间件，应用本地 IDE 运行调试

### 后端启动

**Linux / macOS：**

```shell
# 赋予脚本执行权限
chmod +x scripts/**/*.sh

# 开发环境（推荐）
./scripts/env/install_unix_dev.sh
./scripts/docker/libs_only.sh
gow run admin

# 生产环境
./scripts/env/install_unix_prod.sh
./scripts/docker/full_deploy.sh

# PM2 进程托管（生产进阶）
./scripts/deploy/pm2_service.sh
```

**Windows（PowerShell 管理员）：**

```powershell
# 放行脚本策略（首次仅需执行一次）
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# 初始化环境
.\scripts\env\install_windows_dev.ps1

# 本地开发
.\scripts\docker\libs_only.ps1
gow run admin

# 一键完整部署
.\scripts\docker\full_deploy.ps1
```

### 前端启动

前端统一存放于 `frontend/admin` 目录，三版前端共享依赖安装命令：

| 前端版本 | 目录 | 启动命令 | 端口 |
|---------|------|---------|------|
| React | `frontend/admin/react` | `pnpm dev` | 7000 |
| Vue Element | `frontend/admin/vue-element` | `pnpm dev` | 3000 |
| Vue Vben | `frontend/admin/vue-vben` | `pnpm dev:antd` | 5666 |

```shell
# 安装依赖
pnpm install

# React 版本
cd frontend/admin/react && pnpm dev

# Vue3 Element 版本
cd frontend/admin/vue-element && pnpm dev

# Vue3 Vben 版本
cd frontend/admin/vue-vben && pnpm dev:antd
```

---

## 功能列表

### 组织与权限

| 功能 | 说明 |
|------|-----|
| 用户管理 | 管理和查询用户，支持高级查询和按部门联动用户，用户可禁用/启用、设置/取消主管、重置密码、配置多角色、多部门和上级主管、一键登录指定用户等功能 |
| 租户管理 | 管理租户，新增租户后自动初始化租户部门、默认角色和管理员。支持配置套餐、禁用/启用、一键登录租户管理员功能 |
| 角色管理 | 管理角色和角色分组，支持按角色联动用户，设置菜单和数据权限，批量添加和移除员工 |
| 权限管理 | 管理权限分组、菜单、权限点，支持树形列表展示 |
| 组织管理 | 管理组织，支持树形列表展示 |
| 部门管理 | 管理部门，支持树形列表展示 |
| 职位管理 | 用户职务管理，职务可作为用户的一个标签 |
| 菜单管理 | 配置系统菜单，操作权限，按钮权限标识等，包括目录、菜单、按钮 |

### 系统功能

| 功能 | 说明 |
|------|-----|
| 接口管理 | 管理接口，支持接口同步功能，主要用于新增权限点时选择接口，支持树形列表展示、操作日志请求参数和响应结果配置 |
| 字典管理 | 管理数据字典大类及其小类，支持按字典大类联动字典小类、服务端多列排序、数据导入和导出 |
| 任务调度 | 管理和查看任务及其任务运行日志，支持任务新增、修改、删除、启动、暂停、立即执行 |
| 文件管理 | 管理文件上传，支持文件查询、上传到 OSS 或本地、下载、复制文件地址、删除文件、图片支持查看大图功能 |
| 缓存管理 | 缓存列表查询，支持根据缓存键清除缓存 |

### 消息与日志

| 功能 | 说明 |
|------|-----|
| 消息分类 | 管理消息分类，支持 2 级自定义消息分类，用于消息管理消息分类选择 |
| 消息管理 | 管理消息，支持发送指定用户消息，可查看用户是否已读和已读时间 |
| 站内信 | 站内消息管理，支持消息详细查看、删除、标为已读、全部已读功能 |
| 登录日志 | 登录日志列表查询，记录用户登录成功和失败日志，支持 IP 归属地记录 |
| 操作日志 | 操作日志列表查询，记录用户操作正常和异常日志，支持 IP 归属地记录，查看操作日志详情 |

### 个人中心

| 功能 | 说明 |
|------|-----|
| 个人中心 | 个人信息展示和修改，查看最后登录信息，密码修改等功能 |

---

## 项目结构

```
go-wind-admin/
├── backend/                        # 后端项目
│   ├── api/                        # Protobuf API 定义与生成代码
│   │   ├── protos/                 # .proto 源文件（按领域分层）
│   │   └── gen/go/                 # buf 生成的 Go 代码
│   ├── app/admin/service/          # Admin 服务应用
│   │   ├── cmd/server/             # 入口 (main.go, wire.go)
│   │   ├── configs/                # 配置文件 (YAML)
│   │   └── internal/               # 业务核心（data/service/server）
│   ├── pkg/                        # 公共包
│   │   ├── scripting/              # 多语言脚本引擎（Lua + JavaScript）
│   │   ├── oss/                    # 对象存储（MinIO）
│   │   ├── eventbus/               # 事件总线
│   │   └── ...                     # 其他工具包
│   ├── scripts/                    # 部署脚本（env/docker/deploy）
│   └── sql/                        # 初始化 SQL 文件
├── frontend/admin/                 # 前端项目
│   ├── react/                      # React 19 + Ant Design V6
│   ├── vue-element/                # Vue 3 + Element Plus
│   └── vue-vben/                   # Vue 3 + Ant Design Vue + Vben Admin
└── docs/                           # 项目文档
```

---

## 截图展示

<table>
    <tr>
        <td><img src="./docs/images/admin_login_page.png" alt="后台用户登录界面"/></td>
        <td><img src="./docs/images/admin_dashboard.png" alt="后台分析界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_user_list.png" alt="后台用户列表界面"/></td>
        <td><img src="./docs/images/admin_user_create.png" alt="后台创建用户界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_tenant_list.png" alt="后台租户列表界面"/></td>
        <td><img src="./docs/images/admin_tenant_create.png" alt="后台创建租户界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_org_unit_list.png" alt="组织单位列表界面"/></td>
        <td><img src="./docs/images/admin_org_unit_create.png" alt="创建组织单位界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_position_list.png" alt="后台职位列表界面"/></td>
        <td><img src="./docs/images/admin_position_create.png" alt="后台创建职位界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_role_list.png" alt="后台角色列表界面"/></td>
        <td><img src="./docs/images/admin_role_create.png" alt="后台创建角色界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_permission_list.png" alt="后台权限列表界面"/></td>
        <td><img src="./docs/images/admin_permission_create.png" alt="后台创建权限界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_menu_list.png" alt="后台目录列表界面"/></td>
        <td><img src="./docs/images/admin_menu_create.png" alt="后台创建目录界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_task_list.png" alt="后台调度任务列表界面"/></td>
        <td><img src="./docs/images/admin_task_create.png" alt="后台创建调度任务界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_dict_list.png" alt="后台数据字典列表界面"/></td>
        <td><img src="./docs/images/admin_dict_entry_create.png" alt="后台创建数据字典条目界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_internal_message_list.png" alt="后台站内信消息列表界面"/></td>
        <td><img src="./docs/images/admin_internal_message_publish.png" alt="后台发布站内信消息界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_login_policy_list.png" alt="登录策略列表界面"/></td>
        <td><img src="./docs/images/admin_login_policy_create.png" alt="登录策略创建界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_login_audit_log_list.png" alt="后台登录日志界面"/></td>
        <td><img src="./docs/images/admin_api_audit_log_list.png" alt="后台操作日志界面"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_api_list.png" alt="API列表界面"/></td>
        <td><img src="./docs/images/api_swagger_ui.png" alt="后端内置Swagger UI界面"/></td>
    </tr>
</table>

## 联系我们

- 微信个人号：`yang_lin_bo`（备注：`go-wind-admin`）
- 掘金专栏：[go-wind-admin](https://juejin.cn/column/7541283508041826367)

## 致谢

[![JetBrains](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)

感谢 JetBrains 提供免费的 GoLand & WebStorm 开源授权。
