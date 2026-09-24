# 认证与令牌链路（Authentication & Tokens）参考文档

> **定位**：本仓认证机制的唯一权威说明——登录链路全流程、令牌结构与配置、刷新轮换与 Cookie、
> 机器令牌（AK/SK）、MFA、限流/策略/验证码等辅助闸门、会话管理与吊销、白名单端点、运维与已知问题。
> 前端权限消费侧（路由/按钮/字段黑名单）见 [frontend_authority.md](./frontend_authority.md)；
> 租户隔离与数据范围另见 [tenant_isolation.md](./tenant_isolation.md)、[data_scope_design.md](./data_scope_design.md)。

## 1. 组件地图

```
前端（三端：react/vue-element/vue-vben）
  ├─ 登录表单（验证码图 + AES 加密口令 + Header 携带验证码）
  ├─ access token：仅内存（zustand/Pinia/模块级 store）
  ├─ refresh token：HttpOnly Cookie（JS 不可读）；refresh_exp cookie（可读）供静默恢复/定时器
  └─ 401 → 单飞刷新队列 → /admin/v1/refresh-token（Cookie 自动携带）→ 重放
后端
  ├─ AuthenticationScheme（authn）+ Authorizer（authz）引擎配置：configs/auth.yaml（authn.jwt / authz.type）
  ├─ internal/data/authenticator.go —— 令牌签发/验签/吊销/轮换/缓存（engine 适配层）
  ├─ internal/service/authentication_service.go —— 登录/刷新/验证码/找回
  ├─ internal/service/mfa_service.go —— TOTP 绑定与登录挑战
  ├─ internal/service/access_key_service.go —— AK/SK 机器令牌交换
  ├─ internal/data/login_rate_limiter.go —— Redis 双维度失败计数限流
  ├─ internal/data/login_policy_checker.go —— 登录策略匹配（纯函数+单测）
  └─ pkg/middleware/auth/auth.go —— 每请求认证（验签+缓存核对）+ ViewerContext/Operator 注入
```

## 2. 登录链路（`doGrantTypePassword`，按代码顺序）

```
POST /admin/v1/login（grant_type=password）
  ├─ 0  ctx 复位：NoopContext + privacy.Allow（登录查询绕过租户隔离——显式豁免通道）
  ├─ 1  限流预检：IP + 用户名双维度 Redis 计数（锁定期直接 400）
  ├─ 2  图形验证码：X-Captcha-Id/X-Captcha-Value Header，Redis verify-and-delete（单次有效）
  ├─ 3  租户解析：tenant_code 空=平台（tid 0）；非空查 sys_tenants，非 ON 统一文案拒绝
  ├─ 4  登录策略·全局段：target_id 为空的策略（IP/时间窗/设备），被封锁 IP 连 user 表都不查
  ├─ 5  identifier 反查：输入含 @ → email、纯数字 → mobile，反查真实 username；多行歧义拒绝
  ├─ 6  凭证校验（user_credential_repo.FindUserCredential）：
  │      AES 解密（DefaultAESKey）→ 按 (tenant, USERNAME, username) 查凭证 → bcrypt 比对；
  │      用户不存在时跑一次假 bcrypt（恒定时间防枚举）
  │      失败 → 限流计数自增 + 统一 INVALID_PASSWORD 文案（防用户名枚举，真实原因进日志/审计）
  ├─ 7  取用户行：按凭证返回的 user_id 精确 Get（避免同 identifier 多租户歧义）
  ├─ 8  纵深防御：凭证行 tenant 必须等于用户行 tenant，否则 "invalid tenant"
  │      （`authentication_service.go:549-554`——它在取到用户行之后，不在凭证库里）
  ├─ 9  登录策略·用户定向段：target_id = 该 userId 的策略条目（`:558-563`，命中即 Forbidden）
  ├─ 10 授权丰富（`resolveUserAuthority`，调用点 `:574`）：
  │      **先**查 `user.status != NORMAL` 即拒（`:451-455`，Forbidden "user is disabled"——
  │      所以状态检查实际发生在策略段**之后**，不是之前）；
  │      再 authorizeAndEnrich*（一对一/一对多按 DefaultUserTenantRelationType）：
  │      角色链（user→role / membership→role）→ 权限码集合须含 SystemAccessBackendPermissionCode
  │      → 角色 codes 进 token；fillAdminFlags（平台/租户管理员旗标）；
  │      聚合数据范围（dss/dsu）与字段黑名单（hfs）进 token（见对应设计文档）
  ├─ 11a MFA 闸门：用户绑有 ENABLED TOTP → 不发 token，签发 operation_id（挑战缓存），
  │      前端走 /admin/v1/mfa/challenge 二次验证（见第 5 章）——限流计数此时不清零
  └─ 11b 签发：CreateUserToken（access + refresh，jti，Redis 令牌对入缓存）
         → 会话元数据记录 + last_login 记录（均 best-effort）
         → 限流计数清零
         → refresh 经 HttpOnly Cookie 下发（见第 4 章），响应体只回 access + expires_in
```

