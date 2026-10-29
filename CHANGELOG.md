# 更新日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 规范，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

> 历史条目基于 Git 提交记录回溯整理，仅记录对使用者有影响的功能性变更，
> 日常重构、代码风格、CI 调整等不在逐一列举。

## [Unreleased]

## [1.5.0] - 2026-10-29

### 新增

- **BI 仪表盘（设计器 + 大屏渲染）**：图表可视化编排（@go-wind/bi-* 设计器：柱状/折线/
  饼图/仪表盘/雷达等图型、拖拽网格、跨图联动、下钻、九套主题预设），仪表盘/大屏持久化
  入库（`bi_dashboards` 表：payload 为设计器 schema JSON 服务端不解析、版本乐观锁防并发
  覆盖、内置模板禁删可复制派生、租户隔离）。三端同构入口：「BI 分析」→ 仪表盘渲染
  （`?id=N` 渲染库中行）、设计器（`?id=N` 加载库中仪表盘自动保存回写；无 id 本地草稿）、
  仪表盘管理（列表 + 全屏设计器抽屉 + 复制/删除）。演示数据 static 内嵌；真实业务由宿主
  经数据集 resolver 注入。页级全屏（仪表盘/设计器均可）。内置「基础仪表盘模板」启动播种。
  详见 [docs/bi_dashboard.md](./docs/bi_dashboard.md)。
- **打印模板（单据打印设计器）**：可视化拖拽设计打印模板（文本 / 表格 / 条码 / 二维码，毫米精确
  布局，多页跨页表头），内核 fork 自 worm-vue3-print（MIT，go-wind-print 仓自维护）。模板按资源
  绑定（resourceKey），任意列表行可一键打印——模板匹配精确绑定优先、通用模板（未绑定）兜底；
  支持勾选行批量打印（合并为单文档，份间自动分页）。输出链：浏览器打印 / 服务端 PDF
  （`POST /admin/v1/print-templates:render-pdf`，后端 Chromium 渲染，Docker 构建参数
  `WITH_PRINT_BROWSER=1` 启用浏览器层）/ 静默打印（对接 worm 客户端）。三端管理页齐备
  （暗色适配、复制内置模板派生），内置种子含通用 A4/A5 模板与用户工牌演示。设计/部署详见
  [docs/print_template.md](./docs/print_template.md)；采用者给自己的模块接打印见
  [docs/crud_module_guide](./docs/crud_module_guide/README.md) 行打印节。

### 变更

- 亮色模式首次拥有自己的设计 token（react）：此前浅色整端只有暗色一套 token，亮色全走
  antd `defaultAlgorithm` 的派生值——画布、容器、边框、大圆角没有一处是按规范落的，
  "亮色发平、卡片没有浮起感"的真因就在这里。现补 `lightTheme.ts` 与卡片静止/悬浮两档投影。
- 搜索卡与列表卡并到同一层级档（三端）：**投影只表达层级，不表达"这是块卡片"**。react 原先
  搜索卡吃 `shadow-md`、列表一条投影都没有，读起来像搜索卡压在列表上面；vue-element 四张卡
  吃 Element Plus 的 12px 全向晕，α 比本端自己的悬浮档还重、且没有 Y 偏移。现三端静止卡同取
  §2.2 的 `0 1px 2px rgba(0,0,0,.05)`（vue-element 暗色沿用本端 `.el-card` 定过的黑 α 轻投影）。
- 卡片内缩归一为单一来源，内容落到同一条竖线（三端）：搜索字段、工具条、表头文字此前分别落
  45px / 21px / 13px——盒子对齐了、内容没对齐。现统一 20px 内缩，react 单行搜索卡高度
  117→85px。vue-element 的第三条间距来源藏在 JS 里（表格高度计算在 CSS padding 之外又预留
  20px，实测叠成 36px 空档），一并交回 CSS 管。
- 搜索卡→列表卡、表格→分页的间距统一到 16px；分页器下方留白由卡片内缩提供，不再上下各写一份。
  （其后按用户实测反馈把 vue-element 的搜索卡→列表卡收到 12px，这一档三端暂不同值，见 §2.6 该行注。）
- vue-element 卡片圆角 8→12（§2.4 大容器档，三端里它是唯一离群者），骨架卡同步镜像，
  避免"骨架→真卡"切换时跳形；圆角阶梯保持外 > 内（卡片 12 > 输入 8 > 按钮 6 > 表格面 6）。
- 侧栏折叠动效收成一条 240ms 时钟（react）：原先四项参与属性各自时长不同，图标位移 100ms 就
  走完、面板还在动 300ms，读起来"生硬"。路由切换入场过渡与点击波纹同时接回。

### 修复

- 一批"写了但从未生效"的暗色适配规则：react 偏好抽屉等 5 个 `.less` 仍用 antd v5 的 cssVar
  名（v6 下整块空转）、页签栏一段死"暗色适配"块、聊天卡上的白 α 投影（§2.3 要求暗色投影走黑 α）。
