# 只改一个字段：速查（改哪几个文件、跑哪几条命令）

> **定位**：[02 章](./tutorial/02-get-it-running.md) 教你把系统跑起来，[04 章](./tutorial/04-first-crud-module.md) 教你从零加一个模块。
> 中间那一档——"**资源已经有了，我只给它加一个字段，而且只在我这一端**"——是日常最高频的活，两章都不覆盖。这页就是补那一档。
> 只读表格即可开工；每格后面的代码位置是判据，冲突时以代码为准。
> 本页写法：逐个 `file:行号` 读回原文 + 读 `go.mod` 钉住的依赖源码，**未起后端打真实请求**——未验证项在 §6 末尾明确列出。

样例模块全篇统一用 **字典类型**（`sys_dict_types` / 三端的 dict 页面）——它是最标准的纯标量 CRUD，没有边、没有枚举，加字段的动作最干净。

**路径基准**：后端代码位置一律相对 `backend/app/admin/service/`（跑命令时在 `backend/`），前端相对各端根目录。

---

## 0. 先定档：你要跑的命令数取决于这一问

**"这个字段在接口返回里已经有了吗？"** 一句话判：

```bash
# 在 backend/ 下，看 domain proto 里有没有这个字段
grep -n "json_name" api/protos/dict/service/v1/dict_type.proto
```

| 档 | 判据 | 后端命令 | 后端手写文件 | 前端文件（你那一端） |
|---|---|---|---|---|
| **A** | 字段已在 proto 里，只是页面没显示/没录入 | **0 条** | 0 | 3 个（见 §3） |
| **B** | 新字段，标量（字符串/数字/布尔/时间） | 4 条（§2.1） | 3 个 / 4 处（§2.2） | 3 个 + 生成的 `index.ts` |
| **C** | 新字段，枚举类型 | 4 条 | 3 个，另多接 3 处（§2.3） | 同 B |
| **D** | 除了字段还要新端点/新页面 | —— | 走 [04 章](./tutorial/04-first-crud-module.md) | 走 04 章 |

A 档纯前端，直接跳 §3。B/C 档先看 §1 的三条前提，能省掉一半动作。

---

## 1. 三条前提事实（决定了"不用做什么"）

1. **加列不需要 SQL、不需要迁移命令。** 启动时 ent 自动建列：`internal/data/ent_client.go:46-51` 在 `cfg.Data.Database.GetMigrate()` 为真时跑 `client.Schema.Create(...)`，而 `configs/data.yaml:7` 就是 `migrate: true`。→ 改完 schema 只要**重启后端进程**。
   所以业务字段照 04 章一律 `Optional().Nillable()`（本仓所有 schema 都这样，见 `ent/schema/dict_type.go:34-45`）——加非空列要自动迁移去给已有行填值，那是给自己加活。要"必填"交给前端 `rules` + proto validate。
2. **想加的字段很可能已经在 mixin 里。** `AutoIncrementId / TimeAt / OperatorID / IsEnabled / SortOrder / TenantID` 六个 mixin（`data/ent/schema/dict_type.go:50-59`）已经贡献了 `id / created_at / updated_at / deleted_at / created_by / updated_by / deleted_by / is_enabled / sort_order / tenant_id`。先看这张表，别在 `Fields()` 里重造一个 `status`。
3. **BFF proto 不定义 message，所以加字段只改一个 proto。** `api/protos/admin/service/v1/i_dict_type.proto:9` 只是 `import` domain 包再挂 `google.api.http`；43 个 `i_*.proto` 里只有 `i_admin_portal.proto`、`i_dashboard.proto` 自己声明 message，而它们不是 CRUD 资源。→ domain 那一处加完就够了。

字段编号规则照 04 章：业务字段取下一个空号，审计字段固定 `100/101/102`（*By）与 `200/201/202`（*At）；全部 `optional` + `json_name` camelCase。

---

## 2. 后端（B / C 档）

### 2.1 命令序列（在 `backend/` 下，gow 优先）

```bash
gow api        # proto -> *.pb.go / *.pb.validate.go
make openapi   # -> openapi.yaml（这份是打进二进制的，「接口同步」读它）
gow ent admin  # schema -> ent CRUD 代码（含 migrate/schema.go）
make ts        # -> 各端 src/api/generated/（vben 在 apps/admin/src/ 下）
```