错误语义统一化（防枚举）：`normalizeLoginVerifyError`（`authentication_service.go:149-158`）把
USER_NOT_FOUND / USER_FREEZE / INVALID_PASSWORD 归并为同文案；租户编号错误同文案；忘记密码对不存在邮箱同返回成功。
真实原因保留在服务端日志与登录审计的 FailureReason。

⚠ **一处已知的归并缺口**：口令有效期到期时 `FindUserCredential` 返回的是
`ErrorBadRequest("password expired, please reset your password")`（`user_credential_repo.go:458-465`，
阈值取 sys_config `passwordPolicy.ConfigKeyMaxAgeDays`），而 `normalizeLoginVerifyError` 的 `switch`
只认上面三类，`default` 原样上抛。由于这条错误发生在 bcrypt **比对成功之后**，它实际上告诉调用方
"这个账号的口令是对的、只是过期了"——防枚举在该场景下被削弱。要补的话就是把这条 reason 也纳入归并
（并保留日志侧的真实原因）。

`grant_type=refresh_token` 在 login 端点被拒（引导到专用刷新端点）；
`grant_type=client_credentials` 被拒（机器令牌只走 AK/SK 交换，第 6 章）。

## 3. 令牌体系

### 3.1 结构（`authentication/service/v1/user_token.proto`，自描述 JWT）

| claim | 字段 | 说明 |
|---|---|---|
| `uid` | user_id | 用户 ID（机器令牌恒 0） |
| `tid` | tenant_id | 租户 ID；0=平台上下文 |
| `cid` / `did` | client_id / device_id | 客户端/设备标识（登录策略 DEVICE 维度用 did） |
| `sub` | username | 机器令牌为 `ak:<AK>` |
| `roc` | roles | 角色码列表（机器令牌 `["machine"]`） |
| `ipa` / `ita` | is_platform_admin / is_tenant_admin | 管理员旗标（fillAdminFlags 产出；MFA 重置/越权校验消费） |
| `ds`/`dss`/`dsu` | data_scope(s)/unit_ids | 数据范围聚合（语义见 data_scope_design 第 3 节） |
| `ouid` | org_unit_id | 当前组织单元 |
| `hfs` | hidden_fields | 字段黑名单（"资源.字段"，语义见 frontend_authority 字段级一节） |
| `jti` | jti | 令牌对唯一标识（吊销/轮换/会话元数据键） |

### 3.2 配置（`configs/auth.yaml`）与引擎