- 暗色占位符收回三级档 `#6E7681`（react）：照抄浅色值会让提示语跳到与已输入内容同一优先级上。
- 语义色文字档三端接上同一组值：link / text / plain 按钮与 tag 的前景走 `--gowind-*-text`，
  不再吃基准色（白字在 `#FF3860` 上只有 3.51:1）。
- vue-element 的一条机制错判：`_element-plus.scss` 顶部 `@forward ... with (...)` 的设计 token
  map **到不了运行时**（EP 样式按预编译 CSS 引入），文档原记的"浅色边框 `#E4E4E7` 由该 map 落地"
  从未生效——卡片描边现由本端自建 `--gowind-card-border` 承担。
- 暗色侧栏展开的子菜单面板不再比侧栏更暗（react）：一块 `rgba(0,0,0,.22)` 的"凹陷底"与
  §4 的"父级展开链路只做文字/图标提亮，不铺底色"直接冲突，实测与侧栏只差 1.049:1——起不到
  分组作用、又让三端里只有 react 的菜单看着多出一层。收平走 token 而非删 CSS：antd 出厂的
  `darkSubMenuItemBg` 是 `#000c17`，只删那条 `!important` 反而会露出更黑的藏青井。
- 聊天页两张卡不再挂浮层晕（vue-element）：`.chat-card` 与四张 Pro 卡并列为页面容器，却仍吃
  `--el-box-shadow-light`——亮色 `0 0 12px 12% 黑`、暗色实测 `0 0 12px 72% 黑`，正是本端为
  `.el-card` 定档时点名"会在窄间距卡片间叠出黑缝"的那一档。现改走 `--gowind-card-shadow` /
  `--gowind-card-border`，与 Pro 卡同档（亮 `0 1px 2px 5%` + `#E4E4E7`、暗 `0 1px 2px 36%` + 8% 白），
  明暗两态实测与 react 同值。设计文档原把"聊天页"记在浮层一档，一并纠正。
- 浅色页签选中态不再铺中性灰底（react）：§4「温和配方」给浅色定的是**主色 15% 淡底**（vue-element 早已照做），
  代码却写成 `rgba(128,128,128,.12)`——压白栏合成 `#F0F0F0`，与页面画布 `#F1F3F6` 只差 1.008:1，读起来像白栏
  被抠了个洞而非"这页被选中"；且与 hover 的 `.08` 同色相、只差 α，选中与悬停糊成一片。现改吃 `color-mix`
  淡蓝底（实测 `#D9E9FB`、对栏 1.24），描边退回透明、card 的字重一并收平；暗色那档本就是规范要求的中性灰，逐值未动。
- 暗色认证页占位符不再兼任标签却按注记上色（react）：登录页四个字段与 MFA 的验证码框都没有 `Form.Item label`，
  提示语是字段的唯一可读名称，却吃着全站 placeholder 的"退后"档——三级灰 `#6E7681` 在 L1 面板上只有 3.86:1。
  现只在认证表单作用域、且只在暗色提到次要档 `#8B949E`（实测 5.77:1），全站 token 与暗色那条 `!important`
  规则不动；浅色按用户判断保持原值（§2.2 本就记着 react 浅色 token 故意不含 `colorTextPlaceholder`）。
  四个字段加 MFA 输入框同时补上 `aria-label`，让字段名不再只靠颜色传达。
- 暗色折叠后弹出的子菜单浮层不再露出厂藏青（react）：那一层是浮层，按 §2.3 该吃 `#1C2128`，实测却是
  antd 出厂 `darkPopupBg` = `#001529`——比本项目的暗色蓝黑系更暗、更偏蓝，和抽屉/下拉摆在一起能一眼分出
  "两套黑"（不是可读性问题：`#F8FAFC` 在其上 17.61:1）。现给 `darkTheme.ts` 的 `Menu` 段补
  `darkPopupBg: '#1C2128'`；`darkItemBg` 保持出厂未动，它只喂根 UL，而那层早已被 `SiderMenu` 的
  inline `background: 'transparent'` 盖住。弹层的圆角 12 与黑 α 投影本就合规，逐值未改。

## [1.4.0] - 2026-10-15

### 新增

- AI Agent 编排（全栈，三端齐备）：Agent 应用 CRUD + 图执行引擎（五种节点：开始/
  知识检索/LLM/条件分支/结束；模板引用上游输出、条件表达式走 expr 语法）+ 可视化画布
  编辑器（react @xyflow/react 先行，vue-element / vue-vben 移植 vue-flow 版）+ 试运行
  SSE 流式执行（ai_agent_run 事件逐节点实时推送，运行轨迹先呈现、同步 trace 后覆盖）+
  运行台账（落库 + 行内抽屉浏览）。模型/知识库/配额复用 AI 模块既有链路；已部署实例：
  make openapi + 重启 + 「接口同步」（新端点 `/admin/v1/ai/agents*`）与「菜单同步」后可见。