两条注意：

- **`make ts` 会同时写三端**，即使你只改一端。另外两端的 `index.ts` 也随之变化——**一起提交，不要 revert**（提交口径见 [CONTRIBUTING.md](../CONTRIBUTING.md)：生成产物三端齐、业务页面只需一端）。
- 只改前端能看见的东西（A 档）不用跑任何一条。

### 2.2 手写的就三个文件、四处

| 文件 | 处 | 做什么 |
|---|---|---|
| `api/protos/dict/service/v1/dict_type.proto` | 1 | `DictType` 里加一行 `optional <类型> <字段名> = <下一个空号> [json_name = "camelCase", (gnostic.openapi.v3.property) = {description: "…"}];` |
| `internal/data/ent/schema/dict_type.go` | 1 | `Fields()` 里加 `field.<Type>("<字段名>").Comment("…").Optional().Nillable()`（判据见 `:34-45`） |
| `internal/data/dict_type_repo.go` | 2 | 抄现成的链式调用，见下表 |

`dict_type_repo.go` 的两处：

| 位置 | 代码 | 加字段要做的事 |
|---|---|---|
| Create | `:170-177` 的 `tx.DictType.Create().SetNillable…` 链 | 接上 `SetNillable<字段>(req.Data.<字段>)` |
| Update | `:233-246` 的 `UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(), func(dto){…})` 回调 | 同上 |

**BFF proto 不用改**（§1 第 3 条），**service 层也不用改**：`internal/service/dict_type_service.go` 里 List/Get 直通 repo，Create/Update 只注入 operator（`:57` / `:79-82` 把 `updated_by` 追加进 mask），全文 0 处业务字段名。

**其余全不用改**：`List` / `Count` / `Get` 走泛型仓储（`ListWithPaging(ctx, builder, builder.Clone(), req)`，`:91`），搜索、排序、分页、字段掩码都不按字段写代码——加一个可搜索字段是**零后端改动**。

一个反例值得记：Update 链里**故意没有** `type_code`，因为 schema 把它标了 `Immutable()`（`ent/schema/dict_type.go:37`）。给不可变字段接上 Update 是静默失效，不是编译错误。

### 2.3 C 档（枚举）额外三处

照 `api_repo.go`：转换器存进结构体（`:32-34`）→ `init()` 里
`mapper.NewEnumTypeConverter[…_name, …_value]`（`:55`）并 `AppendConverters(…NewConverterPair())`（`:87-89`）→
Create/Update 两处**仍然是 `SetNillable<字段>(…)`，只是实参要过一层 `ToEntity`**：
`SetNillableScope(r.scopeConverter.ToEntity(api.Scope))`（`:243`、`:313`）。

读侧不用手写——`NewConverterPair()` 已经把枚举转换挂进 mapper 了。**只有写侧的 `SetNillable` 链要自己接**，
忘了 `ToEntity` 时编译不过（类型不匹配），但 proto 枚举名与 Go 枚举名对不上是**运行期**才表现为字段零值。

---

## 3. 前端：你那一端要碰的三个文件

| | 列表（加一列） | 抽屉/弹窗表单（加一项） | 词条 |
|---|---|---|---|
| **react** | `src/pages/app/system/dict/DictTypeList.tsx:50` `columns` 数组 | 同目录 `DictTypeDrawer.tsx`：表单项 `:126+`、编辑回填 `:45-48`、新建默认值 `:109` | `src/locales/{zh-CN,en-US}/_modules/dict-type.json` |
| **vue-element** | `src/pages/app/system/dict/dict-type-list.vue`：`table.columns:85`（`prop` camelCase）、需要可搜时 `search.fields:49` | 同目录 `dict-type-drawer.vue`：`ElFormItem` 区 `:17-52`、新建默认值 `defaults:88-93`、**编辑回填 `:113-116`（逐字段手写）**、校验 `formRules:99-106` | `src/locales/{zh-CN,en-US}/pages/dict.json` |
| **vue-vben** | `apps/admin/src/views/app/system/dict/dict-type-list.vue:73` `columns`（`field` camelCase） | 同目录 `dict-type-drawer.vue:33` `useVbenForm` 的 `schema` | `apps/admin/src/locales/langs/{zh-CN,en-US}/page.json` |

