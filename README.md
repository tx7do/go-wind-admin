# GoWind Admin｜风行，开箱即用的企业级前后端一体中后台脚手架

> **让中后台开发如风般自由 — GoWind Admin**

风行（GoWind Admin）是一款开箱即用的企业级Golang全栈后台管理系统的脚手架。

系统后端基于GO微服务框架[go-kratos](https://go-kratos.dev/)，前端有Vue3和React两个版本，兼顾微服务的扩展性与单体部署的便捷性。

尽管依托微服务框架设计，但系统前后端均支持单体架构模式开发与部署，灵活适配不同团队规模及项目复杂度需求，平衡灵活性与易用性。

产品具备上手简易、功能完备的核心优势，依托风行对企业级场景的深度适配能力，可助力开发者快速落地各类企业级管理系统项目，大幅提升开发效率。

[English](./README.en-US.md) | **中文** | [日本語](./README.ja-JP.md)

## 演示地址

> 前端地址：<https://demo.admin.gowind.cloud>
>
> 后端Swagger地址：<https://api.demo.admin.gowind.cloud/docs/>
>
> 默认账号密码: `admin` / `admin`

## 风行·核心技术栈

秉持高效、稳定、可扩展的技术选型理念，系统核心技术栈如下：

- 后端基于 [Golang](https://go.dev/) + [go-kratos](https://go-kratos.dev/) + [wire](https://github.com/google/wire) + [ent](https://entgo.io/docs/getting-started/)
- 前端基于 [Vue](https://vuejs.org/) + [TypeScript](https://www.typescriptlang.org/) + [Ant Design Vue](https://antdv.com/) + [Vben Admin](https://doc.vben.pro/)

## 风行·快速上手指南

### 后端

一键安装`golang`和`docker`等前置依赖：

```bash
# Ubuntu
./backend/script/prepare_ubuntu.sh

# Centos
./backend/script/prepare_centos.sh

# Rocky
./backend/script/prepare_rocky.sh

# Windows
./backend/script/env/install_windows_dev.ps1

# MacOS
./backend/script/prepare_macos.sh
```

一键安装三方组件和`go-wind-admin`服务：

```bash
./backend/script/docker_compose_install.sh
```

### 前端

#### 1. 安装 Node.js（npm 随 Node.js 自带）：

访问Node.js官方下载页：<https://nodejs.org/>，下载对应系统（Windows/macOS/Linux）的LTS稳定版本并安装。

安装完成后，打开终端/命令提示符，输入以下命令验证安装成功：

```bash
node -v  # 输出Node.js版本号即成功
npm -v   # 输出npm版本号即成功
```

#### 2. 安装 pnpm：

```bash
npm install -g pnpm
```

#### 3. 启动前端服务：

进入 frontend 目录，执行以下命令，完成前端依赖安装、编译并启动开发模式：

```bash
pnpm install
pnpm dev
```

### 访问测试

- 前端地址：<http://localhost:5666>， 登录账号：`admin`，密码：`admin`
- 后端文档地址：<http://localhost:7788/docs/openapi.yaml>

## 风行·核心功能列表

| 功能   | 说明                                                                       |
|------|--------------------------------------------------------------------------|
| 用户管理 | 管理和查询用户，支持高级查询和按部门联动用户，用户可禁用/启用、设置/取消主管、重置密码、配置多角色、多部门和上级主管、一键登录指定用户等功能。 |
| 租户管理 | 管理租户，新增租户后自动初始化租户部门、默认角色和管理员。支持配置套餐、禁用/启用、一键登录租户管理员功能。                   |
| 角色管理 | 管理角色和角色分组，支持按角色联动用户，设置菜单和数据权限，批量添加和移除员工。                                 |
| 权限管理 | 管理权限分组、菜单、权限点，支持树形列表展示。                                                  |
| 组织管理 | 管理组织，支持树形列表展示。                                                           |
| 部门管理 | 管理部门，支持树形列表展示。                                                           |
| 职位管理 | 用户职务管理，职务可作为用户的一个标签。                                                           |
| 接口管理 | 管理接口，支持接口同步功能，主要用于新增权限点时选择接口，支持树形列表展示、操作日志请求参数和响应结果配置。                   |
| 菜单管理 | 配置系统菜单，操作权限，按钮权限标识等，包括目录、菜单、按钮。                                                                  |
| 字典管理 | 管理数据字典大类及其小类，支持按字典大类联动字典小类、服务端多列排序、数据导入和导出。                              |
| 任务调度 | 管理和查看任务及其任务运行日志，支持任务新增、修改、删除、启动、暂停、立即执行。                                 |
| 文件管理 | 管理文件上传，支持文件查询、上传到OSS或本地、下载、复制文件地址、删除文件、图片支持查看大图功能。                       |
| 消息分类 | 管理消息分类，支持2级自定义消息分类，用于消息管理消息分类选择。                                         |
| 消息管理 | 管理消息，支持发送指定用户消息，可查看用户是否已读和已读时间。                                          |
| 站内信  | 站内消息管理，支持消息详细查看、删除、标为已读、全部已读功能。                                          |
| 个人中心 | 个人信息展示和修改，查看最后登录信息，密码修改等功能。                                              |
| 缓存管理 | 缓存列表查询，支持根据缓存键清除缓存。                                                      |
| 登录日志 | 登录日志列表查询，记录用户登录成功和失败日志，支持IP归属地记录。                                        |
| 操作日志 | 操作日志列表查询，记录用户操作正常和异常日志，支持IP归属地记录，查看操作日志详情。                               |

## 风行·后台截图展示

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

## [感谢JetBrains提供的免费GoLand & WebStorm](https://jb.gg/OpenSource)

[![avatar](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)
