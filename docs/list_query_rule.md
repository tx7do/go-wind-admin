# 列表查询规则

## 通用列表查询请求

| 字段名       | 类型        | 格式                                  | 字段描述    | 示例                                                                                                       | 备注                                                               |
|-----------|-----------|-------------------------------------|---------|----------------------------------------------------------------------------------------------------------|------------------------------------------------------------------|
| page      | `number`  |                                     | 当前页码    |                                                                                                          | 默认为`1`，最小值为`1`。                                                  |
| pageSize  | `number`  |                                     | 每页的行数   |                                                                                                          | 默认为`10`，最小值为`1`。                                                 |
| query     | `string`  | `json object` 或 `json object array` | AND过滤条件 | json字符串: `{"field1":"val1","field2":"val2"}` 或者`[{"field1":"val1"},{"field2":"val2"}]` | 两种顶层形态**都是 AND**（解析器把数组整段建为一个 `AND` 组，见下方「数组语义」）。同字段多值请用 `field__in` 或 `$or`。规则见：[过滤规则](#过滤规则) |
| filter    | `string`  | Google AIP-160 过滤字符串              | AND过滤条件（另一种写法） | 例如 `name = "tom" AND age > 10`                                                              | 与 `query` 二选一，不是叠加 |
| filterExpr| `object`  | `pagination.v1.FilterExpr`            | 结构化过滤条件 | `{"type":"AND","conditions":[{"field":"name","op":"EQ","value":"tom"}]}`                      | 优先级最高，给了它就不再读 `query`/`filter` |
| orderBy   | `string`  | `json string array`                 | 排序条件    | json字符串：`["-create_time", "type"]`                                                                       | json的`string array`，字段名前加`-`是为降序，不加为升序。具体规则请见：[排序规则](#排序规则)      |
| noPaging  | `boolean` |                                     | 是否不分页   |                                                                                                          | 此字段为`true`时，`page`、`pageSize`字段的传入将无效用。                          |
| fieldMask | `string`  | 其语法为使用逗号分隔字段名                       | 字段掩码（SELECT 投影） | 例如：id,realName,userName。                                                                                 | 落到 List/Get 的 SELECT 投影（库 `fieldSelector.BuildSelector` → `BuildSelect`，`go-crud/entgo v0.0.55` 的 `field/field_selector.go:58-67`）；为空时为`*`。驼峰会被归一成 snake_case（`field/utils.go:25-38`）。**拼错的项在本仓不会被静默剔除，而是整条查询报错**，见下方「坏列名与坏后缀是两种失败」。字段级权限（hfs）与此掩码无关：它是在响应上清值，见 [frontend_authority.md](./frontend_authority.md)。 |

**三个过滤入口是互斥的（proto 里同属一个 `oneof`）**，解析顺序为 `filterExpr` → `query` → `filter`，取第一个非空的（`go-crud/pagination v0.0.16` 的 `filter/converter.go:17-35`）。同时传不会报错，后面的被忽略。

## 排序规则

排序操作本质上是`SQL`里面的`Order By`条件。

| 序列 | 示例                 | 备注           |
|----|--------------------|--------------|
| 升序 | `["type"]`         |              |
| 降序 | `["-create_time"]` | 字段名前加`-`是为降序 |

## 过滤规则

过滤器操作本质上是`SQL`里面的`WHERE`条件。

过滤器的规则，遵循了Python的ORM的规则，比如：

- [Tortoise ORM Filtering](https://tortoise.github.io/query.html#filtering)。
- [Django Field lookups](https://docs.djangoproject.com/en/4.2/ref/models/querysets/#field-lookups)

如果只是普通的查询，只需要传递`字段名`即可，但是如果需要一些特殊的查询，那么就需要加入`操作符`了。

特殊查询的语法规则其实很简单，就是使用双下划线`__`分割字段名和操作符：

```text
{字段名}__{查找类型} : {值}
{字段名}.{JSON字段名}__{查找类型} : {值}
```

**数组语义与 OR**：顶层数组只是"多个条件写成多行"，逻辑连接词仍然是 **AND**——解析器把数组整体建成一个
`ExprType_AND` 组（`go-crud/pagination v0.0.16` 的 `filter/query_string_converter.go:88-95`）。所以
`[{"type":"a"},{"type":"b"}]` 是 `type=a AND type=b`，**恒为空集**，不是"同字段两个值任一命中"。

要 OR，只有两条路：

- 同字段多值 → `{"type__in":"[\"a\",\"b\"]"}`；
- 跨字段/复杂条件 → 顶层对象里用 `$or`（也支持 `$and` 嵌套）：`{"$or":[{"name__contains":"张"},{"mobile__contains":"138"}]}`。
  单个逻辑节点里 `$and` 与 `$or` 不能同时出现（`query_string_converter.go:131-133` 直接报错）。

| 查找类型        | 示例                                                            | SQL                                                                                                                                                                                                                       | 备注                                                                                                            |
|-------------|---------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------|
| eq          | `{"name__eq" : "tom"}`                                        | `WHERE "name" = "tom"`                                                                                                                                                                                                     | 与裸键 `{"name":"tom"}` 等价（别名 eq/equal/equals）；写不写都行，见下方 `exact` 行的说明 |
| not         | `{"name__not" : "tom"}`                                       | `WHERE NOT ("name" = "tom")`                                                                                                                                                                                              | `not` 是 `ne` 族的别名（ne/neq/not/not_equal/not_equals），见下方别名说明 |
| in          | `{"name__in" : "[\"tom\", \"jimmy\"]"}`                       | `WHERE name IN ("tom", "jimmy")`                                                                                                                                                                                          |                                                                                                               |
| not_in      | `{"name__not_in" : "[\"tom\", \"jimmy\"]"}`                   | `WHERE name NOT IN ("tom", "jimmy")`                                                                                                                                                                                      |                                                                                                               |
| gte         | `{"create_time__gte" : "2023-10-25"}`                         | `WHERE "create_time" >= "2023-10-25"`                                                                                                                                                                                     |                                                                                                               |
| gt          | `{"create_time__gt" : "2023-10-25"}`                          | `WHERE "create_time" > "2023-10-25"`                                                                                                                                                                                      |                                                                                                               |
| lte         | `{"create_time__lte" : "2023-10-25"}`                         | `WHERE "create_time" <= "2023-10-25"`                                                                                                                                                                                     |                                                                                                               |
| lt          | `{"create_time__lt" : "2023-10-25"}`                          | `WHERE "create_time" < "2023-10-25"`                                                                                                                                                                                      |                                                                                                               |
| range / between | `{"create_time__range" : "[\"2023-10-25\", \"2024-10-25\"]"}` | `WHERE "create_time" BETWEEN "2023-10-25" AND "2024-10-25"` <br>或<br> `WHERE "create_time" >= "2023-10-25" AND "create_time" <= "2024-10-25"`                                                                             | `range` 与 `between` 同义。需要注意的是: <br>1. 有些数据库的BETWEEN实现的开闭区间可能不一样。<br>2. 日期`2005-01-01`会被隐式转换为：`2005-01-01 00:00:00`，两个日期一致就会导致查询不到数据。 |
| isnull      | `{"name__isnull" : "True"}`                                   | `WHERE name IS NULL`                                                                                                                                                                                                      |                                                                                                               |
| not_isnull  | `{"name__not_isnull" : "False"}`                              | `WHERE name IS NOT NULL`                                                                                                                                                                                                  |                                                                                                               |
| contains    | `{"name__contains" : "L"}`                                    | `WHERE name LIKE '%L%';`                                                                                                                                                                                                  |                                                                                                               |
| icontains   | `{"name__icontains" : "L"}`                                   | `WHERE name ILIKE '%L%';`                                                                                                                                                                                                 |                                                                                                               |
| startswith  | `{"name__startswith" : "La"}`                                 | `WHERE name LIKE 'La%';`                                                                                                                                                                                                  |                                                                                                               |
| istartswith | `{"name__istartswith" : "La"}`                                | `WHERE name ILIKE 'La%';`                                                                                                                                                                                                 |                                                                                                               |
| endswith    | `{"name__endswith" : "a"}`                                    | `WHERE name LIKE '%a';`                                                                                                                                                                                                   |                                                                                                               |
| iendswith   | `{"name__iendswith" : "a"}`                                   | `WHERE name ILIKE '%a';`                                                                                                                                                                                                  |                                                                                                               |
| exact       | `{"name__exact" : "a"}`                                       | `WHERE name = 'a';`                                                                                                                                                                                                       | 精确等值（与裸键 `{field: value}` 的 EQ 等价，exact 是显式拼写）                                             |
| iexact      | `{"name__iexact" : "a"}`                                      | `WHERE name ILIKE 'a';`                                                                                                                                                                                                   | 大小写不敏感精确匹配（显式拼写）                                                                              |
| like        | `{"name__like" : "a"}`                                        | `WHERE name LIKE 'a';`                                                                                                                                                                                                    | LIKE，无自动通配（区别于 contains 族自动加 `%`）。⚠ 本仓 Ent 路径未实现，见下方「六个解析得动、库不实现的查找类型」                                                              |
| ilike       | `{"name__ilike" : "a"}`                                       | `WHERE name ILIKE 'a';`                                                                                                                                                                                                    | 同上，大小写不敏感（别名 `i_like`）。⚠ 同上，Ent 路径未实现                                                                            |
| not_like    | `{"name__not_like" : "a"}`                                    | `WHERE name NOT LIKE 'a';`                                                                                                                                                                                                 | 取反 LIKE（别名 `notlike`）。⚠ 同上，Ent 路径未实现                                                                                    |
| json_contains | `{"data.foo__json_contains" : "x"}`                          | （JSON 列）路径包含判定                                                                                                                                                                                                    | JSON 路径字段（`{field}.{jsonpath}` 形态）专用。⚠ 同上，Ent 路径未实现                                                                |
| array_contains | `{"tags__array_contains" : "x"}`                           | （数组列）成员包含判定                                                                                                                                                                                                     | 数组类型列专用。⚠ 同上，Ent 路径未实现                                                                                                |
| exists      | `{"data.foo__exists" : true}`                                 | （JSON 列）路径存在性判定                                                                                                                                                                                                  | JSON 路径字段专用。⚠ 同上，Ent 路径未实现                                                                                            |
| regex       | `{"title__regex" : "^(An?\|The) +"}`                          | MySQL: `WHERE title REGEXP BINARY '^(An?\|The) +'`  <br> Oracle: `WHERE REGEXP_LIKE(title, '^(An?\|The) +', 'c');`  <br> PostgreSQL: `WHERE title ~ '^(An?\|The) +';`  <br> SQLite: `WHERE title REGEXP '^(An?\|The) +';` |                                                                                                               |
| iregex      | `{"title__iregex" : "^(an?\|the) +"}`                         | MySQL: `WHERE title REGEXP '^(an?\|the) +'`  <br> Oracle: `WHERE REGEXP_LIKE(title, '^(an?\|the) +', 'i');`  <br> PostgreSQL: `WHERE title ~* '^(an?\|the) +';`  <br> SQLite: `WHERE title REGEXP '(?i)^(an?\|the) +';`   |                                                                                                               |
| search      | `{"title__search" : "tom"}`                           | PG: `to_tsvector(title) @@ plainto_tsquery(..)`<br>MySQL: `MATCH(title) AGAINST(.. IN NATURAL LANGUAGE MODE)`<br>SQLite / 其他方言: `title LIKE "%value%"` | 全文搜索，按方言分支实现（`go-crud/entgo v0.0.55` 的 `filter/structured_filter.go:451-489`）；值为空时该条件不产生谓词 |

**六个"解析器认、Ent 层不实现"的查找类型**（`like` / `ilike` / `not_like` / `json_contains` /
`array_contains` / `exists`）——**这是本页面最容易踩的一条**：

它们在 `operatorMap` 里是合法拼写，解析器会正常产出对应的 `Operator_LIKE` / `Operator_EXISTS` 等枚举，
**不报 400**；但本仓走的 Ent 过滤器 `switch` 里没有这些分支，落到 `default: return nil`
（`go-crud/entgo v0.0.55` 的 `filter/structured_filter.go:129-175`），于是**该条件被整条丢弃**。
若它是唯一条件，结果就是**不过滤的全量数据 + HTTP 200**。

所以：需要无自动通配的 LIKE 语义时，用 `exact`/`regex` 顶上（两者 Ent 侧有实现）；需要"不含某子串"用
`{"field__not":"x"}` 的等值取反或 `not_in`，不要写 `not_like`。**改这条的权威判据是那个 `switch` 的 case 列表**。

**别名族与权威来源**：上表为各族规范拼写。解析器（`go-crud/pagination v0.0.16` 的
`filter/operator_converter.go` `operatorMap`，`:11-102`）另接受整族别名——数值比较的长拼写（`greater_than`/`greater_than_or_equal`/`less_than`
等）、`i_*` 前缀变体（`i_contains` 等）、`nin`/`notin`、`is_not_null` 拼写族、`regexp`/`regex` 互换等。
**操作符集以该映射表为唯一权威**，本表不逐一枚举别名。注意区分两个模块：操作符表与 query 解析在
`go-crud/pagination`，把枚举翻成 SQL 的实现在 `go-crud/entgo`，两者版本不同、能力不等价（见上一段）。

**解析行为：转换器 fail-closed，但仓储层多半不往上抛**——这句话分两半，合起来才是实际行为：

1. **解析器**对未知操作符、超过三段的键（`field__op1__op2__extra`）确实返回 error
   （`query_string_converter.go:282-283`、`:395-396`），**不会只丢那一个键**——它一次性返回整份
   `FilterExpr = nil`。所以一个坏键毁掉的是**这个请求的全部条件**。
2. **谁把这份 error 变成 HTTP 状态码，各资源不一样**：
   - 走泛型仓储的常规 CRUD（`ListWithPaging`）只 `log.Error` **不上抛**
     （`go-crud/entgo v0.0.55` 的 `repository.go:333-340`），拿到 nil 表达式后照常查询 ⇒
     **过滤全失效、返回全量、HTTP 200**。前端搜索框里多打一个下划线就能撞上。
   - 手写过 filter 的两处会上抛：`internal/data/user_repo.go:308-312`（返回 err）、
     `internal/data/permission_repo.go:132-135`。这两张表的坏键才会真的报错。

   实践结论：**别指望"后缀写坏"的键报错**。前端序列化器（下一节）之所以要严格守卫后缀，正是因为后端不会替你兜住。
   三段键的两种带参数形态：`{field}__{datepart}__{op}` 为日期部件查询（下表，**第三段比较操作符必填**）；
   `{field}.{jsonfield}__{op}` 为 JSON 路径查询（点号分隔路径）。

**坏列名与坏后缀是两种失败**（2026-09-25 探针实测：直调 `NotificationChannelRepo.List`、sqlite 内存库，其余参数正常，每次只写坏一处）：

`query` / `fieldMask` / `orderBy` 三个输入都会把字段名送去查一张列白名单，但那张表**只登记了 go-crud 自带示例 schema 的两张表**（`go-crud/entgo v0.0.55` 的 `ent/ent.go:74-82`，键是 `menus` 与 `users`，见 `ent/menu/menu.go:26`）。本仓表名一律 `sys_*`、一张都不在上面，于是三处 `columnAllowed` 全部走"表未登记 ⇒ fail-open"这一支（`filter/structured_filter.go:210-221`、`sorting/structured_sorting.go:14-23`、`field/field_selector.go:41-50`）：白名单不拦，坏列原样拼进 SQL，由数据库拒绝。

| 写坏的东西 | 实测结果（仓储层） | 成因 |
|---|---|---|
| `query` 的列名段 `{"no_such_column__contains":"x"}` | error `query list failed` | fail-open ⇒ SQL 出现不存在的列（`repository.go:190-193` 把数据库错误压成这一句） |
| `query` 的操作符后缀 `{"name__badop":"x"}` | **无 error、返回全量**（探针表 2 行） | 整份 `FilterExpr` 变 nil，仓储只 `log.Error` |
| `fieldMask` 里混进一个未知项 | error `query list failed`（**不是**"少返回那一列"） | 同上 fail-open，坏列进了 SELECT |
| `orderBy` 里写一个不存在的列 | error `query list failed`（两形都测过：`["noSuchColumn"]` 与裸 `no_such_column`） | 同上，坏列进 `ORDER BY` |

两件事要分清楚：

- 上表"结果"只写到仓储层为止。error 会经 service 原样上抛（如 `internal/service/notification_channel_service.go:52` 是 `return s.repo.List(...)`，不吞错），HTTP 状态码由"有没有 error"推出来（坏列 ⇒ 5xx、坏后缀 ⇒ 200 + 全量），**这一格没逐个打 HTTP 请求实测**。
- 只有当表**在** go-crud 那张白名单上、而列不在时，条件才会被静默丢弃：`getField` 拿空串让 `Process` 跳过这一条（`filter/structured_filter.go:122-128`、`:184-206`）。本仓的 `sys_menus` / `sys_users` 都不在那张表上，**别把这条当成通用行为**。另外含 SQL 元字符的字段名会被硬性标识符校验无条件拒掉（`filter/structured_filter.go:191-193`、`field/field_selector.go:20`），那是防注入、与列白名单无关。

**大小写与驼峰：`query` 与 `fieldMask` 一律归一，`orderBy` 看传形**（同一探针实测）：

- `query` 的字段段：四处 `stringcase.ToSnakeCase`（`query_string_converter.go:268,295,300,334`）；
- `fieldMask`：`field/utils.go:25-38` 的 `NormalizePaths`（调用点 `field/field_selector.go:64`）；
- `orderBy` 分两形，`sorting/order_by_string_converter.go:20-32` 按首尾有没有 `[`…`]` 分流：
  - **JSON 数组形** `["-createdAt"]` —— 三端 `PaginationQuery.makeOrderBy` 发的就是这一形（`JSON.stringify`）。它逐项 `ToSnakeCase`（`:78`），**驼峰能用**（实测 `["smtpHost"]` 真按 `smtp_host` 排出了序），`-` 前缀为降序；
  - **AIP 裸串形** `created_at` —— **不**归一，驼峰原样进 `ORDER BY "smtpHost"` ⇒ 报错；而且这一形**根本不接受 `-name`**（`-` 不是合法 field-path 字符，回 `unmarshal order by '-name': invalid character '-'`，且这条 error 是 `repository.go:366-369` 少数**会上抛**的分支）。要降序只能走 JSON 形。

所以"排序键写蛇形"是**更稳的约定**（少依赖一层归一、也不踩裸串形那两条），但它不是"不写就报错"的铁律：走三端序列化器发出的 JSON 形时，驼峰会被归一。

以及将日期提取出来的查找类型：

> ⚠ **本仓 Ent 路径下这一族不可用**，语法能过、结果不对。理由是这个版本里日期部件只有"解析"没有"落地"：
> 第三段部件名被正常写进 `filterCondition.DatePart`（`query_string_converter.go:355`），
> 但 `go-crud/entgo v0.0.55` 的 `filter/structured_filter.go` 里 `DatePart()`（`:493`）与 `DatePartField()`（`:533`）
> **两个方法在全模块内没有任何调用点**，而各比较助手（`Equal` `:225` 等）一律打在原始列上。
> 于是 `{"created_at__year__eq":"2024"}` 实际生成的是 `WHERE "created_at" = '2024'`——
> 拿时间列去比一个短字符串，PG 下大概率是类型错误（HTTP 500，**此后果为推断，未实机验证**），
> 即便类型相容也是错的结果。要按年/月过滤，目前请改用区间：`{"created_at__gte":"2024-01-01","created_at__lt":"2025-01-01"}`。
> 下表因此只描述**解析器接受的部件拼写**，不描述实际 SQL。

| 查找类型         | 示例                                   | SQL（**未生效**，见上方提示）                          | 备注                   |
|--------------|--------------------------------------|---------------------------------------------------|----------------------|
| date         | `{"pub_date__date__eq" : "2023-01-01"}`  | `WHERE DATE(pub_date) = '2023-01-01'`             |                      |
| year         | `{"pub_date__year__eq" : "2023"}`        | `WHERE EXTRACT('YEAR' FROM pub_date) = '2023'`    | 哪一年（别名 `yr`）          |
| iso_year     | `{"pub_date__iso_year__eq" : "2023"}`    | `WHERE EXTRACT('ISOYEAR' FROM pub_date) = '2023'` | ISO 8601 历年           |
| month        | `{"pub_date__month__eq" : "12"}`         | `WHERE EXTRACT('MONTH' FROM pub_date) = '12'`     | 月份，1-12              |
| day          | `{"pub_date__day__eq" : "3"}`            | `WHERE EXTRACT('DAY' FROM pub_date) = '3'`        | 该月的某天(1-31)          |
| week         | `{"pub_date__week__eq" : "7"}`           | `WHERE EXTRACT('WEEK' FROM pub_date) = '7'`       | ISO 8601 周编号 一年中的周数 |
| week_day     | `{"pub_date__week_day__eq" : "tom"}`     | ``                                                | 星期几（别名 `weekday`）    |
| iso_week_day | `{"pub_date__iso_week_day__eq" : "tom"}` | ``                                                |                      |
| quarter      | `{"pub_date__quarter__eq" : "1"}`        | `WHERE EXTRACT('QUARTER' FROM pub_date) = '1'`    | 一年中的季度              |
| time         | `{"pub_date__time__eq" : "12:59:59"}`    | ``                                                |                      |
| hour         | `{"pub_date__hour__eq" : "12"}`          | `WHERE EXTRACT('HOUR' FROM pub_date) = '12'`      | 小时(0-23)             |
| minute       | `{"pub_date__minute__eq" : "59"}`        | `WHERE EXTRACT('MINUTE' FROM pub_date) = '59'`    | 分钟 (0-59)（别名 `min`）  |
| second       | `{"pub_date__second__eq" : "59"}`        | `WHERE EXTRACT('SECOND' FROM pub_date) = '59'`    | 秒 (0-59)（别名 `sec`）   |
| microsecond | `{"pub_date__microsecond__eq" : "999"}`   | `WHERE EXTRACT('MICROSECOND' FROM pub_date) = '999'` | 微秒（datePartMap 部件，别无别名） |

## 三端序列化器行为与守卫

三端列表页的查询参数经 `PaginationQuery.toRawParams()` 序列化（react / vue-element：
`src/core/transport/rest/pagination.ts`；vue-vben：`apps/admin/src/transport/rest/pagination.ts`。
**三份行为一致但不是同一份文件**：vben 把守卫写成模块级函数、注释与行序也各自漂移过，改一处记得同步另两处），行为约定：

- **字符串值统一转 `__contains`**（裸键 EQ→模糊匹配；contains 是完整值精确匹配的超集）；
- **ID 类字段**（`*_id`/`idXxx`/`^id$`）保持 EQ，不转 contains；
- **`isTenantUser` 时剔除 `tenant_id`/`tenantId` 键**；
- `orderBy` 缺省 `['-created_at']`；`fieldMask` 数组转逗号分隔串；
- **操作符后缀守卫 `hasOperatorSuffix`**：键尾段（**仅小写 snake_case 形态**）已是操作符拼写则不再追加 `__contains`——
  否则 `a__gte__contains` 会被三段解析成 `a CONTAINS value`（语义静默反转）或按未知键报错。驼峰形态的盲区见下节。

**守卫的残余缺口：后缀必须写 snake_case 小写**（2026-09-13 只补齐了拼写集合，没补大小写形态）：

- 后端查表前会做 `ToLower(ToSnakeCase(后缀))`（`go-crud/pagination v0.0.16` 的 `filter/operator_converter.go:106`），
  所以 `field__notEqual`、`field__greaterThan` 这类驼峰拼写**后端是认的**；
- 前端守卫只做 `.toLowerCase()`（三份 `pagination.ts` 的 `hasOperatorSuffix`），认不出驼峰，
  于是给它追加 `__contains` 变成 `field__notEqual__contains`；
- 后端按三段解析时，第二段若不是日期部件就被**整段丢掉**（`query_string_converter.go:366-392`：
  `case 3` 的非 datePart 分支只取 `keys[2]` 当操作符，见 `:383-388`），最终生成 `field CONTAINS value`。

净效果：**"不等于"静默反转成"包含"**，HTTP 200，无日志。写搜索键时后缀一律用表中的规范拼写
（`not_equal` 而非 `notEqual`），或干脆只依赖序列化器自动追加的 `__contains`。

**历史（已于 2026-09-13 修掉的那一半）**：三端守卫表曾未覆盖后端映射表的拼写集合——缺
`i_*` 前缀变体（`istarts_with` 等）、数值比较长拼写（`greater_than` 族）、`not_isnull`、
以及 `json_contains`/`array_contains`/`exists`/`search`/`exact`/`iexact`，后果与上面同形。
现已按 `operatorMap` 的 **snake_case 拼写集合**对齐；库映射表再变更时须同步三端守卫表。

## 项目代码

* [go-wind-admin Gitee](https://gitee.com/tx7do/go-wind-admin)
* [go-wind-admin Github](https://github.com/tx7do/go-wind-admin)