四处容易漏的：

- **编辑回填在 react / element 两端都是逐字段手写的**：react `DictTypeDrawer.tsx:45-48` 的
  `setFieldsValue({...})`、element `dict-type-drawer.vue:113-116` 的 `drawer.formData.xxx = row.xxx ?? …`。
  漏了这一处，新建正常、**一编辑该字段就空**（vben 是整行 `baseFormApi.setValues(row)`，`:130`，不用逐字段加）。
- **hooks / composables 不用碰。** `react/src/api/hooks/dict.ts`、`vue-*/src/api/composables/dict.ts` 里没有任何字段名，字段跟着生成的类型走。
- **词条文件布局三端不同**，且 vben 是**共用一个大 `page.json`**：两种语言都要加，只加 `zh-CN` 会在英文界面露出 key。react 用扁平 camelCase 键（`typeName` / `typeNamePlaceholder` / `requiredTypeName` 一族）。
- **表单校验另写一处**：element 的 `formRules:99` 与 `defaults:88` 是分开的两个对象，只加 `ElFormItem` 会得到一个能留空的必填字段。

改完跑你那一端的门禁（三端表见根 [AGENTS.md](../AGENTS.md)）：react `npm run typecheck`；vue-element `npx vue-tsc --noEmit`；vue-vben `pnpm run check:type`。

---

## 4. 明确不需要做的五件事

| 别做 | 为什么不用 |
|---|---|
| 点「接口同步」/ 改 Api 表 | 闸门按 `(path, method)` 建行（`internal/service/api_service.go:200-201` 遍历 `doc.Paths.Map()` × operations）。加字段不改路径，也就不新增任何一行。**只有新端点才需要同步**，且那是 04 章的事。 |
| 配字段权限 | 字段黑名单是 opt-in 的数据库行，语义写在 schema 注释里：`未配置 = 全字段可见`（`ent/schema/role_field_permission.go:13-15`，`field_name` 存 proto 的 `json_name`）。新字段默认对所有人可读可写。 |
| 加菜单 / 走菜单同步 | 字段进的是已有页面，没有新路由。 |
| 写 SQL、找迁移脚本 | §1 第 1 条。`backend/sql/` 下只有演示数据。 |
| 手改 `src/api/generated/` | 生成物，下一次 `make ts` 就没了。 |

---

## 5. 两个"接口 200 但结果是错的"

新手最贵的不是编译错误，是这两条：