- `authn.type: jwt`（oidc / preshared_key 配置节为预留，未接线）；
- `authn.jwt.method`：本仓 `configs/auth.yaml:8` 配的是 **RS256**；代码里的兜底值才是 HS256
  （`method` 为空时，`authenticator.go:126-129`）。取值直接用字面量：`WithSigningMethod` 走
  `jwtV5.GetSigningMethod(alg)`（`kratos-authn/engine/jwt v1.1.11` 的 `options.go:16-18`），
  所以合法拼写就是 golang-jwt 注册名——HS/RS/PS/ES 各 256/384/512 与 **`EdDSA`**。
  ⚠ `auth.yaml:5` 的注释把最后一个写成 `Ed25519`，那不是注册名，照抄会得到不支持的算法；
  本仓的 `isAsymmetricMethod`（`:169-174`）也只认 `EDDSA`；对称走 `key`，
  非对称走 PEM `private_key`/`public_key`；
- TTL：`access_token_expires: 5400s`（1.5h）、`refresh_token_expires: 43200s`（12h）——
  protobuf Duration 串，未配置用代码默认；
- **生产密钥注入**：环境变量 `GWA_AUTH_JWT_PRIVATE_KEY` / `GWA_AUTH_JWT_PUBLIC_KEY` /
  `GWA_AUTH_JWT_KEY` 优先于 yaml（yaml 内置开发示例密钥，`authenticator.go`
  `applyJwtKeyOverrides` 应用覆盖并告警）；轮换命令见 [backend_deploy.md](./backend_deploy.md)；
- **引擎只装配了 admin 一套**（`Authenticator.AdminAuthenticator`，`authenticator.go:71` / 构造于 `:110`）。
  `getAuthenticator`（`:662-673`）里 `case ClientType_app:` 是**空分支**：不报错、也不赋引擎，返回
  `(nil, nil)`——而调用方一律只判 `err`（如 `:252-255`、`:486-492`），拿到 nil 引擎后直接调方法即
  **运行期空指针**。装配侧 `NewClientType()` 恒返回 admin（`internal/data/data.go:21-23`），所以
  只有**请求显式带 `client_type: app`** 才会走到这条路（登录入参的 client_type 由前端传值，刷新端点强制 admin）。
  → **app 端未实现**：前端/配置不要传 app；要接 app 必须先补这个分支。

### 3.3 签发与吊销模型（`internal/data/authenticator.go`）

- `CreateUserToken`：jti → access + refresh 双 JWT → `userTokenCache.AddTokenPair`
  （Redis，键含 clientType+uid+jti，TTL 各按配置）；
- **每请求校验**（`Authenticate`，被 `pkg/middleware/auth` 的 accessTokenChecker 调用）：
  JWT 验签 → 过期检查 → **Redis 缓存核对**（`IsValidAccessToken`）——
  令牌不在缓存（已吊销/被轮换掉/未入缓存）即拒。因此**吊销即时生效**（删缓存键），
  密钥轮换 + 吊销 = 全量下线；
- 吊销面：`RevokeUserToken`（按 uid，登出用）、`RevokeTokenByJti`（单会话，在线用户强制下线用）、
  `RevokeUserTokenAllClientTypes`（改密踢人等）；
- `CreateMachineToken`：**只签 access**、入缓存、不进会话元数据（机器身份无用户行）。

## 4. 刷新链路（HttpOnly Cookie + 轮换）

### 4.1 Cookie 策略（`authentication_service.go` 头部常量 + `setRefreshCookies`）

| Cookie | 属性 | 用途 |
|---|---|---|
| `refresh_token` | **HttpOnly**、`Path=/admin/v1/refresh-token`、SameSite=Lax、Secure 按 TLS 自适应 | 刷新凭据本体：JS 读不到、仅刷新请求自动携带（纵深防御） |
| `refresh_exp` | 非 HttpOnly、`Path=/`、仅 Unix 秒 | 前端静默恢复/刷新定时器读的过期时间戳（无敏感信息） |

Secure 自适应（`resolveCookieSecure`）：TLS 直连或可信反代 `X-Forwarded-Proto: https`
→ true；明文 HTTP → false（否则浏览器拒收、dev 落不了地）。清 Cookie 按各自 Path 匹配删除。
SameSite=Lax 按站点判断——localhost 不同端口同站，dev 直连后端（vue-element/vben 形态）可落地。