- 通知渠道全量扩展（go-utils/notify 39 渠道接入，react 先行）：企业微信应用/钉钉工作通知/
  飞书应用/公众号模板消息/Telegram/Slack/Discord/Teams/Google Chat/Webex/Line/Matrix/
  Mattermost/RocketChat/Zulip、Bark/ntfy/Gotify/Server酱/WxPusher/Pushover/PushBullet/
  PushDeer、极光/个推、华为云/云片/Twilio/Vonage/WhatsApp 短信、Mailgun/SendGrid/Postmark、
  IFTTT/Join/Home Assistant、Datadog/Opsgenie/PagerDuty。新渠道配置整包存 settings 列
  （敏感字段加密落库、回显脱敏、留空不修改），配置表单按后端 schema 动态渲染（新增渠道
  只需一份字段声明）。台账/路由规则页新增「扩展渠道（EXT）」选项——告警与通知可路由到
  任一扩展渠道；阿里云/腾讯云短信拆为两个一级选项并入「短信 / 消息」分组。已部署实例：
  make openapi + 重启 + 「接口同步」后新端点生效。
- 短信通知渠道（阿里云 / 腾讯云，全栈三端）：通知渠道页可选「短信」类型，配置服务商访问密钥
  （加密存储、不回显）、短信签名与验证码模板；「测试发送」按渠道类型分派（邮件走测试邮件、
  短信发一条真验证码）。短信正文由服务商侧审核模板渲染，系统只填验证码——没有自由文本通道，
  因此路由规则页不提供 SMS 选项（无验证码的投递会明确失败）。已部署实例升级：重启前先
  `make openapi`（新端点 `POST /notification-channels/{id}/send-test` 需在管理页
  「接口同步」重建 Api 表后才可达）；台账 `event_type` 枚举加 `CHANNEL_TEST_SMS` /
  `PHONE_BIND_CODE` 两值（迁移自动加列）。
- 忘记密码支持手机号：输入绑定的手机号即走短信验证码（需已配置启用的短信渠道），邮箱路径不变；
  绑定联系方式接口同步启用手机号分支（验证码经短信下发，通过后写入手机号登录凭证）。
- 通知投递台账新增服务商侧消息 ID 列（Receipt.ProviderID），跨系统对账有据可查。

### 变更

- 通知发送层下沉 go-utils/notify 子模块（本仓 channel 包收薄为适配层）；SMTP 渠道
  新增发件人显示名 smtpFromName。
- 三端新建/编辑表单统一由弹窗改抽屉（AI Agent、路由规则、通知模板、监控告警、渠道、
  改密、MFA 绑定、知识库文档管理等十余处），长表单不再挤在弹窗里。
- vue-vben 端去微服务单包化：26 个 workspace 框架包并入 framework/，workspace 归零、
  全工程唯一 node_modules——采用者克隆即用，不再依赖 workspace 拓扑。

### 修复

- 设计语言对齐扫尾（三端）：登录/品牌面板与插画硬编码色接回 token、图表与画布元素
  显式取 token 不吃 CSS 层叠、固定列 hover 漏字、空表格占位行禁 hover 高亮、
  vben vxe 暗色表头/行 hover 配色对齐设计语言、抽屉暗色底色统一 L1 表面 #111827。
- vue-vben 画布/抽屉系列修复：分组下拉注册名、Drawer/Empty 未注册整块不渲染、
  抽屉表单 Switch 被 w-full 撑满整行、AI 页新建按钮 icon+文本竖排、画布抽屉盖不住
  布局侧栏 + fitView 竞态、dev API 改走 vite 代理同源转发（直连被 CORS 拦）、
  pnpm dev 起不来的 rimraf 死引用。

## [1.3.0] - 2026-10-08

### 新增

- 概览组新增「工作台」页（概览 → 分析页 + 工作台，二级菜单定型）：三端同构聚合页，
  含欢迎横幅、常用页面快捷入口、我的站内信摘要（未读数）、我的活跃会话；数据全部
  复用既有接口，无新增后端资源。种子菜单新增 Workspace 行（归类 DASHBOARD 模块，
  平台/租户管理员默认可见）；已部署实例按既有口径升级：任一端「菜单同步」（MERGE）
  补菜单行后，在「权限管理」里把它勾给相应角色即可见。

### 修复

- 动态后端路由（`app.accessMode: backend`）react / vue-element 两端修复并全量实测：
  react 修 vue-router 风格菜单树导致的创建卡死（嵌套绝对路径归位为目录组内相对路径、
  目录组缺省重定向、补 `/` 根落地路由）、后端组件路径归一（剥 `/app` 段与扩展名、
  下划线→横线）与菜单/面包屑/标签页标题 i18n（种子键位补进 routes 文案表）；
  vue-element 修 `BasicLayout` 误映射到空壳布局（后端模式整页空白）、组件路径
  `/app` 前缀对齐与横线/下划线混存兜底、种子键标题翻译。实测通过后两端默认值
  均保持 `frontend`（前端路由），三端口径不变。

## [1.2.0] - 2026-10-04

### 新增

