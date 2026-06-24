# GoWindAdmin (GoWind Admin System)

> **Efficiently build enterprise-grade admin systems, making development as smooth as the wind.**

GoWindAdmin is an out-of-the-box enterprise-grade Golang full-stack admin system scaffold.

The backend is based on the GO microservice framework [go-kratos](https://go-kratos.dev/), and the frontend offers three versions: `Vue3 Vben`, `Vue3 Element Plus`, and `React Antd`, balancing microservice scalability with monolithic deployment convenience.

Although built on a microservice framework, both the frontend and backend support monolithic architecture for development and deployment, flexibly adapting to different team sizes and project complexity requirements.

Easy to get started, feature-rich, deeply adapted to enterprise scenarios, helping developers quickly deliver various enterprise management system projects and significantly improve development efficiency.

**English** | [中文](./README.md) | [日本語](./README.ja-JP.md)

## Demo

> Demo Portal: <https://demo.admin.gowind.cloud>
>
> Vue3 Vben Demo: <https://vben.admin.gowind.cloud>
> Vue3 Element Plus Demo: <https://ele.admin.gowind.cloud>
> React Demo: <https://react.admin.gowind.cloud>
>
> Backend Swagger: <https://api.demo.admin.gowind.cloud/docs/>
>
> Default account/password: `admin` / `admin`

## Tech Stack

Adhering to the philosophy of efficient, stable, and scalable technology selection:

- **Backend**: `Golang`, `go-kratos`, `Wire`, `Ent ORM` / `Gorm`, `MySQL`, `Redis`, `Docker`
- **Common Infrastructure**: `JWT Authentication`, `Casbin` / `OPA` / `Zanzibar` Authorization, `SSE Push`, `Swagger API Docs`
- **Scripting Engine**: `go-scripts` · `Lua` (gopher-lua) · `JavaScript` (goja) · Multi-language Hook plugin system
- **Vue Vben Edition**: `Vue3` + `TypeScript` + `Vite` + `Ant Design Vue` + `Vben Admin`
- **Vue Element Edition**: `Vue3` + `TypeScript` + `Vite` + `Element Plus` (lightweight pure version)
- **React Edition**: `React19` + `TypeScript` + `Vite` + `React Router` + `Zustand` + `Ant Design V6` + `@ant-design/pro-components` (**No UMI framework**)

## Quick Start

### Environment Scripts

- Linux / macOS Development: `scripts/env/install_unix_dev.sh`
- Linux / macOS Production: `scripts/env/install_unix_prod.sh`
- Windows Development: `scripts/env/install_windows_dev.ps1`

### Docker Deployment Modes

- **full_deploy**: Starts middleware + backend application, suitable for one-click demo or production deployment.
- **libs_only (Recommended)**: Starts middleware only, run application locally in IDE for daily development.

### Backend Startup

#### Linux / macOS

```shell
# Grant script execution permissions
chmod +x scripts/**/*.sh

# Development (Recommended)
./scripts/env/install_unix_dev.sh
./scripts/docker/libs_only.sh
gow run admin

# Production
./scripts/env/install_unix_prod.sh
./scripts/docker/full_deploy.sh

# PM2 Process Management (Advanced Production)
./scripts/deploy/pm2_service.sh
```

#### Windows (PowerShell Administrator)

```powershell
# Allow script execution (only needed once)
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# Initialize environment
.\scripts\env\install_windows_dev.ps1

# Local development
.\scripts\docker\libs_only.ps1
gow run admin

# One-click full deployment
.\scripts\docker\full_deploy.ps1
```

### Frontend Startup

Frontend projects are located in the `frontend/admin` directory. Dependency installation is unified, but startup commands differ:

- React: Directory `frontend/admin/react`, command `pnpm dev`, port: `7000`
- Vue Element: Directory `frontend/admin/vue-element`, command `pnpm dev`, port: `3000`
- Vue Vben: Directory `frontend/admin/vue-vben`, command `pnpm dev:antd`, port: `5666`

```shell
# Install dependencies
pnpm install

# React Edition
cd frontend/admin/react
pnpm dev

# Vue3 Element Edition
cd frontend/admin/vue-element
pnpm dev

# Vue3 Vben Edition
cd frontend/admin/vue-vben
pnpm dev:antd
```

## Features

| Feature                 | Description                                                                                                                                                                                                             |
|-------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| User Management         | Manage and query users, support advanced search and department-linked users; enable/disable users, set/unset manager, reset password, configure multiple roles/departments/managers, one-click login as specified user. |
| Tenant Management       | Manage tenants. Adding a tenant auto-initializes tenant departments, default roles, and admin. Support plan configuration, enable/disable, one-click login as tenant admin.                                             |
| Role Management         | Manage roles and role groups; support user selection by role, set menu and data permissions, batch add/remove employees.                                                                                                |
| Permission Management   | Manage permission groups, menus, and permission points; supports tree-view listing.                                                                                                                                                                                                 |
| Organization Management | Manage organizations with tree-view listing.                                                                                                                                                                            |
| Department Management   | Manage departments with tree-view listing.                                                                                                                                                                              |
| Permission Management   | Manage permission groups, menus, and permission points; supports tree-view listing.                                                                                                                                    |
| Position Management     | Manage user positions; positions can serve as user labels.                                                                                                                               |
| API Management          | Manage APIs, support API synchronization (mainly for selecting interfaces when adding permission points), tree-view listing, configure operation log request parameters and responses.                                  |
| Menu Management         | Configure system menus, operation and button permission identifiers, including directories, menus, and buttons.                                                                                                                                                                                         |
| Dictionary Management   | Manage dictionary categories and entries, support category-linked entries, server-side multi-column sorting, data import/export.                                                                                        |
| Task Scheduler          | Manage tasks and task run logs; support create, update, delete, start, pause, and run immediately.                                                                                                                      |
| File Management         | Manage file uploads, search files, upload to OSS or local storage, download, copy file address, delete files, support image preview (large view).                                                                       |
| Message Categories      | Manage message categories (2-level custom categories) for message management category selection.                                                                                                                        |
| Message Management      | Manage messages, send messages to specified users, view read status and read time.                                                                                                                                      |
| Internal Mail           | Manage internal messages, view details, delete, mark as read, mark all as read.                                                                                                                                         |
| Personal Center         | View and edit personal info, view last login info, change password, etc.                                                                                                                                                |
| Cache Management        | Query cache list and clear cache by key.                                                                                                                                                                                |
| Login Logs              | Query login logs for successful and failed logins; supports IP geolocation.                                                                                                                                             |
| Operation Logs          | Query operation logs for normal and abnormal operations; supports IP geolocation and viewing operation details.                                                                                                         |

## Backend Screenshots

<table>
    <tr>
        <td><img src="./docs/images/admin_login_page.png" alt="Backend user login page"/></td>
        <td><img src="./docs/images/admin_dashboard.png" alt="Backend dashboard"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_user_list.png" alt="Backend user list page"/></td>
        <td><img src="./docs/images/admin_user_create.png" alt="Backend create user page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_tenant_list.png" alt="Backend tenant list page"/></td>
        <td><img src="./docs/images/admin_tenant_create.png" alt="Backend create tenant page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_org_unit_list.png" alt="Organization unit list page"/></td>
        <td><img src="./docs/images/admin_org_unit_create.png" alt="Create organization unit page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_position_list.png" alt="Backend position list page"/></td>
        <td><img src="./docs/images/admin_position_create.png" alt="Backend create position page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_role_list.png" alt="Backend role list page"/></td>
        <td><img src="./docs/images/admin_role_create.png" alt="Backend create role page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_permission_list.png" alt="Backend permission list page"/></td>
        <td><img src="./docs/images/admin_permission_create.png" alt="Backend create permission page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_menu_list.png" alt="Backend menu list page"/></td>
        <td><img src="./docs/images/admin_menu_create.png" alt="Backend create menu page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_task_list.png" alt="Backend scheduled task list page"/></td>
        <td><img src="./docs/images/admin_task_create.png" alt="Backend create scheduled task page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_dict_list.png" alt="Backend dictionary list page"/></td>
        <td><img src="./docs/images/admin_dict_entry_create.png" alt="Backend create dictionary entry page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_internal_message_list.png" alt="Backend internal message list page"/></td>
        <td><img src="./docs/images/admin_internal_message_publish.png" alt="Backend publish internal message page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_login_policy_list.png" alt="Login policy list page"/></td>
        <td><img src="./docs/images/admin_login_policy_create.png" alt="Login policy create page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_login_audit_log_list.png" alt="Backend login audit log page"/></td>
        <td><img src="./docs/images/admin_api_audit_log_list.png" alt="Backend operation audit log page"/></td>
    </tr>
    <tr>
        <td><img src="./docs/images/admin_api_list.png" alt="API list page"/></td>
        <td><img src="./docs/images/api_swagger_ui.png" alt="Backend built-in Swagger UI page"/></td>
    </tr>
</table>

## Contact

- WeChat: `yang_lin_bo` (note: `go-wind-admin`)
- Juejin column: [go-wind-admin](https://juejin.cn/column/7541283508041826367)

## Thanks to JetBrains for providing free GoLand & WebStorm

[![avatar](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)