### 4.2 刷新端点（`RefreshToken` → `doGrantTypeRefreshToken`）

- 端点在鉴权白名单内（refresh token 自描述 JWT **独立鉴权**，不依赖 access token）；
- Cookie 取 refresh → `VerifyRefreshToken`：JWT 验签 + 过期 + **Lua 脚本原子
  「验证 RT→删除旧令牌对」**（`VerifyAndRevokeTokenPair`）——每次刷新都轮换 jti，
  旧 access/refresh 立即作废；
- 重新走授权丰富（第 2 章第 9 步同款，角色/数据范围/字段黑名单**随每次刷新重算**——
  这是"档位变更最迟随刷新生效"的机制根源）；
- 会话元数据**继承旧会话的登录时间**（旧元数据已随旧令牌对原子删除，新元数据携带 loginAt）；
- 新 refresh 仍走 Cookie，响应体只回新 access。

### 4.3 前端消费（react 实例，`src/bootstrap.ts` + `core/transport/rest/preset-interceptors.ts`）

- 页面加载 bootstrap：读 `refresh_exp` 有效时静默调 `/refresh-token` 换 access
  （"session silently restored via refresh cookie"）；
- 401 拦截：`refreshTokenQueue` 单飞（并发 401 只发一次刷新），成功后重放队列、失败清队列走重登；
  刷新请求自身的 401 不再触发刷新（防死循环）。

## 5. MFA（TOTP）

- 绑定面（`mfa_service.go`）：StartEnrollMethod 产出 otpauth 密钥/二维码 → ConfirmEnrollMethod
  验码落 `sys_user_mfa_factors`（secret 加密存储）；ListEnrolledMethods / GetMFAStatus；
  DisableMFA：不传 user_id 自解绑；传 user_id 指定他人**仅平台管理员**（救援重置，告警留痕）；
  RevokeMFADevice 单设备解绑；
- 登录挑战（VerifyMFAChallenge，白名单端点）：`operation_id` 单次有效挑战缓存
  （Peek 可重试、`RecordLoginFailure` 计错、超限作废；`TakeLoginChallengeAtomic` 原子消耗
  **防双花**——并发同 opId 仅一人得 token）；TOTP 校验 ±1 窗口/30s/6 位/SHA1；
  错码计入登录限流（IP+用户名）；通过后清零限流计数、记 last_login、签发**常规令牌对**；
  审计贯通：本端点请求体无 username，经 `X-Audit-Username` 响应头回传挑战上下文用户名给
  登录审计中间件兜底（非 ASCII 过滤）；
- 中间件探测 `HasEnabledTotp` **fail-closed**（查询失败拒登，不降级单因子）。

## 6. 机器令牌（AK/SK 交换）

`access_key_service.go IssueToken`（`POST`，白名单端点——AK/SK 本身即凭据）：

1. 复用登录限流器（IP+AK 双维度，防爆破）；
2. `GetByAccessKeyBySystem`（SystemViewer 通道）查 AK 行；不存在/密钥错误统一文案；
3. 状态 OFF / `expires_at` 已过 → 拒；
4. SHA-256(secret) 摘要 `subtle.ConstantTimeCompare` 恒定时间比对（明文不落库）；
5. 成功：清限流计数、`CreateMachineToken`（纯 access、入缓存、
   `sub="ak:<AK>"`、`roles=["machine"]`、tid=AK 所属租户）、best-effort 刷新 last_used_at；
6. authz 语义：subject 为 machine 角色——noop 引擎全放行；casbin 下无策略=fail-closed，
   要给机器身份开面须显式配策略。

管理面（AccessKey CRUD/ResetSecret）与隔离现状见 [tenant_isolation.md](./tenant_isolation.md) 第 6 节。

## 7. 辅助闸门

### 7.1 图形验证码（`captchaClient`，go-utils 封装）

