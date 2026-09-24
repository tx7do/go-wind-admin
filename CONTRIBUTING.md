# 贡献指南

感谢你对 GoWind Admin 的关注！本文档说明如何向本项目提交代码、文档与问题反馈。在开始之前，请先阅读 [README.md](./README.md) 了解项目定位与技术栈。

## 行为准则

参与本项目即代表你同意遵守 [行为准则](./.github/CODE_OF_CONDUCT.md)。请在讨论、Issue 与 PR 中保持友善与专业。

## 快速上手

1. **Fork 并克隆仓库**

   ```bash
   git clone https://github.com/<your-name>/go-wind-admin.git
   cd go-wind-admin
   git remote add upstream https://github.com/<org>/go-wind-admin.git
   ```

2. **搭建开发环境**

   - 后端：参考 [docs/backend_development_environment_preparation.md](./docs/backend_development_environment_preparation.md)
   - 前端：参考 [docs/frontend_development_environment_preparation.md](./docs/frontend_development_environment_preparation.md)
   - Windows 本地启动：参考 [docs/windows-startup-guide.md](./docs/windows-startup-guide.md)

3. **创建分支**

   请基于最新的 `main` 创建特性分支，不要直接在 `main` 上开发：

   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feat/your-feature
   ```

## 开发约定

### 代码生成

本项目大量依赖代码生成，**请不要手工编辑生成产物**：

| 产物 | 生成命令 | 说明 |
|------|---------|------|
| Go API / HTTP / gRPC / 校验代码 | `gow api` 或 `make api` | 由 `backend/api/protos/` 下的 proto 生成至 `backend/api/gen/go/` |
| OpenAPI 文档 | `make openapi` | 产物写到 `backend/app/admin/service/cmd/server/assets/openapi.yaml`，**打进二进制**（用途见下方提示） |
| TypeScript HTTP 客户端 | `make ts` | 一条命令依次跑三个 buf 模板，同时生成到三端各自的 `src/api/generated/` |
| Ent ORM 代码 | `gow ent admin` 或 `make ent` | 由 `app/admin/service/internal/data/ent/schema/` 下的 schema 生成 |
| 新模块接入装配 | `make register ENTITY=xxx` | 把仓储/服务构造行与路由注册注入手写的 `cmd/server/wiring_ent.go` 和 `internal/server/rest_server.go`（靠 `// register:*` 锚点行定位，幂等可重跑） |

`gow` 与 `make` 产出完全一致，选 `gow` 的理由很实际：Windows 没有原生 make，且本仓 Makefile 是嵌套结构、需要跨目录 cd（`gow` 会自动发现 `app/*/service`，见根 [AGENTS.md](./AGENTS.md)）。`make ts` / `make openapi` 目前只有 Makefile 入口。

修改 `.proto` 或 `ent/schema` 后请重新生成并**连同产物一并提交**。三端 TS 是同一条 `make ts` 的产物、本来就同时更新，不存在"只提交一端"的选项。

> `make register` 只覆盖 CRUD 生成器的标准形态（`New<X>Repo(ctx, entClient)` / `New<X>Service(ctx, <x>Repo)`）。依赖更多的模块请手工补那两行，别指望工具替你写完 wiring。

> 漏跑 `make openapi` 的后果很费解：管理页「接口同步」读的是**打进二进制的** `openapi.yaml`，所以它会告诉你同步成功了，新端点却依旧不在 Api 表里；接口缺表项时租户闸门按 fail-closed 返回 403，而路由本身看起来完全正常。正确顺序是 `make api && make openapi` 后重启进程，再触发同步。详见根 [AGENTS.md](./AGENTS.md) 与 [docs/tenant_isolation.md](./docs/tenant_isolation.md)。

> **提交三端生成产物 ≠ 三端都要写页面。** 业务页面 / 表单 / hook 属人工移植，PR **只需落一个端**即可进入评审（本仓以 react 为行为基准，见根 [AGENTS.md](./AGENTS.md)），另外两端由维护者后续补齐。愿意顺手多写一端的，请在 PR 描述里注明。
>
> 另一种方向：你若只是想维护自己选中的那一端、准备删掉另外两端，见 [docs/adopt-one-frontend.md](./docs/adopt-one-frontend.md)。

### 输入校验

业务接口请在 proto 中用 protoc-gen-validate 的 `(validate.rules)` 声明字段约束（配套 `import "validate/validate.proto";`），避免在 service 层手写判空：

```proto
string email = 11 [(validate.rules).string.max_len = 254];  // 见 api/protos/authentication/service/v1/authentication.proto:149
```