- 企业 OIDC SSO 登录（全栈，三端齐备）：授权码流 + Redis state 单次有效，环境变量
  opt-in 接入；真实 Keycloak 实测修两处（SsoLogin 补隐私层绕过、user_repo nil mask
  守卫）；Keycloak 演练环境（deploy/sso）与 CI 冒烟 e2e 防回归。
- 通知域 P3 收官（两片）：用户通知偏好（静音时段 + 分类退订）与通知模板管理/渲染
  全栈落地，事务性邮件按约定 code 可被模板覆写（无模板回落 mailtext）；偏好 tab 与
  模板管理页三端齐备。
- 监控告警联动：指标阈值规则 + 周期评估 + 经通知域分发；告警/恢复通知文案按收件人
  偏好语言渲染；告警规则页三端齐备。
- 审计日志服务端导出：XLSX / CSV 全量流式下载（突破客户端一万行上限），覆盖五类
  审计日志、AI 用量流水与通知投递台账，三端入口统一。
- 套餐配额硬执行与水位告警：USER / STORAGE / API_CALL 等四类配额接入租户闸门与
  业务入口，配额水位提前告警经通知域推送；租户自助配额用量面板（个人中心）三端齐备。
- 用户偏好语言：locale 字段 + 事务性邮件、告警通知、AI 审计日报均按收件人偏好语言渲染。
- 租户白标：登录页按租户编号渲染 Logo / 名称（免鉴权公开查询），三端齐备。
- 系统级常驻任务可视化：asynq Inspector 只读视图（方案 C'，零新表），三端任务监控页齐备。
- 备份恢复工具链：RestoreCoreTables 空库恢复、备份桶超期对象生命周期清理、恢复演练文档。
- AI 工具调用：Function Call 协议层（多轮工具调用循环接进流式对话）+ `ai_chat_tool`
  SSE 帧工具调用可见化，三端对话 UI 齐备；语义搜索按调用者菜单权限裁剪，堵住越权检索缺口。
- 字段级权限铺开第二资源：Role（permissions 权限集）。
- 登录策略 REGION 维度真实现，MAC 维度移出三端下拉（对齐文档 7.3）。
- react 补齐 /dashboard/analytics 分析页；全局搜索面板三端统一（vben 形态为基准）。

### 变更

- 通用代码三批下沉 tx7do 家族库（架构理顺，业务行为不变）：netutil / doctext / mailer /
  fieldperm / eventbus / sqlutil 与 crypto / password / converter / ossutil / auditutil /
  authorizer / sliceutil 下沉 go-utils；脚本宿主的钩子注册表、语言无关模块框架 + HTTP
  出站护栏（`SCRIPT_HTTP_ALLOWED_DOMAINS`）、Lua 宿主桥下沉 go-scripts（本仓保留编排器
  与业务桥接）；查看者标准上下文（UserContext / SystemContext）下沉 go-crud/viewer
  v0.0.8（本仓 pkg/entgo/viewer 缩为数据范围映射适配层）。
- 三端 dev 端口迁移 5xxx → 15xxx（react 15888 / vue-element 15777 / vue-vben 15666），
  避开姐妹仓默认段。
- 移除 oauth.proto 第三方绑定半成品与 LoginResponse.RefreshToken 残留字段（零消费方
  死代码清理）。
- 三端共享前端代码收敛为镜像并接 CI 防漂移：pagination 查询序列化、认证拦截器、
  updateMask 构建泛化（六处手写掩码退役）+ 白名单钉校验器与封闭扫描。

### 修复

- 租户侧边栏整箱清空（TenantRepo 读侧补 WithPlan 边回填）。
- 全局搜索三连修：react 补本地菜单匹配 / 结果点击导航 / Ctrl+K / 键盘导航；ele 菜单
  索引存裸 i18n key 导致中文永远搜不到，改存翻译文案并修键盘选中语义。
- vue-element 七处抽屉编辑不回填 / 编辑变新增，及通知模板表单项蒸发的根修。
- 三端登录框 Chrome 自动填充白底覆盖（-webkit-autofill 处理 + autocomplete 语义）。
- AI 问数 few-shot 注入按值 range 拷贝 proto 消息，修结构体复用引发的锁污染。

## [1.1.0] - 2026-09-29

### 新增

- AI 模块（全栈，三端齐备）：模型提供商管理（密钥加密存储 / 连通性测试）、
  SSE 流式对话（多会话 / 历史持久化 / 会话重命名）、用量配额记账与用量统计页
  （含当月汇总 RPC），并登记进业务模块枚举与套餐白名单映射链（租户按套餐开通）。
  详见 `docs/ai_module.md` 与教程第 10 章。
- 知识库 RAG：文档上传（txt / md / docx / pdf 抽取文本）→ 切片向量化（pgvector）
  入库 → 对话按知识库检索注入上下文；三端管理页与聊天知识库选择器齐备；
  向量重索引任务（ai_doc_reindex）、切片批量入库与 embedding 全链计量、
  切片表越万行自动建 HNSW 索引。
