# 文档索引（docs/）

本目录是 GoWind Admin 的全部项目文档入口。文档分**两层**，各有分工：

| 层 | 位置 | 定位 | 读者 |
|---|---|---|---|
| **教程层** | [`tutorial/`](./tutorial/) | 循序渐进的学习路径：每章一个目标，讲清概念与链路，细节链接到参考层 | 从零接手本项目的开发者（采用者 / 新成员） |
| **参考层** | 本目录下各专题文档 | 每篇一个主题的**唯一权威**说明：完整机制、约定、运维边界 | 动手改对应子系统前必读（维护者 / 贡献者） |

**规则：两层冲突时，以参考层与代码为准，修教程层。** 教程只负责"路径与概念"，不承载细节——细节变更只改参考层，教程层的链接与叙述保持低频维护。

---

## 教程层（按序学习）

| 章 | 主题 | 前置 | 你将得到 |
|---|---|---|---|
| [01 架构全景](./tutorial/01-architecture-overview.md) | monorepo 布局、一次请求的完整路径、横切系统地图 | 无 | 知道代码都在哪、请求怎么流 |
| [02 从零跑起来](./tutorial/02-get-it-running.md) | 本地开发环境搭建、三端任选其一登录进系统 | 01 | 一个可调试的本地环境 |
| [03 代码生成链路](./tutorial/03-codegen-chain.md) | proto→Go、proto→TS、schema→ent 三条生成链与边界 | 02 | 知道哪些代码是生成的、哪些是手写的 |
| [04 第一个业务模块](./tutorial/04-first-crud-module.md) | 端到端新增一个 CRUD 模块（后端+前端） | 03 | 亲手跑通一遍完整开发闭环 |
| [05 权限模型](./tutorial/05-permission-model.md) | 认证与令牌、路由/按钮/字段可见性、authz 引擎 | 04 | 理解"谁能看到什么、调到什么" |
| [06 多租户与行级隔离](./tutorial/06-multi-tenant-isolation.md) | Api 表闸门、数据层租户谓词、数据范围五档 | 05 | 理解"行级谁的数据"与部署闸门 |
| [07 审计与等保](./tutorial/07-audit-compliance.md) | 六类审计日志的采集与留存、口令策略/MFA/限流 | 01 | 知道审计能力怎么用、边界在哪 |
| [08 脚本系统入门](./tutorial/08-script-system.md) | 五类扩展点、安全模型、最小示例 | 02 | 会用脚本扩展平台行为而不发版 |
| [09 部署上线](./tutorial/09-deployment.md) | 部署形态决策、SSE 网关、生产密钥、备份 | 02 | 把系统安全地放到生产 |

### 按角色的推荐路线

- **全栈采用者**（拿脚手架做自己产品）：01 → 02 → 03 → 04 → 05 → 06 → 09；07/08 按需。
- **后端开发者**：01 → 02 → 03 → 04 → 06 → 07，再按子系统读参考层。
- **前端开发者**：01 → 02 → 03（重点 TS 生成链）→ 04（前端半）→ 05，再读对应端 `frontend/admin/*/AGENTS.md`。
- **运维 / 安全**：01 → 02 → 07 → 09，参考层 `backend_deploy.md`、`audit-log-producer-design.md`。
- **给本仓提 PR 的贡献者**：全部教程 + 根 `AGENTS.md` 与对应端 `AGENTS.md`（仓库约定以 AGENTS.md 为准）。

---

## 参考层（按主题查阅）

| 文档 | 主题 | 什么时候读 |
|---|---|---|
| [backend_project_struct.md](./backend_project_struct.md) | backend 目录结构与各目录职责 | 改动涉及新目录/新文件位置时 |
| [list_query_rule.md](./list_query_rule.md) | 列表查询协议：分页、排序、过滤操作符、字段掩码 | 写列表/搜索前后端时 |
| [frontend_authority.md](./frontend_authority.md) | 前端权限：路由访问模式、按钮权限码/角色、字段级权限 V1 | 改路由/按钮可见性/受控字段时 |
| [authentication.md](./authentication.md) | 认证与令牌链路：登录全流程、令牌结构与配置、刷新轮换与 Cookie、机器令牌、MFA、限流/策略/验证码、会话吊销 | 改登录/令牌/刷新/MFA/限流/策略前 |
| [tenant_isolation.md](./tenant_isolation.md) | 多租户隔离：上下文链路、HTTP 闸门、数据层读写隔离、套餐联动、覆盖边界、接入与排障 | 改隔离层/Api 表/套餐门禁、新表接租户前 |
| [plan_billing.md](./plan_billing.md) | 套餐与计费管控：三档到期策略全链路、模块白名单、配额与用量计量、租户数据清理 | 改套餐/配额/到期处置、租户 403 排障前 |
| [task_system.md](./task_system.md) | 任务调度系统：配置与启动链、任务数据模型、调度生命周期、系统级常驻任务、脚本任务桥、管理页、多租户语义与排障 | 加任务类型、排"任务没跑"、接新调度需求前 |
| [data_scope_design.md](./data_scope_design.md) | 角色级数据范围：五档语义、聚合、执行层、新表接入步骤 | 新表接入数据范围或改聚合规则前 |
| [audit-log-producer-design.md](./audit-log-producer-design.md) | 六类审计日志的生产者设计与采集层实施状态 | 改审计采集、加新日志字段时 |
| [script_system.md](./script_system.md) | 脚本级插件系统：五类扩展点、安全模型、运维 | 写脚本/改钩子点/接任务桥前 |
| [backend_file_upload.md](./backend_file_upload.md) | 文件上传链路：MinIO 部署、`oss.yaml`、安全校验 | 涉及上传/下载/对象存储部署时 |
| [backend_deploy.md](./backend_deploy.md) | 部署：Docker 两模式、PM2、SSE 网关、生产密钥、hosts | 部署或改部署脚本时 |
| [backend_development_environment_preparation.md](./backend_development_environment_preparation.md) | 后端开发工具安装、Go 模块代理配置 | 新机器搭后端环境时 |
| [frontend_development_environment_preparation.md](./frontend_development_environment_preparation.md) | 前端开发工具安装、npm 镜像源 | 新机器搭前端环境时 |
| [windows-startup-guide.md](./windows-startup-guide.md) | Windows 本地从零启动全流程与常见问题 | Windows 开发机首次搭建时（含 FAQ） |
| [design-language.md](./design-language.md) | 三端视觉唯一权威值表（颜色/圆角/布局尺寸） | 改任何视觉相关代码前 |
| [brand/](./brand/) | 品牌资产（logo / favicon / 锁版 / wordmark） | 换品牌或引用素材时 |

另：仓库级开发约定（铁律、门禁、工具链）在根 [`AGENTS.md`](../AGENTS.md) 与各端 `AGENTS.md`，属于维护层文档，不在本目录。

---

## 维护约定

1. **功能变更时**：改参考层对应文档；检查教程层的链接是否仍成立；叙述性内容尽量不动。
2. **发现教程与代码不符**：以代码与参考层为准修教程，并在 PR 里注明。
3. **新增专题文档**：加入参考层清单表格；如属于某个教程章的深读范围，在对应章补链接。
4. **新增教程章**：更新本索引与 `tutorial/README.md` 的地图，保持前置依赖标注准确。