1. **Update 没带 `updateMask`。** 后端 `FilterByFieldMask` 对缺 mask 的 DTO 是**全字段直写**（`go-utils v1.1.40` 的 `fieldmaskutil/fieldmaskutil.go:63-65`：mask 为 nil 时直接返回、不过滤；调用点 `go-crud/entgo v0.0.55` 的 `repository.go:882`）：表单里没渲染的列会被写成零值，接口照样返回成功。必须用生成的 `useUpdateXxx`（内部 `makeUpdateMask(Object.keys(values))`，react 侧实测 `src/api/hooks/dict.ts:76`），别手拼、别用旧的裸 client 方法。见 04 章"跨端铁律"第一条。
2. **搜索条件的操作符后缀拼错。** 大小写与驼峰**不是**问题——query 解析会把键的字段段做 camel→snake 归一（`go-crud/pagination v0.0.16` 的 `filter/query_string_converter.go:268,295,300,334` 四处 `stringcase.ToSnakeCase`），`typeName__contains` 与 `type_name__contains` 同效。真正**静默**中招的是后缀：

   后缀写成不认识的名字（`name__badop`）时，转换器确实返回 error（`query_string_converter.go:282-283`），
   但泛型仓储只 `log.Error` 不上抛（`go-crud/entgo v0.0.55` 的 `repository.go:333-340`），且那份 error 让
   **整个** `FilterExpr` 变 nil ⇒ 该请求的**全部**过滤条件一起失效，**搜了等于没搜：返回全量、页面上一眼看不出不对**
   （2026-09-25 sqlite 探针实测：`{"name__badop":"x"}` 在通知渠道表上回全量 2 行、无 error；"HTTP 200"是由"没有 error"推出来的，没打请求实测）。
   只有 `user`/`permission` 两处手写过 filter 的 repo 会上抛：`internal/data/user_repo.go:308-312`、
   `internal/data/permission_repo.go:132-135`。

   还有一条同族的静默坑：后缀写成驼峰（`field__notEqual`）时，前端守卫认不出、给它追加 `__contains`，
   后端三段解析把中间段整段丢掉 ⇒ 查询变成 `field CONTAINS value`，"排除"反转成"命中"。
   所以**搜索键后缀一律写表里的规范 snake_case 拼写**。`orderBy` 建议一并写蛇形：三端序列化器发的 JSON 数组形
   会把驼峰归一，但 AIP 裸串形既不归一、也不支持 `-field` 降序——两形的差别与实测见
   [list_query_rule.md](./list_query_rule.md) 的「坏列名与坏后缀是两种失败」。

   > **本文旧版在这一条开头写反了一件事，值得单独记住**：它把"列名本身拼错"也说成静默（"白名单会整条丢掉该条件 ⇒ 200 + 全量"），
   > 依据是 `internal/data/ent/ent.go:122-179` 那张 51 表在册的 `sql.NewColumnCheck`。那是**本仓自己生成的** `checkColumn`，
   > 而 go-crud 的 filter / sorting / fieldMask 读的是它**自带**的那张（只登记它示例 schema 的 `menus` 与 `users`）
   > ⇒ 本仓 `sys_*` 一张都不在上面，三处 `columnAllowed` 全走 fail-open，坏列原样拼进 SQL、由数据库拒绝，
   > 实测回 `query list failed`（`fieldMask` / `orderBy` 同形）。⇒ **列名拼错是报错，后缀拼错才是 200 + 全量。**
   > 机制、四格实测与两个例外，唯一权威在 [list_query_rule.md](./list_query_rule.md) 的「坏列名与坏后缀是两种失败」。

---

## 6. 验收清单

- [ ] **重启**后端（改 schema 后热重载不算，自动迁移只在启动链里跑）；
- [ ] 库里出现新列（`psql`/客户端看 `sys_dict_types`），且**旧行取的是默认值而非报错**；
- [ ] <http://localhost:7788/docs> 里该资源的 schema 出现这个字段，`json_name` 与你前端写的键一致；
- [ ] 建一条带新字段的记录 → 列表能看到 → **编辑时只改另一个字段** → 回来看新字段没被清零（这一条专杀 §5.1）；
- [ ] 你那一端门禁 0 错误；
- [ ] `git status`：生成物齐——`*.pb.go`、`ent/**`（含 `migrate/schema.go`）、`openapi.yaml`、你这一端的 `src/api/generated/`。若你维护的是本仓上游（三端都在），`make ts` 会把另两端的 `index.ts` 一起改到，**跟着提交**；只采用一端的话，另一端本就不在你树上。

**本篇未做的验证**：所有 `file:行号` 都逐个读回过原文；§5 的两条 go-crud 分支（缺 mask ⇒ 全字段直写、未知列 ⇒ 静默丢条件）是**读 `go.mod` 钉住的那个版本的依赖源码**得出的，另加一个本地探针确认 ent 的 `Selector.C("typeName")` 自身不做归一（归一发生在上游 go-crud）。**没有**在跑起来的后端上打真实请求复核（写这页时 `:7788` 无进程）。要较真的话，§5.2 值得起库测一次：拼错列名应返回全量而非 4xx。

---

## 深读

- [04 章 第一个业务模块](./tutorial/04-first-crud-module.md) —— 新资源/新端点
- [03 章 代码生成链路](./tutorial/03-codegen-chain.md) —— 哪些文件是生成的、谁生成
- [list_query_rule.md](./list_query_rule.md) —— 操作符全表、字段掩码
- [frontend_authority.md](./frontend_authority.md) —— 真要把字段设成"某些角色不可见"时
- [adopt-one-frontend.md](./adopt-one-frontend.md) —— 只维护一端时的裁剪与 `make ts` 回流
- 各端约定权威：`frontend/admin/{react,vue-element,vue-vben}/AGENTS.md`