- 智能问数：自然语言 → 只读 SQL → 结构化结果与图表。四重护栏
  （仅 SELECT / 表白名单 / 强制租户谓词 / 行数上限）+ 只读事务，面向租户开放
  自助分析（双白名单）；多轮追问、结果自动图表（图表 / 表格切换）、
  订阅定时问数（ai_query_run：cron 执行 NL→SQL 后经站内信推送）；三端问数页齐备。
- AI 内容生成（表单助手）：场景预设提示词 + AiGenerateButton 通用组件，三端覆盖。
- 全局语义搜索：菜单向量化索引（pgvector），顶栏全局搜索接语义检索 API。
- AI 安全与异常洞察：分析页审计明细行为模式挖掘卡片、Dashboard 当日指标
  AI 图文解读、审计日报 AI 摘要任务（ai_audit_digest：定时聚合 + LLM 摘要 +
  站内信投递）；告警后端只回结构化事实，文案归前端 i18n。
- 脚本系统暴露原生 `ai` 模块：脚本可直调 LLM 对话。
- `docs/sys_config.md`：参数管理（sys_config）参考文档——键值模型、缓存读取器
  （负缓存 / 类型回退）、写路径与多实例 Redis 失效广播、内置参数按键补种、新参数接入
  步骤；接入 docs/README.md 参考层索引与教程 07 深读。

### 变更

- 列表加载态三态化（三端统一）：首屏表格形状骨架（复用列定义、与真实表同高）+
  250ms 阈值内不闪加载态 + 失败态与重试；列表页请求不再自行 catch，错误统一由
  表格记录日志并内联展示原始原因。设计语言规范 `docs/design-language.md` §4
  补「列表加载态」条。
- README（三语）：顶部新增「从这里开始」导览——链接文档索引（含按角色推荐阅读路径）与渐进教程，
  「快速开始」上移至技术栈与安全合规之前，压缩新人首屏到可行动路径的距离。
- `docs/README.md`：通知域条目不再于索引内转载分阶段实施状态史，
  各阶段落地状态以 `notification_domain_design.md` 文内状态标记与验收清单为准
  （对齐该文档「实施进度更新时改本文状态标记、不另开文档」的单一事实源约定）。
- 全仓文档审计修订：`backend_file_upload.md` 上传请求体改记 JSON+base64 现状
  （2026-09-17 三端切换，multipart 保留为兼容回退）；`crud_module_guide` 步数 10→11、
  生成的注册函数名更正为 `Register<Entity>ServiceHTTPServer`、代码生成命令统一 gow-first、
  List 过滤补 contains/ID 排除约定并链 `list_query_rule.md`；`adopt-one-frontend.md`
  workflow 表述跟进 docs-parity CI；`backend_project_struct.md` 补全 pkg 20 个子包清单；
  三语 README 环境脚本路径加 `backend/` 前缀，功能列表补在线用户 / 我的活跃会话 /
  通知规则 / 通知投递台账四行（三语同步，parity CI 通过）。
- 移除失效的 `make install-golang` 目标（原指向的 `install_golang.sh` 已在脚本模块化
  重构中并入 `install_unix_dev/prod`，由 `lib/go-utils.sh` 的 `install_golang()` 承担），
  `backend/scripts/README.md` 对应段落改记现状；修正 `config_repo.go` 两处滞后于
  多实例失效广播的注释。

### 修复

- 修复历史消息 role 大小写映射：部分模型厂商对大写 role 返回 422，统一映射小写；
  连带修模型提供商 Update 的 key 分支逻辑反转。
- 修复 Dashboard AI 洞察接口 500：审计行 user_id / username 为 NULL 时 Scan 报错。
- 修复智能问数实测暴露的缺陷：SQL 提取与拒答对抗（真实 DeepSeek 实测）、
  vue-element 端全局 loading 卡死 / 空图表容器 / 图表重挂载、React 端输入区
  常驻吸底；用量流水补 duration_ms，用量摘要与流水列表同口径。
- 修复 vue-element 启动期 i18n 竞态：语言包未就绪即渲染致整页文案裸 key。
- 修复 vue-element 暗色模式与图表：Tailwind v4 `dark:` 变体改挂 `html.dark`
  跟随应用主题开关；暗色 hover 亮度装饰收敛回库原生通道；ECharts 6 兼容
  （containLabel 废弃迁移、已销毁实例守卫、零尺寸初始化防御）。
- 三端样式整改：语义色文字档成层、语义色按钮禁用态去实底（vue-element）。

## [1.0.0] - 2026-09-24

### 新增
- 脚本系统（脚本级插件系统，Lua / JavaScript）：脚本以数据库为唯一事实源、
  管理页增改即时生效；五类扩展点——实体生命周期钩子（before 同步可否决 /
  after 异步旁路，覆盖 user / tenant / role / internal_message / notification_channel）、
  定时任务（asynq 固定分发订阅 + 处理器代际清理）、事件订阅与发布、
  HTTP 出站（域名白名单 fail-closed + 环回硬禁 + 重定向逐跳复检 + 体积/超时上限，
  `SCRIPT_HTTP_ALLOWED_DOMAINS` 配置）、试运行（一次性隔离引擎）；
  执行审计落 `sys_script_logs`（脚本/触发方式/版本/成败/耗时/错误），
  管理页可查可清理；跨实例变更经 Redis pub/sub 自动重同步；
  三端管理页齐备（Monaco 编辑器、钩子点自动补全、试运行对话框、执行日志抽屉）。
  详见 `docs/script_system.md`。
