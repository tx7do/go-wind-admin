<div align="center">

<img src="docs/brand/vortex-tile.svg" width="120" alt="GoWind Admin" />

# GoWind Admin

**Out-of-the-box, enterprise-grade full-stack admin scaffold (Go backend + one of three frontends)**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vuedotjs)](https://vuejs.org/)
[![React](https://img.shields.io/badge/React-19.x-61DAFB?logo=react)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

[English](./README.en-US.md) | [中文](./README.md) | [日本語](./README.ja-JP.md)

</div>

---

## Project Highlights

- **Three frontends, pick one — not three in one**: `Vue3 Vben` (Ant Design Vue), `Vue3 Element Plus` and `React19 Antd` are **three parallel implementations of the same backend**, so that teams on different stacks each get the one they are fluent in — **one team takes one, one deployment runs one**. Each has its own package root, its own deploy scripts, and they share no dependencies; once you choose, the other two directories can be deleted (steps in [docs/adopt-one-frontend.md](./docs/adopt-one-frontend.md))
- **Enterprise-grade RBAC**: Multi-tenant, multi-role, multi-department, menu / button / data-level permission control (switchable policy engine: Casbin / OPA)
- **Security & MLPS Compliance**: Designed against China MLPS 2.0 (Level 2/3) technical requirements — 180-day audit log retention & archiving, password policy trio, TOTP MFA, application-layer password encryption, dynamic RBAC and tenant isolation, scheduled backup rotation. See [Security & Compliance](#security--compliance-mlps-20)
- **Microservice + Monolith**: Built on the go-kratos microservice framework, yet supports monolith-mode development and deployment — flexible for any team size
- **Full-stack Code Generation**: Protobuf → Go API / TypeScript clients, Ent Schema → ORM, one-click CRUD scaffolding; companion desktop GUI generator and CLI ([go-wind-toolkit](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind-uiapp), see [Companion Tools](#companion-tools))
- **Production-ready**: JWT auth, SSE push, async task scheduling, Swagger docs, one-click Docker deployment

### Why three frontends

**Because teams work in different stacks — not because one team needs React and Vue at the same time.**

One backend, one API contract, three frontend implementations: React teams take `react`, Vue teams take `vue-vben` or `vue-element`. Nobody has to switch stacks just to adopt this scaffold.

Keeping all three usable is a cost borne **upstream**; as an adopter you maintain only the one you picked and delete the other two directories (what to adjust: [docs/adopt-one-frontend.md](./docs/adopt-one-frontend.md)).

---

## Demo

Three URLs, three parallel demos of the same backend capabilities — **open each, compare, you will want only one**:

| Frontend Edition | Demo |
|------------------|------|
| Vue3 Vben | <https://vben.admin.gowind.cloud> |
| Vue3 Element Plus | <https://ele.admin.gowind.cloud> |
| React | <https://react.admin.gowind.cloud> |

- Backend Swagger: <https://api.demo.admin.gowind.cloud/docs/>
- Default account / password: `admin` / `Abcd@1234`

---

## Tech Stack

<table>
<tr><th>Layer</th><th>Technologies</th></tr>
<tr><td><strong>Backend Framework</strong></td><td><code>Golang</code> · <code>go-kratos v2</code> · <code>Protobuf / Buf</code></td></tr>
<tr><td><strong>ORM</strong></td><td><code>Ent</code> (primary) · <code>GORM</code> (auxiliary) · <code>MySQL</code> · <code>PostgreSQL</code></td></tr>
<tr><td><strong>Middleware</strong></td><td><code>Redis 8.0+</code> · <code>MinIO</code> (S3-compatible object storage)</td></tr>
<tr><td><strong>Authentication & Authorization</strong></td><td><code>JWT</code> · <code>Casbin</code> · <code>OPA</code></td></tr>
<tr><td><strong>Realtime</strong></td><td><code>SSE</code> (server push) · <code>Asynq</code> (async tasks)</td></tr>
<tr><td><strong>Scripting Engine</strong></td><td><code>go-scripts</code> · <code>Lua</code> (gopher-lua) · <code>JavaScript</code> (goja) · multi-language hook plugin system</td></tr>
<tr><td><strong>Frontend</strong></td><td><strong>Pick one</strong> — the three rows below are parallel options, not something to adopt together</td></tr>
<tr><td><strong>Vue Vben Edition</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Ant Design Vue</code> · <code>Vben Admin</code></td></tr>
<tr><td><strong>Vue Element Edition</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Element Plus</code> (lightweight pure edition)</td></tr>
<tr><td><strong>React Edition</strong></td><td><code>React 19</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Zustand</code> · <code>Ant Design V6</code> (no UMI)</td></tr>
<tr><td><strong>Deployment & Ops</strong></td><td><code>Docker</code> · <code>Docker Compose</code> · <code>PM2</code> · <code>Swagger UI</code></td></tr>
</table>

---

## Security & Compliance (MLPS 2.0)

Security capabilities are designed with reference to the technical requirements of China's Cybersecurity Multi-Level Protection Scheme 2.0 (MLPS 2.0, Level 2/3), ready out of the box for privacy-conscious private deployment scenarios:

| Requirement | Implementation |
|------------|----------------|
| **Security Audit** | Full coverage of six audit log types: login / operation / API / data access / permission change / policy evaluation, recording the client IP (login / operation / API additionally resolve geolocation) and the `X-Request-ID` request id sent by the frontend. Daily scheduled archiving via asynq: 180-day in-database retention (`AUDIT_RETENTION_DAYS`, adjustable); expired rows are exported to JSONL archive files for long-term traceability |
| **Identity Authentication** | Password complexity (≥8 chars, at least 3 of 4 character classes), password history reuse check (last 3 by default), password validity period (90 days by default) — thresholds are adjusted via the "Parameter Management" platform parameters (built-in keys seeded at startup; environment-variable configuration is deprecated); TOTP multi-factor authentication (MFA); image captcha; Redis login failure rate limiting (IP + username dual dimensions); configurable login restriction policies |
| **Access Control** | Dynamic RBAC engine (switchable policy engine: Casbin / OPA); role–permission–API mappings stored in the database with instant hot-reload on changes; menu / button level permission control, plus role-scoped row-level data scope (V1 pilot: position table) and field-level permissions (V1 pilot: user table, blacklisted fields pruned from responses); every authorization decision is logged to policy evaluation logs for traceability |
| **Multi-Tenant Isolation** | Compile-time ent Privacy data isolation: read queries get automatic tenant filtering; Create guards against forged tenants, Update / Delete inject tenant predicates (cross-tenant mutations match 0 rows); tenant requests are fail-closed validated against the Api table by `(path, method)`; plan module whitelists and expiry read-only policy |
| **Data Confidentiality** | Login passwords encrypted at the application layer (AES) in transit and bcrypt-hashed at rest; sensitive task configs encrypted at rest with AES-256-GCM (transparent via Ent hooks); JWT RS256 asymmetric signing; refresh token in HttpOnly Cookie; transport-layer TLS enabled at deployment (backend `server.rest.tls` config, or nginx / load balancer termination) |
| **Data Backup & Recovery** | [`scripts/backup/pg_backup.sh`](./backend/scripts/backup/pg_backup.sh) scheduled full backups (pg_dump, 30 copies auto-rotation by default), Docker container / local direct-connect dual modes, with recovery documentation |
| **Frontend Security** | Each of the three frontends ships its own `scripts/deploy/nginx.conf`, setting X-Frame-Options / HSTS / Content-Security-Policy response headers in production; react and vue-element additionally inject a CSP `<meta>` into `index.html` at build time (inline scripts allowlisted by sha256), so one layer survives a different web server |

> **Note**: MLPS evaluation covers more than technical requirements — management policies, physical environment, personnel organization, etc. This project covers the technical measures portion, providing direct support for evaluation preparation in private deployments, but it does not replace the full MLPS certification process.

---

## Quick Start

### Environment Requirements

| Tool | Version |
|------|---------|
| Go | 1.26+ (follow `backend/go.mod`) |
| Node.js | `^20.19.0 \|\| >=22.12.0` — the intersection of the three `engines` fields is set by vue-element (vue-vben only asks `>=20.10.0`, react declares none). **21.x is not covered**, nor is anything below 20.19 |
| pnpm | `>= 9.12.0` (floor from vue-vben's `engines.pnpm`). vue-vben additionally pins `pnpm@11.18.0` via `packageManager`: enable corepack (`corepack enable`) to switch automatically, otherwise install 11.x yourself |
| Docker | 20.0+ |

### Environment Scripts

- Linux / macOS Development: `scripts/env/install_unix_dev.sh`
- Linux / macOS Production: `scripts/env/install_unix_prod.sh`
- Windows Development: `scripts/env/install_windows_dev.ps1`

### Docker Deployment Modes

- **full_deploy**: Starts middleware + backend application together, suitable for one-click demos or production deployment.
- **libs_only (Recommended for development)**: Starts middleware only; run the application locally in your IDE for debugging.

### Backend Startup

> Backend commands go through the `gow` CLI (install: `go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest`, see [Companion Tools](#companion-tools)).

**Linux / macOS:**

```shell
# All commands below run inside backend/ (scripts/ lives there)
cd backend

# Grant script execution permissions
# scripts/ nests three levels; a glob misses env/lib and deploy/sse, so use find
find ./scripts -name '*.sh' -exec chmod +x {} +

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

**Windows (PowerShell as Administrator):**

```powershell
# All commands below run inside backend/ (scripts/ lives there)
cd backend

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

All three live under the `frontend/admin` directory — **pick one**, install it the usual way, and ignore the other two:

| Frontend Edition | Directory | Command | Port |
|------------------|-----------|---------|------|
| React | `frontend/admin/react` | `pnpm dev` | 5888 |
| Vue Element | `frontend/admin/vue-element` | `pnpm dev` | 5777 |
| Vue Vben | `frontend/admin/vue-vben` | `pnpm dev:antd` | 5666 |

```shell
# Pick ONE: cd into that frontend first, then install and start.
# There is no package.json at the repo root or under frontend/admin/,
# so running `pnpm install` there only fails with ENOENT.
cd frontend/admin/react
pnpm install
pnpm dev                    # port 5888

# The other two ends:
cd frontend/admin/vue-element && pnpm install && pnpm dev            # port 5777
cd frontend/admin/vue-vben   && pnpm install && pnpm dev:antd        # port 5666
```

> vue-vben is itself a pnpm workspace (`pnpm-workspace.yaml` + `apps/` + `packages/`), so dependencies must be installed at **its** root; `pnpm dev:antd` then selects the `@vben/web-antd` app from the workspace. Installing inside `apps/admin` bypasses the catalog version pins.

---

## Features

> All list pages (business data and audit logs) support "Export" with paginated aggregation under the current filters, in CSV or XLSX format (up to 10,000 rows).

### Organization & Permissions

| Feature | Description |
|---------|-------------|
| User Management | Manage and query users with advanced search and department-linked users; enable/disable users, set/unset manager, reset password, configure multiple roles / departments / managers, one-click login as a specified user. |
| Tenant Management | Manage tenants. Adding a tenant auto-initializes its departments, default roles, and admin. Supports plan configuration, enable/disable, and one-click login as the tenant admin. |
| Plan & Quota Management | Manage tenant subscription plans and resource quotas (module whitelists, usage limits, etc.); CRUD for plans and quota items. |
| Role Management | Manage roles and role groups; user selection by role; menu grants, data-scope configuration (five levels / custom organization-unit sets), and field-level permissions (blacklisted field sets); batch add/remove employees. |
| Permission Management | Manage permission groups, menus, and permission points with tree-view listing. |
| Organization Management | Manage organizations with tree-view listing. |
| Position Management | Manage user positions; positions can serve as user labels. Excel import is supported: client-side template download, row-by-row processing through the existing create API with row-level error reporting, and the organization column is resolved by exact organization-name match. |
| Menu Management | Configure system menus, operation permissions, and button permission identifiers — directories, menus, and buttons. Menu synchronization (available on all three frontends) supports two modes: transactional truncate-rebuild, or incremental merge, which matches existing entries by full path and updates them in place while preserving existing menu IDs and role grants. |

### System Features

| Feature | Description |
|---------|-------------|
| API Management | Manage APIs with API synchronization (mainly for picking endpoints when adding permission points), tree-view listing, and operation-log request/response configuration. |
| Dictionary Management | Manage dictionary categories and entries; category-linked entries, server-side multi-column sorting, import/export. |
| Task Scheduler | Manage tasks and task run logs; create, update, delete, start, pause, and run immediately. |
| File Management | Manage uploads: search, upload to OSS or local storage, download, copy file address, delete, and image preview (large view). |
| Login Policy | Manage login restriction policies; configure restriction type, method, value, and reason for target users. |
| Account Login | Sign in with username / email / phone number, combinable with image captcha, login policies, and TOTP MFA. |
| Multi-Factor Authentication (MFA) | TOTP-based MFA: login challenge, personal binding management, and admin rescue reset of a user's MFA. |
| Password Recovery | Reset password via a code sent to the bound email: 10-minute single-use code, all sessions revoked on success, silent handling prevents user enumeration. |
| Notification Channels | Manage notification channels (EMAIL / SMTP); encrypted password storage, masked in lists; enable/disable and test sending. |
| Server Monitoring | Read-only view of runtime metrics (CPU cores, memory, goroutines, uptime, etc.) with auto refresh. |
| Script System | Script-based plugin system (Lua / JavaScript, database as the source of truth, admin-UI changes take effect immediately): entity lifecycle hooks (before can veto / after is async), scheduled tasks (asynq), HTTP egress (domain allowlist, fail-closed), test runs and execution logs. See [docs/script_system.md](./docs/script_system.md). |
| Parameter Management | Manage platform-wide system parameters as key/value pairs (distinct from business dictionaries); built-in parameters are seeded at startup and cannot be deleted; reads go through a server-side caching accessor, and in multi-instance deployments changes are broadcast over Redis pub/sub to invalidate every instance's cache. |
| Machine Credentials (AK/SK) | Tenant-scoped AccessKey / SecretKey management: the Secret is shown exactly once at creation; enable/disable, delete, and secret rotation are supported (rotation immediately invalidates the previous Secret). An AK / Secret pair can be exchanged at the token-exchange endpoint for a tenant-scoped machine JWT (machine role, access token only); the exchange endpoint applies per-IP + per-AK failure rate limiting. |
| Language Management | Manage supported languages: name, code, native name, enabled and default status. |

### Messaging & Logs

| Feature | Description |
|---------|-------------|
| Message Categories | Manage message categories (2-level custom categories) used in message management. |
| Message Management | Send by scope (all users / specified users) with revocation; broadcast fan-out runs on an async task queue (resumable, idempotent); view read status and read time. |
| Internal Mail | Manage internal messages: view details, delete, mark as read, mark all as read. |
| Login Logs | Query login logs for successful and failed logins; supports IP geolocation. |
| Operation Logs | Query operation logs for normal and abnormal operations; supports IP geolocation, resource identification, and detail view. |
| API Logs | Query API audit logs recording operator, path, method, and success status; supports IP geolocation. |
| Data Logs | Query data access audit logs; SQL is lexically masked, with involved table names and data categories extracted automatically. |
| Permission Logs | Query permission change audit logs recording operator, target object, and reason, with request snapshots retained. |
| Policy Evaluation Logs | Query policy evaluation logs recording each authorization decision with its evaluation context; supports trace_id correlation for troubleshooting. |
| Redis Cache Monitor | Read-only Redis monitoring displaying INFO, DBSIZE, and slowlog data; performs no write operations. |

### Personal Center

| Feature | Description |
|---------|-------------|
| Personal Center | View and edit personal info, check last-login info, change password, bind / rebind email (verification code), etc. |

---

## Project Structure

```
go-wind-admin/
├── backend/                        # Backend project
│   ├── api/                        # Protobuf API definitions & generated code
│   │   ├── protos/                 # .proto sources (layered by domain)
│   │   └── gen/go/                 # Go code generated by buf
│   ├── app/admin/service/          # Admin service application
│   │   ├── cmd/server/             # Entry point (main.go, wiring_ent.go DI assembly)
│   │   ├── configs/                # Configuration files (YAML)
│   │   └── internal/               # Business core (data/service/server)
│   ├── pkg/                        # Shared packages
│   │   ├── scripting/              # Multi-language scripting engine (Lua + JavaScript)
│   │   ├── oss/                    # Object storage (MinIO)
│   │   ├── eventbus/               # Event bus
│   │   └── ...                     # Other utility packages
│   ├── scripts/                    # Deployment & backup scripts (env/docker/deploy/backup)
│   └── sql/                        # Demo data SQL (default data is seeded automatically at service startup)
├── frontend/admin/                 # Frontend projects (pick one — you only maintain that one)
│   ├── react/                      # React 19 + Ant Design V6
│   ├── vue-element/                # Vue 3 + Element Plus
│   └── vue-vben/                   # Vue 3 + Ant Design Vue + Vben Admin
└── docs/                           # Project documentation
```

---

## Screenshots

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

## Companion Tools

- **[go-wind-toolkit / gowind-uiapp](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind-uiapp)** — Cross-platform desktop code generator (Go + Wails). Import SQL or connect to your database (MySQL / PostgreSQL / SQLite / SQL Server / Oracle) to generate server-side and frontend code from gRPC / RESTful templates, including simple forms. Also ships a non-interactive, JSON-output CLI (`gowind-cli`) for scripts and AI agents.
- **[gow — GoWind CLI](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind)** — The recommended command-line entry for this project: `gow run admin` to run the service, `gow ent` / `gow api` for code generation, `gow generate` to scaffold CRUD microservices from a database DSN, and `gow extract` for progressive microservice extraction. Run it under `backend/`; it discovers `app/*/service` automatically. Prefer it over the Makefile for daily development.

## Community & Contribution

Welcome to join the GoWind Admin community. The documents below describe how to contribute code, report issues, and disclose security vulnerabilities:

- [Contributing Guide](./CONTRIBUTING.md) — development environment, codegen conventions, commit conventions and the PR process
- [Code of Conduct](./.github/CODE_OF_CONDUCT.md) — community interaction expectations
- [Security Policy](./SECURITY.md) — vulnerability disclosure process and scope
- [Changelog](./CHANGELOG.md) — release notes
- Issue templates: [Bug Report](./.github/ISSUE_TEMPLATE/bug_report.md) · [Feature Request](./.github/ISSUE_TEMPLATE/feature_request.md)
- [PR Template](./.github/PULL_REQUEST_TEMPLATE.md)

## Contact

- WeChat: `yang_lin_bo` (note: `go-wind-admin`)
- Juejin column: [go-wind-admin](https://juejin.cn/column/7541283508041826367)

## Acknowledgements

[![JetBrains](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)

Thanks to JetBrains for providing free GoLand & WebStorm open-source licenses.
