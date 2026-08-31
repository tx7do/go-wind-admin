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
| Go API / HTTP / gRPC / OpenAPI | `make api` | 由 `backend/api/protos/` 下的 proto 生成至 `backend/api/gen/go/` |
| TypeScript HTTP 客户端 | `make api` | 同步生成至各前端 `src/api/generated/` |
| Ent ORM 代码 | `make ent` | 由 `internal/data/ent/schema/` 下的 schema 生成 |

修改 `.proto` 或 `ent/schema` 后，请同步重新生成并一并提交，保持三端一致。

### 输入校验

业务接口请在 proto 中使用 [`buf.validate`](https://buf.build/docs/proto/validate) 声明字段约束（如 `string email = 1 [(buf.validate.field).string.email = true];`），由 `protoc-gen-validate` 自动生成校验代码，避免在 service 层手写判空。

### 代码规范

- **Go**：遵循 [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) 与 `golangci-lint`（见 `backend/Makefile` 的 `lint` 目标）。提交前请本地执行 `make lint vet`。
- **前端**：三套前端各有独立的 `eslint.config.*` 与 `stylelint.config.*`，请遵守对应配置。
- **命名与风格**：新代码应与所在文件既有风格保持一致（缩进、注释密度、命名习惯）。

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

1. 确保本地通过 `make lint vet test`（后端）与对应前端的 lint
2. 若改动涉及 proto / schema，确认已重新生成代码并提交
3. PR 标题遵循上述提交规范
4. PR 描述请按 [模板](./.github/PULL_REQUEST_TEMPLATE.md) 填写：动机、改动内容、自测情况、是否影响三端
5. 一个 PR 只解决一个问题，便于评审与回退

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