- 设计语言规范：`docs/design-language.md` 作为三端视觉唯一权威值表
  （颜色/圆角/布局尺寸），React 端主题偏好值已同步对齐。
- 在线用户（会话管理）：登录 / 刷新轮换 / MFA 验证通过时在 Redis 记录会话元数据
  （用户、租户、客户端类型、登录 IP、User-Agent、设备 ID、登录时间），
  新增在线会话列表接口（关键词过滤 + 分页，登录时间倒序）与强制下线接口；
  React 端新增「系统管理 → 在线用户」页面（30 秒自动刷新，支持单会话强制下线）。
- 安全加固：修改密码与管理员重置密码成功后，立即吊销目标用户全部客户端类型的
  令牌（含当前会话），强制重新登录；前端改密成功回调同步登出并引导重新登录。
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
  口令策略三件套（复杂度 / 历史复用 / 有效期，阈值经「参数管理」平台参数调整，
  内置键启动时播种；环境变量口径已废弃，见 `pkg/password/policy.go` 包头注释）。
- 站内信广播投递改走 Asynq 异步任务队列（断点恢复 + 幂等投递）；
  消息管理支持发送范围选择（全员 / 指定用户）与消息撤销。
- 通知渠道管理（渠道类型二选一：`EMAIL` 走 SMTP / `WEBHOOK` HTTP 回调）：渠道增删改查、SMTP 密码加密存储与
  列表脱敏展示（编辑留空不改密码）、启用 / 停用与测试发送；Webhook 侧五种签名风格
  （NONE / DINGTALK / FEISHU / WECOM / CUSTOM）与载荷模板；页面三端齐备（React / vue-element / vue-vben）。
- 通知域事件缝与路由：`sys_notification_rules` 决定"业务事件 → 渠道 / 是否异步"（初始四行随播种），
  每次投递落 `sys_notification_deliveries` 一行台账（SENDING / SENT / FAILED / SKIPPED），
  投递台账页三端齐备。`SMS` 在枚举里预留，注册表无实现 ⇒ 路由到它按 FAILED 记账。
- 服务监控页：只读展示服务运行时指标（CPU 核数、内存、goroutine 数、运行时长等），
  自动刷新；页面三端齐备。
- 忘记密码全流程（SMTP 二期）：向绑定邮箱发送 6 位验证码（10 分钟单次有效，
  用户不存在时静默成功防枚举），凭码重置密码并立即吊销全部会话；
  登录页新增「忘记密码」入口（React 忘记密码页两步流程：邮箱发码 → 验证码 + 新密码重置）。
  邮箱绑定 / 换绑接入真实验证流程（向新邮箱发码，凭码写入 EMAIL 登录凭证）。
- 六类审计日志页「导出全部」：按当前筛选条件分页聚合导出 CSV
  （RFC4180 转义 + BOM，1000 行/页聚合，上限 1 万行），三端齐备。
- 富文本编辑器图片上传：工具栏上传 / 粘贴截图 / 拖拽图片统一走内置文件上传
  （服务端中转 + 元数据落库），不再插入 base64；上传成功返回 HMAC-SHA256
  签名公开访问 URL（1 年有效期），新增免鉴权图片代理端点
  （恒定时间验签 + 过期校验 + MinIO 流式回源）。
- 角色级数据权限范围（V1）：若依式五档数据范围（全部 / 仅本单元 / 本单元及子级 /
  自定义单元集 / 仅本人），多角色登录期聚合并集写入令牌，查询侧按范围注入行级过滤
  （V1 试点：岗位表）；三端角色管理接入范围配置（列表标签列 + 抽屉五档与自定义组织
  树）。含组织单元物化路径修复（根节点 /ID/ 包含自身、改父后重算子树路径，旧格式
  全表根节点共享同一前缀会使子树前缀展开误伤全表）。详见
  `docs/data_scope_design.md`。
- 角色级字段权限（V1，试点：用户表）：角色配置黑名单字段集，登录 / 刷新时聚合并集
  写入令牌 hfs claim；服务端响应包装器读路径按黑名单清值（字段自响应中消失）、写
  路径清值并剔除 field_mask 防越权显式置零；perm-codes 响应同时下发
  hiddenFields，前端据此裁剪列表列 / 搜索项 / 详情与编辑表单。三端角色管理接入
  字段权限勾选区。
- AK / SK 机器凭证：租户级 AccessKey / SecretKey 管理，三端页面支持创建（Secret
  一次性展示）、启停、删除与密钥轮换重置（新 Secret 一次性展示、旧 Secret 立即
  失效）；令牌交换端点凭 AK / Secret 签发租户作用域机器 JWT（machine 角色、纯
  access 无 refresh），交换端点按 IP + AK 复用登录限流器（失败计数 / 锁定 / 成功
  清零）。
