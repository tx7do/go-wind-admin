# 更新日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 规范，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

> 历史条目基于 Git 提交记录回溯整理，仅记录对使用者有影响的功能性变更，
> 日常重构、代码风格、CI 调整等不在逐一列举。

## [Unreleased]

### 新增
- 新增项目治理文件：`CONTRIBUTING.md`、`SECURITY.md`、`CHANGELOG.md`，
  以及 `.github/` 下的 Issue / PR 模板与行为准则。
- 多因素认证（MFA）：基于 TOTP 的登录挑战、个人中心绑定管理，以及管理员救援重置用户 MFA 的解锁路径。
- 登录支持邮箱 / 手机号作为账号标识，登录策略接入登录主流程。
- SSE 反向代理网关部署包（nginx，透传 `/events` 至后端 SSE transport），独立于 compose 部署。
- React 前端暗色主题色板定稿（暗夜蓝黑底色 + 科技蓝主色）。
- 四类审计日志生产者全链路接通：操作 / 权限 / 数据访问 / 策略评估审计落库，
  含 SQL 词法脱敏、涉及表名与数据分类提取、操作者与目标信息、资源 ID、
  评估上下文快照与 trace_id 关联。
- Vben 审计日志详情抽屉：操作 / 数据 / 权限 / 策略评估日志支持详情查看，
  长文本（SQL、评估上下文、新旧值）以代码块卡片展示。
- 等保合规安全能力：审计日志 180 天留存与每日归档任务（JSONL 导出）、
  数据库定时备份脚本（pg_dump + 30 份轮换，附恢复文档）、
  口令策略三件套（复杂度 / 历史复用 / 有效期，环境变量可调）。
- 站内信广播投递改走 Asynq 异步任务队列（断点恢复 + 幂等投递）；
  消息管理支持发送范围选择（全员 / 指定用户）与消息撤销。

### 变更
- 刷新令牌（refresh token）由本地存储迁移至 HttpOnly Cookie，并改用自描述 JWT。
- CORS 补齐 `allow_credentials`，以适配 HttpOnly Cookie 刷新令牌的跨域携带。
- 安全加固：内容安全策略（CSP）、输入校验、日志注入防护、枚举零值守卫。
- 后端日志统一迁移至 `kratos-bootstrap` logger，移除 refresh cookie 明文 HTTP 落地。
- 操作 / 权限审计剔除 GET 等读请求与登录 / 刷新等会话维护端点，聚焦真实写操作，
  避免页面浏览产生噪音日志。

### 修复
- 修复字典类型 / 字典项创建 500（ent O2M required edge 误写），字典功能全链路恢复。
- 修复改密 / 重置密码漏设解密标志，AES 密文被当明文哈希入库导致改完即锁号。
- 修复 Vben 审计日志页表格渲染崩溃（geoLocation 插槽裸读污染 vnode 树）
  与刷新后会话静默恢复时序问题（恢复完成前被重定向到登录页）。

## [0.x] — 早期开发版本

GoWind Admin 当前处于 `0.x` 阶段，尚无正式发布版本。
下列条目为近期（以 Git 提交记录为准）对使用者有影响的变更摘要。

### 新增
- 基于多租户的 RBAC 体系：用户、租户、角色、权限、菜单、组织、职位管理。
- 套餐与配额管理：租户订阅套餐及其资源配额（模块白名单、用量上限等）的计量与管控。
- 三套前端：React 19 + Ant Design V6、Vue3 + Element Plus、Vue3 + Vben（Ant Design Vue），
  共享同一套由 proto 生成的 TypeScript HTTP 客户端。
- 代码生成管线：proto → Go HTTP/gRPC + OpenAPI + TS 客户端；Ent schema → ORM；
  Wire 依赖注入。
- 文件上传与下载链路（MinIO / 本地存储），含元数据落库与安全校验。
- 异步任务调度（Asynq）、SSE 服务端推送、多语言脚本引擎（Lua / JavaScript）与 Hook 插件系统。
- 登录增加租户编号 `tenant_code` 解析租户。

### 修复
- 修复多租户场景下按非主键字段查询导致的隔离缺陷与 `ent not singular` 登录失败。
- 修复消息全员广播阻塞请求、错误全吞与撤销非原子问题。
- 修复建租户时重复检测失效、租户范围查询未过滤等问题。
- 移除生产路径遗留的 `.Debug()` 与 `syncWithOpenAPI` 的 `Fatal` 调用。
- 修复登录 `loginMethodConverter` 漏注册与生产路径遗留 `.Debug()`。
- 修复文件上传下载、用户关联更新、凭证解密等多处缺陷。
- 禁用预签名上传路径并隐藏登录页注册 / 找回密码入口。
- 编辑器按需懒加载，切断 barrel 对子编辑器的静态 re-export。

### 变更
- 移除 `lucide-vue-next` 依赖。

> 本段为历史摘要，后续正式发版将按 Keep a Changelog 规范逐版本记录。