- `Generate` → id + base64 图 + answer；**必须显式 `Save` 进 Redis**（键
  `gowind:captcha:<captchaId>`）——漏 Save 则 Verify 永远失败；
- `Verify` 为 verify-and-delete（单次有效）；登录侧经 `X-Captcha-Id`/`X-Captcha-Value`
  Header 传递（避免改动 proto 与三端生成代码）；
- `CaptchaEnabled` 常量当前 true；未配置 captchaClient 时 fail-open（可用性优先）。
  自动化测试可直读 Redis 答案（根 AGENTS.md 的 e2e 做法）。

### 7.2 登录限流（`login_rate_limiter.go`）

- 双维度键：`gowind:login:fail:ip:<ip>` / `gowind:login:fail:user:<username>`（机器令牌交换复用，user 维度=AK）；
- 阈值 5 次 / 锁定窗口 15 分钟 / 计数 TTL=锁定窗口（**固定窗口，不是滑动**：Lua 只在 `INCR` 结果为 1 时
  `EXPIRE`，`login_rate_limiter.go:46-49`，所以 TTL 从**首次失败**起算、后续失败不续期——该文件 `:19-20`
  的注释写成"滑动窗口"是错的）；`CheckAndIncr` 为 Lua 原子
  （锁定判定→自增→TTL）；任一维度达阈值即锁；
- Redis 不可用 → **fail-open**（防御性增强不阻断登录可用性），仅告警；
- 成功（含 MFA 通过）→ `Reset` 清零两维度。

### 7.3 登录策略（`login_policy_service` + `login_policy_checker.go`）

- 存储：`sys_login_policies`（租户级表，`ent/schema/login_policy.go:21`），条目含 TargetID（0=全局/否则定向用户）、
  Method（IP/TIME/DEVICE）、Type（黑/白名单）、Value；
- 匹配（纯函数 `MatchLoginPolicy`，含单测）：黑名单任一命中→拒；白名单存在约束
  （全局或定向当前用户）且当前值未命中任何白名单→拒。IP 支持精确与 CIDR；
  TIME 为 `HH:MM-HH:MM`（支持跨午夜）；DEVICE 精确匹配 device_id；
  **MAC/REGION 未实现**（HTTP 上下文无 MAC；REGION 待接 IP 地理库）；
- 两段式调用：全局段在 identifier 反查与密码校验**之前**（封锁 IP 不消耗查询），
  用户定向段在密码通过、userId 已知后；
- 策略查询失败 → **fail-open**（仅告警——登录可用性优先，与验证码开关同取向）。

### 7.4 忘记密码（`authentication_forgot_password.go`，两白名单端点）

- ForgotPassword：6 位数字 vcode（纳秒取模，一次性语义）→ `vcodeCache.Save("reset_password", identifier, code, 10min)`
  → 经通知渠道（SMTP，见通知渠道管理页）发码；用户不存在同返回成功（防枚举）；
- ResetPasswordByCode：验码（消耗）→ 重置口令（AES 解密入参、bcrypt 存储）→
  **吊销该用户全部会话**。

## 8. 会话管理与在线用户（`online_session_service.go` + `session_meta.go`）

- 会话元数据（签发/刷新时记录，Redis，键 `us:{ct}:{uid}:{jti}`，`user_token_cache.go:33-34`；
  另有刷新令牌 `rt:{ct}:{uid}:{jti}`、黑名单 `bl:{jti}`）：登录时间（轮换继承）、
  clientType、ip/ua 等——「在线用户」「我的会话」两页的数据源；
- ListOnlineSession（平台侧全量）/ ListMyOnlineSession（本人）；
  RevokeMyOnlineSession（自撤销单会话）；**ForceLogoutSession（平台侧强制下线单会话，
  `RevokeTokenByJti` 即时生效）**；
- 密码修改 → 全 clientType 吊销（踢人）；
- `WhoAmI` 返回操作者 uid/username（operator 上下文）；注册入口已整体移除
  （账号一律管理员建，避免半注册孤儿）。

