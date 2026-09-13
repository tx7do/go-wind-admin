# 列表查询规则

## 通用列表查询请求

| 字段名       | 类型        | 格式                                  | 字段描述    | 示例                                                                                                       | 备注                                                               |
|-----------|-----------|-------------------------------------|---------|----------------------------------------------------------------------------------------------------------|------------------------------------------------------------------|
| page      | `number`  |                                     | 当前页码    |                                                                                                          | 默认为`1`，最小值为`1`。                                                  |
| pageSize  | `number`  |                                     | 每页的行数   |                                                                                                          | 默认为`10`，最小值为`1`。                                                 |
| query     | `string`  | `json object` 或 `json object array` | AND过滤条件 | json字符串: `{"field1":"val1","field2":"val2"}` 或者`[{"field1":"val1"},{"field1":"val2"},{"field2":"val2"}]` | `map`和`array`都支持，当需要同字段名，不同值的情况下，请使用`array`。具体规则请见：[过滤规则](#过滤规则) |
| or        | `string`  | `json object` 或 `json object array` | OR过滤条件  | 同 AND过滤条件                                                                                                |                                                                  |
| orderBy   | `string`  | `json string array`                 | 排序条件    | json字符串：`["-create_time", "type"]`                                                                       | json的`string array`，字段名前加`-`是为降序，不加为升序。具体规则请见：[排序规则](#排序规则)      |
| noPaging  | `boolean` |                                     | 是否不分页   |                                                                                                          | 此字段为`true`时，`page`、`pageSize`字段的传入将无效用。                          |
| fieldMask | `string`  | 其语法为使用逗号分隔字段名                       | 字段掩码（SELECT 投影） | 例如：id,realName,userName。                                                                                 | 落到 List/Get 的 SELECT 投影（库 `fieldSelector.BuildSelector` / `builder.Select`）；为空时为`*`。字段级权限（hfs）会先自此掩码剔除黑名单字段，见 [frontend_authority.md](./frontend_authority.md)。 |

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

| 查找类型        | 示例                                                            | SQL                                                                                                                                                                                                                       | 备注                                                                                                            |
|-------------|---------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------|
| not         | `{"name__not" : "tom"}`                                       | `WHERE NOT ("name" = "tom")`                                                                                                                                                                                              | `not` 是 `ne` 族的别名（ne/neq/not/not_equal/not_equals），见下方别名说明 |
| in          | `{"name__in" : "[\"tom\", \"jimmy\"]"}`                       | `WHERE name IN ("tom", "jimmy")`                                                                                                                                                                                          |                                                                                                               |
| not_in      | `{"name__not_in" : "[\"tom\", \"jimmy\"]"}`                   | `WHERE name NOT IN ("tom", "jimmy")`                                                                                                                                                                                      |                                                                                                               |
| gte         | `{"create_time__gte" : "2023-10-25"}`                         | `WHERE "create_time" >= "2023-10-25"`                                                                                                                                                                                     |                                                                                                               |
| gt          | `{"create_time__gt" : "2023-10-25"}`                          | `WHERE "create_time" > "2023-10-25"`                                                                                                                                                                                      |                                                                                                               |
| lte         | `{"create_time__lte" : "2023-10-25"}`                         | `WHERE "create_time" <= "2023-10-25"`                                                                                                                                                                                     |                                                                                                               |
| lt          | `{"create_time__lt" : "2023-10-25"}`                          | `WHERE "create_time" < "2023-10-25"`                                                                                                                                                                                      |                                                                                                               |
| range       | `{"create_time__range" : "[\"2023-10-25\", \"2024-10-25\"]"}` | `WHERE "create_time" BETWEEN "2023-10-25" AND "2024-10-25"` <br>或<br> `WHERE "create_time" >= "2023-10-25" AND "create_time" <= "2024-10-25"`                                                                             | 需要注意的是: <br>1. 有些数据库的BETWEEN实现的开闭区间可能不一样。<br>2. 日期`2005-01-01`会被隐式转换为：`2005-01-01 00:00:00`，两个日期一致就会导致查询不到数据。 |
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
| like        | `{"name__like" : "a"}`                                        | `WHERE name LIKE 'a';`                                                                                                                                                                                                    | LIKE，无自动通配（区别于 contains 族自动加 `%`）                                                              |
| ilike       | `{"name__ilike" : "a"}`                                       | `WHERE name ILIKE 'a';`                                                                                                                                                                                                    | 同上，大小写不敏感（别名 `i_like`）                                                                            |
| not_like    | `{"name__not_like" : "a"}`                                    | `WHERE name NOT LIKE 'a';`                                                                                                                                                                                                 | 取反 LIKE（别名 `notlike`）                                                                                    |
| json_contains | `{"data.foo__json_contains" : "x"}`                          | （JSON 列）路径包含判定                                                                                                                                                                                                    | JSON 路径字段（`{field}.{jsonpath}` 形态）专用                                                                |
| array_contains | `{"tags__array_contains" : "x"}`                           | （数组列）成员包含判定                                                                                                                                                                                                     | 数组类型列专用                                                                                                |
| exists      | `{"data.foo__exists" : true}`                                 | （JSON 列）路径存在性判定                                                                                                                                                                                                  | JSON 路径字段专用                                                                                            |
| regex       | `{"title__regex" : "^(An?\|The) +"}`                          | MySQL: `WHERE title REGEXP BINARY '^(An?\|The) +'`  <br> Oracle: `WHERE REGEXP_LIKE(title, '^(An?\|The) +', 'c');`  <br> PostgreSQL: `WHERE title ~ '^(An?\|The) +';`  <br> SQLite: `WHERE title REGEXP '^(An?\|The) +';` |                                                                                                               |
| iregex      | `{"title__iregex" : "^(an?\|the) +"}`                         | MySQL: `WHERE title REGEXP '^(an?\|the) +'`  <br> Oracle: `WHERE REGEXP_LIKE(title, '^(an?\|the) +', 'i');`  <br> PostgreSQL: `WHERE title ~* '^(an?\|the) +';`  <br> SQLite: `WHERE title REGEXP '(?i)^(an?\|the) +';`   |                                                                                                               |
| search      |                                                               |                                                                                                                                                                                                                           |                                                                                                               |

**别名族与权威来源**：上表为各族规范拼写。解析器（go-crud `filter/operator_converter.go` 的
`operatorMap`）另接受整族别名——数值比较的长拼写（`greater_than`/`greater_than_or_equal`/`less_than`
等）、`i_*` 前缀变体（`i_contains` 等）、`nin`/`notin`、`is_not_null` 拼写族、`regexp`/`regex` 互换等。
**操作符集以该映射表为唯一权威**，本表不逐一枚举别名。

**解析行为（fail-closed）**：未知操作符、以及超过三段的键（`field__op1__op2__extra`）一律报错
（HTTP 400 invalid parameter），不会静默丢弃条件。三段键是两种带参数形态：
`{field}__{datepart}__{op}` 为日期部件查询（下表，**第三段比较操作符必填**，缺省即按未知键报错）；
`{field}.{jsonfield}__{op}` 为 JSON 路径查询（点号分隔路径）。

以及将日期提取出来的查找类型：

| 查找类型         | 示例                                   | SQL                                               | 备注                   |
|--------------|--------------------------------------|---------------------------------------------------|----------------------|
| date         | `{"pub_date__date__eq" : "2023-01-01"}`  | `WHERE DATE(pub_date) = '2023-01-01'`             |                      |
| year         | `{"pub_date__year__eq" : "2023"}`        | `WHERE EXTRACT('YEAR' FROM pub_date) = '2023'`    | 哪一年                  |
| iso_year     | `{"pub_date__iso_year__eq" : "2023"}`    | `WHERE EXTRACT('ISOYEAR' FROM pub_date) = '2023'` | ISO 8601 一年中的周数      |
| month        | `{"pub_date__month__eq" : "12"}`         | `WHERE EXTRACT('MONTH' FROM pub_date) = '12'`     | 月份，1-12              |
| day          | `{"pub_date__day__eq" : "3"}`            | `WHERE EXTRACT('DAY' FROM pub_date) = '3'`        | 该月的某天(1-31)          |
| week         | `{"pub_date__week__eq" : "7"}`           | `WHERE EXTRACT('WEEK' FROM pub_date) = '7'`       | ISO 8601 周编号 一年中的周数 |
| week_day     | `{"pub_date__week_day__eq" : "tom"}`     | ``                                                | 星期几                  |
| iso_week_day | `{"pub_date__iso_week_day__eq" : "tom"}` | ``                                                |                      |
| quarter      | `{"pub_date__quarter__eq" : "1"}`        | `WHERE EXTRACT('QUARTER' FROM pub_date) = '1'`    | 一年中的季度              |
| time         | `{"pub_date__time__eq" : "12:59:59"}`    | ``                                                |                      |
| hour         | `{"pub_date__hour__eq" : "12"}`          | `WHERE EXTRACT('HOUR' FROM pub_date) = '12'`      | 小时(0-23)             |
| minute       | `{"pub_date__minute__eq" : "59"}`        | `WHERE EXTRACT('MINUTE' FROM pub_date) = '59'`    | 分钟 (0-59)            |
| second       | `{"pub_date__second__eq" : "59"}`        | `WHERE EXTRACT('SECOND' FROM pub_date) = '59'`    | 秒 (0-59)             |
| microsecond | `{"pub_date__microsecond__eq" : "999"}`   | `WHERE EXTRACT('MICROSECOND' FROM pub_date) = '999'` | 微秒（datePartMap 部件，别无别名） |

## 三端序列化器行为与守卫

三端列表页的查询参数经 `PaginationQuery.toRawParams()`（react/vue-element：
`core/transport/rest/pagination.ts`；vue-vben：`apps/admin/src/transport/rest/pagination.ts`，
三份同构）序列化，行为约定：

- **字符串值统一转 `__contains`**（裸键 EQ→模糊匹配；contains 是完整值精确匹配的超集）；
- **ID 类字段**（`*_id`/`idXxx`/`^id$`）保持 EQ，不转 contains；
- **`isTenantUser` 时剔除 `tenant_id`/`tenantId` 键**；
- `orderBy` 缺省 `['-created_at']`；`fieldMask` 数组转逗号分隔串；
- **操作符后缀守卫 `hasOperatorSuffix`**：键尾段已是操作符拼写则不再追加 `__contains`——
  否则 `a__gte__contains` 会被三段解析成 `a CONTAINS value`（语义静默反转）或按未知键报错。

**守卫缺口（已于 2026-09-13 修复）**：三端守卫表曾未覆盖后端映射表的全部拼写——缺
`i_*` 前缀变体（`istarts_with` 等）、数值比较长拼写（`greater_than` 族）、`not_isnull`、
以及 `json_contains`/`array_contains`/`exists`/`search`/`exact`/`iexact`，
后果是 `field__exact` 一类后缀键被追加 `__contains` 后经三段解析成 `field CONTAINS value`
（精确匹配被静默转成模糊、字段名被改写）或报错。修复：三端守卫表与库 `operatorMap`
逐字对齐（含小写归一），后续库映射表变更须同步三端守卫表。

## 项目代码

* [go-wind-admin Gitee](https://gitee.com/tx7do/go-wind-admin)
* [go-wind-admin Github](https://github.com/tx7do/go-wind-admin)
