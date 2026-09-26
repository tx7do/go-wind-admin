# 参数管理（sys_config）：平台全局动态参数

本文档是参数管理子系统的唯一权威说明：数据模型、服务侧缓存读取器、写路径与多实例失效广播、内置参数补种机制、新参数接入步骤。**改口令策略阈值链路、接入新的运行时参数、排"改了参数没生效"之前先读本文。**

定位：平台全局运行时参数的键值存储——管理页可改、服务侧按键读取。与两个邻居划清边界：

- **字典管理**是业务枚举（给前端下拉框用的分类/小类），参数是服务侧行为阈值（如口令复杂度下限），两者语义不同（表注释也如此声明）；
- **环境变量口径已废弃**：阈值类配置一律经参数管理（口令策略曾走环境变量，已迁移，见 `pkg/password/policy.go` 包头注释）。

---

## 1. 数据模型

表 `sys_configs`（`internal/data/ent/schema/config.go`）。ent 层实体叫 **`SysConfig`**——`config` 是 ent 预声明标识符、实体不能叫 `Config`；proto 侧实体名是 `config.service.v1.Config`（路由 `/admin/v1/configs`），两侧类型名不必一致（CopierMapper 只按字段名映射）。

| 字段 | 类型 | 说明 |
|---|---|---|
| `name` | string | 参数名称（展示用） |
| `key` | string | 参数键名，**全局唯一**（`uidx_sys_configs_key`），读取器按键定位 |
| `value` | string | 参数值（统一字符串存储，按 `value_type` 解析） |
| `value_type` | enum | `STRING` / `BOOL` / `INT`（默认 `STRING`）；读取器按声明类型解析，类型不符回退默认值 |
| `is_built_in` | bool | 系统内置参数标记；**内置参数禁删可改** |
| （mixin） | — | `AutoIncrementId` / `TimeAt` / `OperatorID`（审计字段） |

- **无租户维度**：schema 不挂 `TenantID` mixin——参数是平台全局的，不做租户隔离。
- `created_at` 另有索引（`idx_sys_configs_created_at`）供列表分页。
- **proto 零值陷阱**：`value_type` 的零值 `CONFIG_VALUE_TYPE_INVALID` 不在 ent enum 里，直传会触发 `ValueTypeValidator` 失败——repo 的 Create/Update 均带零值守卫（`config_repo.go` `newConfigCreate` / `Update`）。

## 2. 读路径：缓存读取器（accessor）

其他服务经 wiring 注入 `*data.ConfigRepo` 后按键读取（`config_repo.go` 读取器段）：

```go
showCaptcha := s.configRepo.GetConfigBool(ctx, "sys.login.captchaEnabled", true)
maxAge := r.configRepo.GetConfigInt(ctx, passwordPolicy.ConfigKeyMaxAgeDays, passwordPolicy.DefaultMaxAgeDays)
```

三个读取器：`GetConfigBool` / `GetConfigInt` / `GetConfigString`，签名统一为 `(ctx, key, def)`。回退语义：

| 情形 | 行为 |
|---|---|
| 键不存在 | 返回 `def`（**负缓存**：键不存在也缓存，防不存在的键反复打库） |
| 缓存命中（含负缓存） | 不查库，直接返回 |
| 声明类型与读取器不符 | warn 日志 + 返回 `def` |
| 值解析失败（如 INT 列塞了非数字） | warn 日志 + 返回 `def` |
| 查库异常 | 不写缓存（下次重试），返回 `def` |

缓存为**进程内 per-key 懒加载**，条目上限 4096（`sysConfigCacheMaxEntries`）：合法参数键是管理员配置的有限集合，超限说明有调用在拿随机键打缓存，此时放弃缓存该键、每次查库兜底，防内存被刷爆。

## 3. 写路径与多实例失效广播

所有写路径（Create / Update / Delete / SeedDefaults）在落库后执行 `invalidateCacheKey`：

1. 删除**本进程**缓存里的对应键（含负缓存条目）；
2. 向 Redis Pub/Sub 频道 **`gowind:config:invalidate`** 发布该键名；

每个实例的 repo 构造时起一个订阅 goroutine（`subscribeInvalidations`），收到其他实例的失效通知后清除本进程对应缓存条目——**多实例部署下参数变更即时全局生效**。连接断开由 go-redis 自动重连；广播是尽力而为：Redis 不可用时 Publish 仅告警，本地失效不受影响。