- 通用参数管理（sys_config）：动态 KV 平台参数（区别于业务字典的枚举配置）+ 服务侧
  缓存 accessor（per-key 懒加载 / 负缓存 / 写路径同步失效）+ 三端管理页；内置参数
  启动时播种、禁删可改；多实例部署下参数变更经 Redis 发布订阅广播，各实例订阅并
  清空本地缓存。
- Excel 导入导出（三端）：列表页「导出」全量铺面（TableExportButton /
  exportsAction 封装：按当前筛选条件分页聚合拉取、格式下拉 CSV / XLSX、上限
  1 万行），业务列表页与审计日志页均接入；岗位管理新增 Excel 导入（客户端模板
  下载、逐行走既有创建接口、行级错误回报，所属组织列按组织名称精确匹配回填组织
  单元，未命中报行级错误）。
- 菜单同步三端补齐：同步整体事务化（中途失败整体回滚，不再留半空表），新增增量
  合并（MERGE）模式——已存在菜单按全路径匹配原位更新（保留 ID / 角色授权 / 手工
  停用状态），缺失才新增；三端菜单管理页补齐「同步菜单」入口与合并语义确认文案。

### 变更
- 登录页与输入控件视觉对齐设计语言规范（认证卡实底、输入框与表面同层、
  占位符提亮，暗色登录左栏修正）；Vben 暗色主题对齐规范近黑蓝系。
- go-crud 依赖升级：entgo v0.0.55、gorm v0.0.24、pagination v0.0.16、viewer v0.0.7，
  并同步新增租户守卫 Create 防伪造 / DeleteOne 越权单测。
- 刷新令牌（refresh token）由本地存储迁移至 HttpOnly Cookie，并改用自描述 JWT。
- CORS 补齐 `allow_credentials`，以适配 HttpOnly Cookie 刷新令牌的跨域携带。
- 安全加固：内容安全策略（CSP）、输入校验、日志注入防护、枚举零值守卫。
- 后端日志统一迁移至 `kratos-bootstrap` logger，移除 refresh cookie 明文 HTTP 落地。
- 操作 / 权限审计剔除 GET 等读请求与登录 / 刷新等会话维护端点，聚焦真实写操作，
  避免页面浏览产生噪音日志。
- 口令策略阈值（复杂度 / 历史复用 / 有效期）由环境变量迁移至「参数管理」平台
  参数：内置键启动时播种、服务侧经 accessor 读取，环境变量配置废弃。
- vue-element / Vben 登录页视觉对齐 React 端与设计语言规范（表单卡片化、标题与
  描述移出卡片、右上角工具栏同构）。

### 移除
- 移除自主注册流程：注册 RPC / 路由 / 鉴权白名单与注册独占的解密传输处理链，
  三端注册页面与登录页入口、OpenAPI 注册端点契约全删——管理底座账号一律由
  管理员创建。已部署实例升级后需在管理页「接口同步」重建 Api 表（sys_apis
  的 register 行随之消失）。

### 修复
- 修复 MFA 登录成功路径 refresh token 走响应体：与主登录路径不一致（HttpOnly Cookie），
  造成 refresh 暴露在 JS 可读响应体、且 MFA 用户会话的静默续期从未生效（前端仅认 Cookie）。
  改为统一走 HttpOnly Cookie 下发。详见 `docs/authentication.md`。
- 修复 `sys_access_keys`（AK/SK 凭证表）为 33 张带租户表中唯一缺仓内租户写隔离守卫
  `TenantMutationGuardPolicy` 的表：补挂守卫（库层 TenantPrivacy 为主、仓内守卫为冗余
  第二道防线的全量双防线），并新增守卫单测钉住跨租户 Update/Delete 0 行命中。
  详见 `docs/tenant_isolation.md`。
- 修复空库全新部署后 admin 无法登录（GitHub issue #58）：默认密码 `admin`
  不满足等保口令复杂度策略，初始化种入默认凭证时被静默拒绝——用户行已建、
  凭证行缺失且错误被吞，重启后初始化不再触发，admin 永久登录失败。
  默认密码统一改为 `Abcd@1234`（三语 README 与启动文档已同步）；
  启动时自愈：检测到凭证表为空（半初始化特征）自动补种默认凭证；
  全新初始化的凭证改绑实际生成的用户 ID，不再假定其为 1。
- 修复字典类型 / 字典项创建 500（ent O2M required edge 误写），字典功能全链路恢复。
- 修复改密 / 重置密码漏设解密标志，AES 密文被当明文哈希入库导致改完即锁号。
- 修复 Vben 审计日志页表格渲染崩溃（geoLocation 插槽裸读污染 vnode 树）
  与刷新后会话静默恢复时序问题（恢复完成前被重定向到登录页）。
- 修复租户隔离 Update / Delete 缺口：TenantPrivacy 的变更评估对非 Create 操作
  直接放行，知道记录 ID 即可跨租户篡改 / 删除；新增 schema 级
  TenantMutationGuardPolicy，对非 Create 变更注入租户谓词（跨租户变更命中 0 行），
  全部 30 个含租户字段的 schema 生效；DeleteOne 因 ent 生成代码无 WhereP，
  依赖规则外的显式校验（记录为已知限制）。