生成的 `*.pb.validate.go` 由 REST 服务在**鉴权之前**调用（kratos `validate.Validator()` 中间件，`backend/app/admin/service/internal/server/rest_server.go:77`），所以白名单路由（登录 / 刷新令牌 / MFA 验证）的入参同样被校验——这几个接口恰恰最没人管却最该管。

> 不要写成 `buf.validate.field`：那是 buf 的新一代 protovalidate 扩展，本仓 `api/buf.yaml` 的依赖里只有 `buf.build/envoyproxy/protoc-gen-validate`，没有 protovalidate 模块，那种写法连编译都过不去。
>
> 校验是**逐字段 opt-in** 的：目前只有 5 个 proto 带规则（authentication 6 条、identity/user 3 条、audit 目录下 3 个文件共 4 条）。给新字段加约束时，记得 `make api` 重新生成后确认 `Validate()` 真的会报错，别以为写了规则就自动生效。

### 代码规范

- **Go**：遵循 [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) 与 `golangci-lint`（配置在 `backend/.golangci.yml`：`api/gen`、`internal/data/ent`、`*.pb.go` 等生成目录已排除，所以 lint 报的都是你能在源头修的问题）。提交前在 `backend/` 下跑 `make lint` 和 `go vet ./...`。
- **前端**：三套前端各有独立的 `eslint.config.*` 与 `stylelint.config.*`，请遵守对应配置。
- **命名与风格**：新代码应与所在文件既有风格保持一致（缩进、注释密度、命名习惯）。

> **别照抄 `make vet`。** Makefile 里那条是裸 `go vet`，而 `backend/` 根目录没有 `.go` 文件，跑完只会吐一句 `no Go files in .../backend`——什么都没检查，却不像失败。要真的过一遍，写 `go vet ./...`。

### 提交规范

本项目采用 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/)，历史提交已统一使用。提交信息格式：

```
<type>(<scope>): <subject>
```

| type | 用途 |
|------|------|
| `feat` | 新功能 |
| `fix` | 缺陷修复 |
| `refactor` | 重构（不改变外部行为） |
| `perf` | 性能优化 |
| `style` | 代码格式（不影响逻辑） |
| `docs` | 文档 |
| `test` | 补充测试 |
| `build` | 构建系统、依赖 |
| `chore` | 杂项、脚手架 |
| `ci` | CI 配置 |

- `scope` 可选，对应受影响的模块（如 `auth`、`tenant`、`editor`）
- `subject` 使用中文或英文均可，简明描述，结尾不加句号
- **示例**：`feat(auth): 登录增加租户编号 tenant_code 解析租户`

大改动请添加正文说明动机与影响范围，必要时附 Issue 链接。

## 提交 Pull Request

1. 确保本地通过后端 `make lint` + `go vet ./...` + `make test`，以及对应前端的 lint / typecheck（三端门禁命令见根 [AGENTS.md](./AGENTS.md)）
2. 若改动涉及 proto / schema，确认已重新生成并提交产物：`make api && make openapi && make ent`（`make ts` 只在前端在范围内时跑）。Windows 无原生 make，`api` / `ent` 可改用等价的 `gow api`、`gow ent admin`
3. PR 标题遵循上述提交规范
4. PR 描述请按 [模板](./.github/PULL_REQUEST_TEMPLATE.md) 填写：类型、动机、改动内容、影响范围、自测、备注。**"影响范围"一节请写清改动落在哪一端**——只需一端，另外两端由维护者移植
5. 一个 PR 只解决一个问题，便于评审与回退

> Windows 上 `make test` 偶发 `fork/exec %TEMP%\...test.exe: Access is denied.`——那是安全策略/杀软拦了刚链接好的测试二进制，**不是你的代码挂了**。按根 AGENTS.md 的办法把测试二进制输出到工作区内再直接跑一次确认，别据此提 Issue。

### 评审标准

- 是否改动生成产物、是否漏跑生成
- 是否破坏多租户隔离（跨租户数据访问需带租户过滤）
- 是否引入新的硬编码密钥 / 凭据（应走配置或环境变量）
- 是否补齐了相应测试
- 文档是否同步更新

## 反馈问题

- 缺陷请使用 [Bug 报告模板](./.github/ISSUE_TEMPLATE/bug_report.md)
- 新功能建议请使用 [功能请求模板](./.github/ISSUE_TEMPLATE/feature_request.md)
- 安全漏洞请按 [SECURITY.md](./SECURITY.md) 流程私下上报，勿直接开公开 Issue

## 致谢

你的每一次贡献——无论代码、文档还是问题反馈——都让 GoWind Admin 更好。感谢你的参与。
