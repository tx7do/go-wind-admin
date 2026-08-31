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
- **安全与等保合规**：按等保 2.0 技术要求内置 180 天审计日志留存归档、口令策略三件套、TOTP MFA、口令应用层加密、动态 RBAC 与多租户隔离、定时备份轮换，详见[安全与等保合规](#安全与等保合规)
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

## 安全与等保合规

本项目的安全能力参照《网络安全等级保护 2.0》（二级/三级）技术要求设计，面向企业高隐私私有化部署场景开箱即用：

| 等保技术要求 | 落地实现 |
|------------|---------|
| **安全审计** | 六类审计日志全覆盖：登录 / 操作 / API / 数据访问 / 权限变更 / 策略评估，记录 IP 归属地与 trace_id。asynq 每日定时归档：库内留存 180 天（`AUDIT_RETENTION_DAYS` 可调），超期数据导出 JSONL 归档文件留痕，库瘦身与日志留存两不误 |
| **身份鉴别** | 口令复杂度（≥8 位、小写/大写/数字/符号四类取三）、历史口令复用检查（默认近 3 条）、口令有效期（默认 90 天），阈值均支持环境变量调整；TOTP 多因素认证（MFA）；图形验证码；Redis 登录失败限流（IP + 用户名双维度）；可配置登录限制策略 |
| **访问控制** | 动态 RBAC 权限引擎（Casbin / OPA / Zanzibar 可切换），角色—权限—接口映射存于数据库，权限变更即时热更新生效；菜单/按钮/数据级权限控制；每次鉴权判定落策略评估日志可追溯 |
| **多租户隔离** | ent Privacy 策略编译级数据隔离；租户请求按 `(path, method)` 经 Api 表 fail-closed 校验（缺权限点即拒绝）；套餐模块白名单与到期只读策略 |
| **数据保密性** | 登录口令应用层 AES 加密传输、bcrypt 哈希存储；敏感任务配置 AES-256-GCM 静态加密（Ent Hook 透明加解密）；JWT RS256 非对称签名；refresh token 走 HttpOnly Cookie；传输层 TLS 由部署层启用（后端 `server.rest.tls` 配置或 nginx / 负载均衡终止） |
| **数据备份恢复** | [`scripts/backup/pg_backup.sh`](./backend/scripts/backup/pg_backup.sh) 定时全量备份（pg_dump，默认保留 30 份自动轮换），支持 Docker 容器 / 本地直连双模式，附恢复操作文档 |
| **前端安全** | 三套前端生产构建均启用 CSP、X-Frame-Options、HSTS 等安全响应头 |

> **说明**：等保测评除技术要求外，还包含管理制度、物理环境、人员组织等非软件范畴的内容。本项目覆盖的是技术措施部分，可为私有化部署的等保测评准备提供直接支撑，但不能替代完整的等保测评流程。

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
| React | `frontend/admin/react` | `pnpm dev` | 5888 |
| Vue Element | `frontend/admin/vue-element` | `pnpm dev` | 5777 |
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
| 套餐与配额管理 | 管理租户订阅套餐及其资源配额（如模块白名单、用量上限），支持套餐与配额项的增删改查 |
| 角色管理 | 管理角色和角色分组，支持按角色联动用户，设置菜单和数据权限，批量添加和移除员工 |
| 权限管理 | 管理权限分组、菜单、权限点，支持树形列表展示 |
| 组织管理 | 管理组织，支持树形列表展示 |
| 职位管理 | 用户职务管理，职务可作为用户的一个标签 |
| 菜单管理 | 配置系统菜单，操作权限，按钮权限标识等，包括目录、菜单、按钮 |

### 系统功能

| 功能 | 说明 |
|------|-----|
| 接口管理 | 管理接口，支持接口同步功能，主要用于新增权限点时选择接口，支持树形列表展示、操作日志请求参数和响应结果配置 |
| 字典管理 | 管理数据字典大类及其小类，支持按字典大类联动字典小类、服务端多列排序、数据导入和导出 |
| 任务调度 | 管理和查看任务及其任务运行日志，支持任务新增、修改、删除、启动、暂停、立即执行 |
| 文件管理 | 管理文件上传，支持文件查询、上传到 OSS 或本地、下载、复制文件地址、删除文件、图片支持查看大图功能 |
| 登录策略 | 管理登录限制策略，配置目标用户的限制类型、限制方式、限制值与限制原因 |
| 账号登录 | 支持用户名 / 邮箱 / 手机号多标识登录，可叠加图形验证码、登录策略与 TOTP 多因素认证 |
| 多因素认证（MFA） | 基于 TOTP 的多因素认证，含登录挑战、个人中心绑定管理，以及管理员救援重置用户 MFA 的解锁路径 |
| 语言管理 | 管理系统支持的多语言，配置语言名称、语言代码、本地名称、启用与默认状态 |

### 消息与日志

| 功能 | 说明 |
|------|-----|
| 消息分类 | 管理消息分类，支持 2 级自定义消息分类，用于消息管理消息分类选择 |
| 消息管理 | 管理消息，支持按发送范围（全员 / 指定用户）发送与消息撤销，全员广播走异步任务队列投递（断点恢复、幂等），可查看用户是否已读和已读时间 |
| 站内信 | 站内消息管理，支持消息详细查看、删除、标为已读、全部已读功能 |
| 登录日志 | 登录日志列表查询，记录用户登录成功和失败日志，支持 IP 归属地记录 |
| 操作日志 | 操作日志列表查询，记录用户操作正常和异常日志，支持 IP 归属地记录与资源对象定位，查看操作日志详情 |
| API 日志 | API 日志列表查询，记录 API 请求的操作者、请求路径、方法与成功状态，支持 IP 归属地记录 |
| 数据日志 | 数据访问日志列表查询，记录数据访问行为，SQL 词法脱敏，自动提取涉及表名与数据分类 |
| 权限日志 | 权限变更日志列表查询，记录权限变更的操作者、目标对象与原因，留存请求快照 |
| 策略评估日志 | 策略评估日志列表查询，记录每次鉴权判定的结果与评估上下文，支持 trace_id 关联排障 |
| Redis 缓存监控 | Redis 缓存监控，只读展示 Redis INFO、DBSIZE 与慢日志数据，不执行写操作 |

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
│   │   ├── cmd/server/             # 入口 (main.go, wiring_ent.go 依赖装配)
│   │   ├── configs/                # 配置文件 (YAML)
│   │   └── internal/               # 业务核心（data/service/server）
│   ├── pkg/                        # 公共包
│   │   ├── scripting/              # 多语言脚本引擎（Lua + JavaScript）
│   │   ├── oss/                    # 对象存储（MinIO）
│   │   ├── eventbus/               # 事件总线
│   │   └── ...                     # 其他工具包
│   ├── scripts/                    # 部署与备份脚本（env/docker/deploy/backup）
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

## 社区与贡献

欢迎参与 GoWind Admin 的建设。以下文档说明如何贡献代码、报告问题与反馈安全漏洞：

- [贡献指南](./CONTRIBUTING.md) —— 开发环境、代码生成约定、提交规范与 PR 流程
- [行为准则](./.github/CODE_OF_CONDUCT.md) —— 社区互动预期
- [安全策略](./SECURITY.md) —— 漏洞上报流程与覆盖范围
- [更新日志](./CHANGELOG.md) —— 版本变更记录
- Issue 模板：[Bug 报告](./.github/ISSUE_TEMPLATE/bug_report.md) · [功能请求](./.github/ISSUE_TEMPLATE/feature_request.md)
- [PR 模板](./.github/PULL_REQUEST_TEMPLATE.md)

## 联系我们

- 微信个人号：`yang_lin_bo`（备注：`go-wind-admin`）
- 掘金专栏：[go-wind-admin](https://juejin.cn/column/7541283508041826367)

## 致谢

[![JetBrains](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)

感谢 JetBrains 提供免费的 GoLand & WebStorm 开源授权。