- 修复 vue-element 页面切换白屏：transition `out-in` 与 vue-router5 懒加载竞态，
  懒 chunk 解析完成时旧页已被移除而新页永不挂载，router-view 整体僵死且零报错
  （连续导航 5~7 次必现）；改为同帧交叉过渡并同步渲染 wrapper，
  连续约 100 次菜单导航零复现。
- 修复凭验证码重置密码必失败（ResetCredential 漏查凭证关联字段致 nil panic）；
  管理员在 UI 重置密码走同一路径同样受影响，一并恢复。
- 修复 React typecheck 门禁空跑：根 tsconfig 无实际检查文件导致 `tsc` 恒定 0 错误；
  门禁真实化后清零 40 项历史类型欠账，含租户用量恒显示 0 的
  数字 / 字符串枚举比较错误与深链初始化 ReferenceError。
- 修复 vue-element ProSearch 超宽表单下字段挤压在左侧、操作按钮孤立折行
  （auto-fill 生成大量窄列，改 auto-fit 均分拉伸铺满）。
- 修复 SMTP mailer NONE 加密模式被默认配置拒绝（现支持明文 + STARTTLS 自动升级）。
- 站内信回执状态上报补注册 HTTP 路由（服务实现早已存在但从未注册路由，
  SENT→RECEIVED 回执补偿通道此前不可达）。
- 修复角色 Update 静默清空关联：go-crud UpdateOne 的 FilterByFieldMask 会清掉
  未进 mask 的关联字段，角色编辑保存会丢失权限 / 组织单元关联；改为先快照关联
  载荷再按 mask 过滤更新。
- 修复权限组物化路径：根节点全表共享 “/” 前缀（违反 /1/10/101/ 包含自身的格式
  约定）、Update 改父不重算路径、同步流 BatchCreate 完全不写路径、
  UpdateParentIDs 批量改父后路径全部失效；统一按 parent 链 BFS 重算子树路径
  （含成环检测与事务内可见性处理），可自愈历史脏数据。
- 修复树删除语义：组织单元删除在子树内仍有岗位时 400 拒绝（先清 / 转移岗位），
  用户与组织单元间的绑定行随子树级联清理（纯绑定关系、用户本身不受影响）且
  删除改事务执行；权限组删除在存在子组或组内权限点时 400 拒绝（此前数据库
  OnDelete SetNull 会把子组静默提升为根、组内权限点静默丢失归属）。
- 修复修改密码静默失效（GitHub issue #68）：三端用户 Update 把密码留在 data
  包裹内且未做顶层 AES 加密，后端仅读顶层密文、解密失败被吞，接口返回 200
  但密码实际未变；三端用户更新 / 管理员重置 / 个人中心改密链路统一拆出顶层
  密文并补加密。
- 修复接口同步从未写入任何行（syncWithOpenAPI 的 Update 请求漏带自身 id，被
  仓库 id==0 守卫在 AllowMissing 分支之前整批弹回）与套餐模块守卫倒置
  （plan_module 的 PlanId 判空条件写反，套餐与模块关联自始无法经 API 建立）。
- 修复 React / vue-element 生产构建 CSP 误拦自有主题预绘内联脚本（部署层响应
  头注入的 script-src 与页面 meta 取交集，拦掉 index.html 内联主题脚本，每页
  CSP 违规且首绘防闪烁失效）：改为构建期对内联脚本逐个计算 sha256 注入 meta
  白名单（内容一变哈希即失效，注入脚本无法借道），部署层响应头删去
  script-src、页面 meta 为唯一权威（GitHub issue #63）。
- 三端批量回归修复（84 页全量扫测）：分栏主表窄视口响应式断裂（三端，含
  Element Plus 2.14 Splitter 需百分比尺寸的兼容处理）；Vben 三页因 antd 组件
  注册缺漏整页破版与通用操作键 i18n 缺失；React 登录验证码表单值被包裹 div
  隔断、平台侧建用户对话框角色数据源错位；React 文件列表大小列因 protojson
  将 64 位整序列化为字符串而崩溃、头像空串 src；React 登录审计详情抽屉标题
  误用缺失 i18n 键显示原始 key；三端查询序列化器统一补操作符后缀守卫
  （__not / __gte 等后缀键禁叠 __contains，防后端语义反转 500）。
- 修复登录成功后最后登录时间 / IP 不记录（密码与 MFA 两条登录路径；用户列表
   与个人中心的「最后登录」字段此前恒为空）。

## [0.x] — 早期开发版本

1.0.0 之前 GoWind Admin 处于 `0.x` 阶段，没有正式发布版本。
下列条目为该阶段对使用者有影响的变更摘要（以 Git 提交记录为准），仅作历史留存。

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

> 本段为 1.0.0 之前的历史摘要；自 [1.0.0] 起按 Keep a Changelog 规范逐版本记录。