## 9. 免鉴权白名单端点（`rest_server.go` `AddWhiteList`）

| 端点 | 自带防护 |
|---|---|
| Login | 全链闸门（第 2 章） |
| GenerateCaptcha / VerifyCaptcha | 单次有效、无敏感返回 |
| RefreshToken | refresh 自描述 JWT 独立鉴权 + 原子轮换 + 缓存吊销核对 |
| MfaService.VerifyMFAChallenge | operation_id 单次有效原子消耗 + 错码限流 + 审计贯通 |
| ForgotPassword / ResetPasswordByCode | vcode 单次 10min + 防枚举 + 重置后全吊销 |
| AccessKeyService.IssueToken | AK/SK 恒定时间比对 + 限流 + 机器 token 最小面 |

白名单**跳过的是整个 `auth.Server` 中间件**——租户闸门就在这个中间件内部
（`pkg/middleware/auth/auth.go:120-132` 的 `CheckTenantAccess`），而 selector 的语义是
"命中白名单 ⇒ 返回 false ⇒ 不执行该中间件"（`kratos-bootstrap/rpc v0.1.3` 的
`whitelist.go:132-144`，装配点 `rest_server.go:110`）。所以**白名单端点不过租户闸门**，
机器 token（tid>0）在该类端点上也不例外——它后续访问的业务端点才过。

不受影响的两项：**参数校验中间件对所有路由生效**（`validate.Validator()` 放在 selector 之外，`rest_server.go:77`，注释写明了理由），**登录/API 审计照记**（`applogging.Server` 在 selector 之前）。

## 10. 运维

| 事项 | 操作 |
|---|---|
| 生产密钥 | `GWA_AUTH_JWT_*` 环境变量注入（yaml 是开发密钥）；轮换=换钥+吊销全量（缓存核对保证即时失效） |
| TTL 调整 | auth.yaml 两字段（Duration 串）；生效于新签发，存量按旧 TTL |
| 自动化登录 | 直读 Redis `gowind:captcha:<captchaId>` 取答案（或临时置 CaptchaEnabled=false 后编译） |
| 限流排查 | Redis 双维度键计数；误锁清理直接删键 |
| 策略调整 | 登录策略管理页（租户级）；匹配语义见 7.3，改动跑 login_policy_checker_test |
| 审计 | 全部登录尝试落 `sys_login_audit_logs`（成败/原因/归属地），见 audit-log-producer-design |
| 会话处置 | 在线用户页强制下线（单会话）/ 改密自动全吊销 |

## 11. 已知问题

| 项 | 现状 | 影响 |
|---|---|---|
| ~~MFA 登录成功路径 refresh token 走响应体~~ | **已修复（2026-09-13）**：`VerifyMFAChallenge` 改调 `setRefreshCookies`，响应体不再携带 refresh 字段，与主登录路径一致 | 修复前：refresh 暴露在 JS 可读响应体 + MFA 用户会话静默续期丢失（前端只认 Cookie）。修复后 MFA 用户获得与其他用户一致的续期链路 |
| `LoginResponse.RefreshToken`/`RefreshExpiresIn` 字段残留 | proto 字段仍在（历史形态），当前**所有路径均不再赋值** | 字段级清理（proto 删字段 + `make api`/`make ts` 三端重生成）属可选跟进，不影响行为 |
| oidc / preshared_key / oauth proto | 配置节与 proto 预留，未接线 | 接入前勿在生产配置里误以为已启用 |
| MAC / REGION 登录策略维度 | 未实现（7.3） | 管理页如已展示该选项需对齐 |

## 12. 边界（明确不做）

- 注册入口（RegisterUser）已移除，不再恢复（孤儿用户教训）；
- login 端点不做 refresh（专用端点）、不做 client_credentials（专用 AK/SK 端点）；
- access token 不落 localStorage/sessionStorage（仅内存，刷新即失，靠 refresh Cookie 恢复）。
