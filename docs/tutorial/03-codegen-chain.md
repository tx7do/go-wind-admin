# 03 · 代码生成链路：proto → Go / TS，schema → ent

> 前置：[02 章](./02-get-it-running.md)环境可用。读完你会知道：这个仓库里哪些代码是生成物、生成器吃什么吐什么、以及为什么"改生成文件"永远是错的修法。

## 1. 三条生成链

| 链 | 输入 | 输出 | 命令（`backend/` 下） |
|---|---|---|---|
| proto → Go | `api/protos/**/*.proto` | `api/gen/go/**`（message 类型、HTTP server 注册函数、错误码） | `gow api`（等价 `make api`） |
| proto → OpenAPI | 同上 | `app/admin/service/cmd/server/assets/openapi.yaml`（编译期内嵌，`/docs` 挂 Swagger，受 `configs/server.yaml:5` 的 `enable_swagger` 开关控制） | `make openapi` |
| BFF proto → TS | `api/protos/admin/service/v1/i_*.proto` | 三端的 `api/generated/**`（含 `apiClient.<Entity>Service` 客户端） | `make ts`（一次生成三端） |
| ent schema → ORM | `app/admin/service/internal/data/ent/schema/*.go` | 同目录下除 `schema/` 外全部（client、predicate、迁移） | `gow ent admin` |

另有一条**反向链**：从数据库表生成 CRUD 代码（DSN 驱动，`gow generate`，`--proto-only` 只出 proto）。
这是配套工具链（[gowind-uiapp](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind-uiapp) 桌面端/CLI）的
能力，产物仍须按本仓约定验收补齐——见[第 4 章末尾](./04-first-crud-module.md)与根 AGENTS.md。

**工具分工约定**：`gow` CLI 优先（`gow run / gow ent / gow api`），`make ts`、`make openapi` 等 gow 未覆盖的走 Makefile。这是仓库铁律，别混用别自造。

## 2. 两层 proto：domain 与 BFF

每个业务实体有**两份 proto**，职责严格分离：

```
api/protos/<domain>/service/v1/<entity>.proto        domain 层：message 定义 + 无 HTTP 注解的 service
api/protos/admin/service/v1/i_<entity>.proto         BFF 层：只声明 HTTP 绑定，引用 domain 类型
```

为什么这样分：

- **domain proto 是数据结构与领域接口的唯一定义**（字段、审计字段编号、oneof query_by、update_mask）；
- **BFF proto 决定暴露什么**：路由（一律 `/admin/v1/<entities>`）、暴露哪些 RPC（可以比 domain 少，比如不给 Count）、`additional_bindings` 多维查询；
- **三端 TS 客户端从 BFF proto 生成**——这就是"改了 proto 必须重跑 `make ts`，前端才有新方法"的原因。

样例（写新 proto 前照抄结构）：`dict/service/v1/dict_type.proto`（domain）+
`admin/service/v1/i_dict_type.proto`（BFF）。字段约定（全部 `optional`、审计字段固定编号
100/101/102 与 200/201/202、`json_name` camelCase）在[第 4 章](./04-first-crud-module.md)与
[crud_module_guide/backend.md](../crud_module_guide/backend.md) 里有逐条说明。

## 3. buf 配置文件家族

`backend/api/` 下的 `buf.*.gen.yaml` 各管一条生成链（清单见
[backend_project_struct.md](../backend_project_struct.md)）：Go 代码、OpenAPI、三个前端各自的 TS。
**新建一个全新 domain 时**要往 `buf.gen.yaml` 的 `go_package` overrides 加一行，否则 `gow api`
不知道新包往哪吐——这是新模块流程里唯一需要动 buf 配置的场景。

## 4. ent：schema 是手写的，其余全是生成物

```
internal/data/ent/schema/<entity>.go     ✍ 手写：Fields() / Mixin() / Indexes() / Annotations()
internal/data/ent/<entity>/**            🤖 gow ent 生成：查询构建器、predicate、迁移 DDL
```

手写 schema 的三个要点（样例 `schema/api.go`、带租户与边的 `schema/dict_type.go`）：

- **Mixin 装配公共列**（全仓实际用到的）：`AutoIncrementId` / `TimeAt`（created_at 等）/
  `OperatorID`（created_by 等）/ `SwitchStatus`（status 枚举，默认 ON）/ `IsEnabled` / `SortOrder` /
  `Remark` / `Description` / `TenantID[uint32]{}`（租户表）/ `Tree[<Entity>]{}`（树表：给 `parent_id`
  与 children/parent 边）/ `TreePath{}`（给 `path` 列）。mixin 提供的列名与 proto 审计字段一一对应，
  mapper 按名接线；
- **枚举列**：`field.Enum(...).NamedValues(...)` + proto 枚举**两端命名逐字一致**
  （不一致会在运行时 500，转换器按名字直传）；
- `Indexes()` 里不能引用 edge 外键列（ent 报 unknown index field）。

`gow ent` **不会清目录、不会冲掉 schema/ 里的手写文件**——只重写固定的生成文件名集合，放心跑。
不是嘴上说安全：`internal/data/ent/` 根下就有三个手写的 `*_test.go`（`data_scope_guard_test.go` 等），
和生成物同目录共存至今。

## 5. 生成物边界（背下来）

| 想改的东西 | 正确做法 |
|---|---|
| `api/gen/go/**` 里的类型/服务 | 改 `api/protos/**` → `gow api` |
| 三端 `api/generated/**` 缺方法 | 确认 BFF proto 有该 RPC → `make ts` |
| `internal/data/ent/**`（非 schema）的查询行为 | 改 `schema/**` → `gow ent` |
| `i_*.proto` 的路由 | 改注解 → `gow api` + `make openapi`，部署侧记得接口同步（第 6/9 章） |
| Swagger 没更新 | `make openapi` |

另外：`cmd/server/wiring_*.go` 是**手写 DI**（本仓已弃用 Wire 工具，文件名是历史遗留），新模块的
装配靠 `make register` 注入（第 4 章第 9 步）。

## 深读

- [backend_project_struct.md](../backend_project_struct.md) —— 目录与 buf 配置清单权威说明
- [crud_module_guide/backend.md](../crud_module_guide/backend.md)
  Step 1–5 —— proto/ent 两阶段的逐条约定与样例索引（本仓内最详尽的 CRUD 手册）
- [list_query_rule.md](../list_query_rule.md) —— 生成物 `PagingRequest` 的查询协议语义（分页/排序/过滤操作符/字段掩码）

下一步：[04 · 第一个业务模块](./04-first-crud-module.md)——把本章的链路亲手跑一遍。