细节：

- **改键场景**：Update 在 `UpdateX` 前快照旧键与新键（`FilterByFieldMask` 会在调用中清掉不在掩码里的字段），两键都失效（`config_repo.go:280-281`）。
- 重复键 → 400 `config key already exists`（唯一约束 `IsConstraintError` 转译）。
- `allow_missing: true` 时 Update 不存在即转 Create（upsert）。
- 删除幂等：目标不存在视为已删除；`is_built_in = true` 的行拒删（400 `built-in config cannot be deleted`）。

## 4. 内置参数播种（键级"缺一补一"）

`ConfigService.init()`（进程启动，SystemViewer 上下文）调 `ConfigRepo.SeedDefaults(ctx, constants.DefaultConfigs)`。与其他默认数据的**表级 `count == 0` 守卫不同**，播种按**键**判断：

- 键不存在 → 按默认值创建（缺一补一，管理员自建行不阻断补种）；
- 键已存在 → 跳过，**绝不把管理员改过的值重置回默认**。

种子定义在 `pkg/constants/default_data.go` 的 `DefaultConfigs`，键与默认值**单源引自 `pkg/password` 的同名常量**——种子行与读取器缺省回退永远一致：

| 键 | 默认 | 语义 |
|---|---|---|
| `sys.password.minLen` | 8 | 口令最小长度 |
| `sys.password.maxAgeDays` | 90 | 口令有效期天数（≤0 停用有效期检查） |
| `sys.password.historyCount` | 3 | 历史口令保留条数（≤0 停用历史检查） |

三条均为 `is_built_in = true`（禁删可改）。当前生产消费方：`user_credential_repo.go` / `user_credential_password_policy.go`（口令创建与改密时读阈值）。

## 5. 管理页与接口

- BFF：`api/protos/admin/service/v1/i_config.proto`，标准 5 RPC，路由前缀 `/admin/v1/configs`。
- 三端管理页（菜单 `menu.system.config`，系统管理 → 参数管理）：增删改查 + 列表；内置参数不提供删除入口。
- 管理页改值 → 走同一套写路径 → 失效广播自动生效，无需重启进程。

## 6. 接入新参数的步骤

1. 在 `pkg/constants/default_data.go` 的 `DefaultConfigs` 追加键（内置参数把默认值常量与消费方回退值**单源**放同一处，如 `pkg/password` 的做法）；
2. 消费方经 wiring 注入 `*data.ConfigRepo`，按键 `GetConfigXxx(ctx, key, def)` 读取——`def` 必须与种子默认值一致；
3. 重新部署。老库启动时 `SeedDefaults` 自动补种，**不需要手工 SQL**；
4. 键命名沿用现状 `sys.<域>.<名>`（如 `sys.password.minLen`）；`value_type` 按语义选，读取器与声明保持一致。

## 7. 边界与已知残留

- **写路径前提**：失效链路假设全部写路径都经 `ConfigRepo`——绕过 repo 直改表的写入不会触发本地失效与广播；新增写路径时必须走 repo。
- 参数变更**不是**六类审计日志中的独立类别；管理页改动落操作日志。
- 无变更历史/版本表，回滚靠管理员改回。
- `sys.login.captchaEnabled` 一类键目前只出现在测试与注释示例里，**不是**已接线的生产消费方。

## 8. 相关文件索引

| 文件 | 说明 |
|---|---|
| `backend/app/admin/service/internal/data/ent/schema/config.go` | `SysConfig` 实体（表 `sys_configs`、唯一键、value_type 枚举） |
| `backend/app/admin/service/internal/data/config_repo.go` | 读写路径、缓存读取器、失效广播、`SeedDefaults` |
| `backend/app/admin/service/internal/service/config_service.go` | BFF 服务实现 + 启动期播种触发点 |
| `backend/api/protos/config/service/v1/config.proto` | domain proto（消息与领域服务） |
| `backend/api/protos/admin/service/v1/i_config.proto` | BFF proto（`/admin/v1/configs` 路由） |
| `backend/pkg/constants/default_data.go` | `DefaultConfigs` 内置参数种子（单源引用 `pkg/password` 常量） |
| `backend/pkg/password/policy.go` | 口令策略阈值常量与消费示例（键清单的权威来源） |
