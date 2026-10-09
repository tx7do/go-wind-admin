# 设计语言规范（Design Language）

> **定位**：react / vue-element / vue-vben 三端共享的视觉语言权威值表。
> **唯一权威**：任何颜色、圆角、尺寸、动效的调整，先改本文档，再同步三端代码；禁止只改代码不改文档。
> **参照实现**：vben 端的主题体系（preferences → CSS 变量 → 组件库 token 派生）是视觉基准与架构参照；react / vue-element 按第 3 章映射表落地。
> **三选一口径**：三端是"三选一"（见 [adopt-one-frontend.md](./adopt-one-frontend.md)）——采用者只在自己选定的那一端设值、跑那一端的门禁；
> "改本文档 + 同步三端"是**上游维护者**的义务，不是使用者的负担。
> **现状**：第 3 章原为"现值 → 目标"迁移清单，2026-09-25 逐端复核后确认**规范值已落到三端代码里**，
> 故该章改写为**落点索引**（要改哪个值去哪个文件），退役旧值只留在各节末的「沿革」里。

---

## 1. 基本决策与理由

| 决策 | 取值来源 | 理由 |
|---|---|---|
| 主题模型 | vben | HSL token 三元组 + CSS 变量下发 + 组件库 token 派生，三端 preferences schema 已同构 |
| 主色 / 语义色 | vben 默认值 | `hsl(212 100% 45%)` 深蓝更"企业级"，且 vben/语义色三端 config 本就同源 |
| 圆角体系 | vben | `--radius: 0.5rem`（控件 8px），卡片/浮层 12px |
| **暗色中性色** | **react 六轮定稿（2026-08-24）** | 近黑蓝 `#0B0F19` 系。**三端均已落在该系**：react `core/preferences/config/darkTheme.ts:37,39`、vue-element `styles/_dark-mode.scss:33-36,54,72`、vben `framework/base/design/design-tokens/dark.css:4-11`（该文件注释即写明对齐本规范 2.3）。vben 原"浅炭暗色"降为沿革，见 2.3 末 |
| 布局尺寸 | vben（react / ele 已同步，落点见 2.6） | 侧栏 224 / 折叠 48 / 顶栏 50 / 页签 38 |

**谁向谁对齐**：vben 提供视觉语言与主题系统；react / vue-element 提供工程实现。不移植 vben 的代码，只移植它的"语言"。

---

## 2. Token 权威值表

约定：HSL 写法为 `hsl(H S% L%)`。vben / vue-element 的 CSS 变量存 `H S% L%` 三元组（不带 `hsl()` 包裹，使用时 `hsl(var(--primary))`）；react 端 antd token 直接填色值字符串。≈HEX 仅供 quick check，以 HSL 为准。

### 2.1 品牌色与语义色（三端统一）

| Token | HSL（权威） | ≈HEX | 用途 |
|---|---|---|---|
| `--primary` | `hsl(212 100% 45%)` | `#006BE6` | 主色；react 端 colorInfo 同值（沿用 vben 映射） |
| `--success` | `hsl(144 57% 58%)` | `#57D188` | 成功（2026-09-13 勘误：原 `#57D1A0` 换算有误，正确值即 ele 端运行时实际值） |
| `--warning` | `hsl(42 84% 61%)` | `#EFBD48` | 警告（2026-09-13 勘误：原 `#EF7A48` 有误） |
| `--destructive` | `hsl(348 100% 61%)` | `#FF3860` | 危险/错误（antd colorError、EP danger/error） |
| `--primary-foreground` | `hsl(0 0% 98%)` | `#FAFAFA` | 主色上的文字 |

**语义色实底上的前景**：`--primary`（相对亮度 0.16）用 `--primary-foreground` 即可（白字 4.95:1）。
但 `--success`（0.49）/ `--warning`（0.55）配近白只有 1.87:1 / 1.74:1（按 2.1 HEX 换算）。实测到的两处真实缺陷：
vben 任务页"启动全部任务"= `#FAFAFA` on `#57D188` **1.85:1**；react 同按钮旧值 `#fff` on `#52c41a` **2.27:1**，
且 react 端还被 `pro-components-dark.css` 的暗色兜底 `!important` 整条覆盖成中性灰（暗色下绿色从未渲染，实测 bg `#1c2128`）。
规定：**实底 success/warning 按钮的前景取 2.3 的暗色 L0 `#0B0F19`（亮/暗两态同值，在 `#57D188` 上 9.98:1）**，
不得沿用 `--*-foreground` 的近白值。三端现状：ele 由 EP 的 soft 变体（`#57D188` 文字 on `#1B2720`，8.02:1 暗色实测）
已合规，react/vben 已按本条改深色墨；`--destructive` `#FF3860` 配白字 3.51:1（EP 徽标暗色实测），12px 仍不足 4.5，
属库内派生值、暂未统一。

**语义色当文字用（Tag 文字态、link/text/plain 按钮、页签激活态、查询区折叠按钮、链接）不许直接用基准色**：
基准色在白底上是 destructive **3.51** / success **1.93** / warning **1.74**（primary 4.95 也只是勉强过），
统一改走"向中性墨（浅色）/ 向白（暗色）混色"的文字档，写成 `color-mix(in srgb, <语义色> P%, <中性基准>)`
——用户换主题预设（主色变了）时档位自动跟随，不需要再手算一遍。

| 档 | 浅色（向 `#0B0F19` 混） | 暗色（向 `#FFFFFF` 混） |
|---|---|---|
| primary | 90% → `#0162D2`（白 5.71 / 画布 `#F1F3F6` 5.13 / 主色 light-9 淡底 `#E6F0FD` 4.96） | 70% → `#4D97EE`（L1 5.89） |
| success | 50% → `#317051`（5.89） | 90% → `#68D694`（9.83） |
| warning | 50% → `#7D6631`（5.50） | 90% → `#F1C45A`（10.80） |
| destructive | 70% → `#B62C4B`（6.09） | 90% → `#FF4C70`（5.51） |

比 2.1 基准色"更暗/更亮"是故意的：文字档只服务文字，实底按钮与徽标的前景看上一条。
变量名三端统一 `--gowind-{primary,success,warning,danger}-text` + `--gowind-solid-ink`，落点见 §3。

**两条踩过坑的结论，改样式前必读**：

1. **浅色的"深墨前景"规则不能整条搬进暗色**。antd/EP 在暗色下自己派生了一套语义色（暗色 error `#DC3355`、
   选中菜单实底 `#035EC7`），跟 2.1 的基准值不是一个颜色：白字 on `#DC3355` = **4.52** 而 `#0B0F19` 墨字只有
   **4.24**；暗色选中菜单实底上白字 **6.12**、primary 文字档 **3.13**（比基准色还低）。
   所以"实底配深墨""选中菜单文字降档"这类规则一律限定 `html[data-theme='light']` / `html:not(.dark)`，
   暗色另走 70/90/90/90 档。这一条曾经制造过 77 处暗色新违规。
2. **三端覆盖层都得写 `!important`，但理由各不相同**：react 的 antd v6 cssinjs 注入在打包样式**之后**，
   且 Tag 规则是 `:where(hash).ant-tag.ant-tag-success:not(.ant-tag-disabled).ant-tag-filled` = **0-4-0**，
   三 class 的覆盖必输；vue-element 由 unplugin-element-plus 把组件 CSS 追加到全局 SCSS 之后；
   vue-vben 的 ant-design-vue v4 虽是 `hashPriority:'low'`（0-1-0，权重能赢），但 dev 与 prod 的注入顺序不同。
   结论：**按权重推不如实测**——写覆盖前先用探针读一次生效值，别按选择器复杂度赌。

**色阶派生规则**：由基准色程序化生成 -50 ～ -900 阶梯（vben / vue-element 已有 `generatorColorVariables`；react 端由 antd `defaultAlgorithm/darkAlgorithm` 自动派生 hover/active）。交互态规则：**hover 取亮一阶、active 取暗一阶，禁止手工挑色**。EP 侧 `light-3/5/7/8/9`、`dark-2` 一律由脚本生成，不手写。

### 2.2 中性色 · 浅色模式

> **落地状态按端不同**（2026-09-25 复核，2026-09-29 补文字档一层）：vue-element ✓（`styles/vendors/_element-plus.scss:31` 页面画布
> `#F1F3F6`、`:52` 边框 `#E4E4E7`）、vue-vben ✓（`framework/base/design/design-tokens/default.css:10`
> `--background-deep: 216 20.11% 95.47%` = `#F1F3F6`）、**react 半 ✓**——react 端只有
> `core/preferences/config/darkTheme.ts` 一套 token 覆盖，**浅色模式的画布/边框仍走 antd `defaultAlgorithm`
> 的默认派生值**，与本表不严格相等（要严格对齐需新增一套 light tokens，未做，属新工作）；
> 但**文字层次（次要档 + 2.1 的语义色文字档）react 已落**：`src/styles/semantic-text.css`，三端同值。

| Token | 值 | ≈HEX | 说明 |
|---|---|---|---|
| canvas（页面画布） | `hsl(216 20.11% 95.47%)` | `#F1F3F6` | 大底，卡片在其上"浮起" |
| surface（卡片/容器） | `#FFFFFF` | — | 卡片、表格容器、顶栏、侧栏 |
| 边框 | `hsl(240 5.9% 90%)` | `#E4E4E7` | 通用边框（≈ zinc-200） |
| 文字-主 | `#1F2937` | — | 标题、重点（gray-800） |
| 文字-常规 | `#374151` | — | 正文（gray-700） |
| 文字-次要 | `#676E7C`（= `hsl(220 9.25% 44.51%)`） | — | 辅助说明、描述表标签。2026-09-29 由 `#6B7280`(gray-500) 提暗一档：旧值在白底 4.83 达标，但落在**实际容器底色**上就不够——antd 表头 `#F5F5F5` 4.43、vxe/EP 表头 `#F0F2F5` 4.31、画布 `#F1F3F6` 4.35；新值同底 4.70 / 4.57 / 4.61 |
| 文字-占位 | `#9CA3AF` | — | placeholder（gray-400），**不参与 4.5 判定**，见下方豁免边界 |

> **占位符档的豁免边界**（2026-09-29 定稿，三端一致）：`#9CA3AF` on 白 = 2.54，但它承载的是"尚未成为内容"的
> 提示语，不是正文，按 4.5 判会把每一张带筛选区的列表页都报成缺陷；antd 同档更浅（`rgba(0,0,0,.25)`）。
> 判据必须窄：**只有坐在表单控件里**（`.el-select/.el-input/.el-textarea/.el-date-editor/.ant-select/.ant-picker/…`）
> 且带 `is-transparent` 空态标记的 `*placeholder` 节点才算这一档。
> `.avatar-placeholder`（姓名首字，85px）、vxe 的"暂无数据"等**是内容文字，不豁免**，实测仍按 4.5 判。
> 选中后的值走 EP 的 `#606266`（去掉 `is-transparent`），同样要量。

灰阶统一走 Tailwind gray/slate 族（与暗色同一冷调体系），**次要档是唯一的例外**：`#676E7C` 不在 gray/slate
的任一级上（gray-500 `#6B7280` 在淡底色上差 0.1~0.2，slate-500 `#64748B` 偏蓝过冷），按 2.1 的混色思路取冷灰。
浅色投影：卡片静止 `0 1px 2px rgba(0,0,0,.05)`，悬浮态 `0 4px 12px rgba(0,0,0,.08)`，克制不发光。

### 2.3 中性色 · 暗色模式（权威 = react 六轮定稿）

分层模型（明度必须随层级递增，"越向前越亮"）：

| 层级 | Token 语义 | 值 | 用途 |
|---|---|---|---|
| L0 大底 | `bgLayout` | `#0B0F19` | 页面画布、Layout 大背景（最暗） |
| L1 表面 | `bgSurface` | `#111827` | 卡片、抽屉、表格容器、侧栏、顶栏 |
| L1 表面（输入控件） | `bgSurface` | `#111827` | 输入框/下拉框与表面**同层**，仅靠边框区分（2026-09-08 对齐 vben 观感修订，原 L2 浮起方案废弃） |
| 浮层 | `bgElevated` | `#1C2128` | 下拉、弹窗、Popover |
| 表头 | — | `#1F2937` | 表格表头独立次级色 |

配套（暗色必读，均为定稿结论，勿改）：

- **边框要"拒绝隐形"**：输入类 `rgba(255,255,255,.1)`（hover `.2`，focus 主色；2026-09-08 对齐 vben 实测值修订）、通用 `rgba(148,163,184,.28)`、分隔线 `rgba(255,255,255,.08)`（5% 时卡片与大底糊成一片）。
- **文字**：主 `#F8FAFC`、次 `#8B949E`（L1 上 5.77 / 浮层 `#1C2128` 上 5.26，与 2.2 的浅色次要档 `#676E7C` 同为一档，
  2026-09-29 三端逐页实测对齐）、三级 `#6E7681`（L1 上 **3.86**，只给 12px 以上的非关键注记；2026-09-29 的三端逐页实测
  没有任何渲染文本取该值，真要用进正文得先提到次要档）、placeholder `#9CA3AF`（提亮对齐 vben，豁免口径见 2.2 末注）；
  表单标签降一档用 `#DCE3ED`（值是主角）。
- **暗色语义色文字档另算**：暗色下 antd/EP 自己派生一套语义色，浅色的"深墨前景"规则在暗色是倒退，
  详见 2.1「语义色当文字用」的两条踩坑结论。
- **填充/hover**：浮层列表项 hover `rgba(255,255,255,.08)`；fill 四阶 `rgba(255,255,255,.14/.09/.05/.03)`。
- **投影必须用黑**：`0 6px 16px rgba(0,0,0,.45), 0 3px 6px rgba(0,0,0,.3)`（antd darkAlgorithm 派生的白投影在暗底不可见）。
- **遮罩**：Drawer `rgba(0,0,0,.6)`，通用 `rgba(0,0,0,.45)`。
- **主色 α 衍生**（随 2.1 主色联动）：行 hover `rgba(0,107,230,.08)`、选中 `rgba(0,107,230,.15)`、focus 柔光 `0 0 0 3px rgba(0,107,230,.12)`。
- 卡片/抽屉大圆角 12（`borderRadiusLG`），见 2.4。

> **沿革 · 未采纳的备选方案（vben 浅炭暗色）**：大底 `#14161A`（`hsl(220 13.06% 9%)`）+ 表面 `#1C1E23`（`hsl(222.34 10.43% 12.27%)`）。当时评估"若观感偏好更柔和不近黑，整体切换只需替换上表 L0/L1 与浮层三个值 + 文字基色改 `#F2F2F2`，三端各改一处"；**最终未采纳**——vben 已直接落在上表的 react 系（`dark.css:4-5` 的对齐注释为证），此段仅作比选记录。

### 2.4 圆角

| 层级 | 值 | 说明 |
|---|---|---|
| 控件（按钮/输入/选择） | **8px** | vben 基准 `--radius: 0.5rem`；react 端 `radius: "6"→"8"` |
| 卡片/抽屉/弹窗 | **12px** | antd `borderRadiusLG`；Tailwind 语境 = `rounded-xl` |
| 登录/注册等认证面板 | **24px** | vben 基准 `rounded-3xl`（独立大面板，区别于页内卡片） |
| 胶囊 | 999px | Tag、开关类 |

### 2.5 字体

**字体栈按端实测记录，本节不再挂"权威栈"**（2026-09-25 复核：原先列在此处的
`-apple-system … 'PingFang SC' … 'Microsoft YaHei'` 栈**没有任何一端按原样实现**，
故改为逐端记实值。真要统一字体栈是一项待议的新工作，别把现状当成"某端跑偏"）：

| 端 | 落点 | 实测值 |
|---|---|---|
| react | `frontend/admin/react/src/styles/global.css:18-20` | `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'`（无 PingFang / 雅黑，CJK 由 `Noto Sans` 与系统兜底） |
| vue-element | `frontend/admin/vue-element/src/styles/index.scss:43-44` | `'Noto Sans SC', 'Noto Sans JP', 'Noto Sans KR', sans-serif`（自托管 CJK 优先） |
| vue-vben | `framework/base/design/design-tokens/default.css:2-4` | 与 react 同族、仅大小写写法不同：`-apple-system, blinkmacsystemfont, 'Segoe UI', roboto, 'Helvetica Neue', arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'` |

- 字号：控件 14 / 表格 13 / 卡片标题 16 / 页面标题 18；rem 基准 16。
- 字重：正文 400、标题与强调 600。vue-element 已按此落地（`src/styles/index.scss:46`
  `--el-font-weight-primary: 400`；旧值全局 500 见本节沿革）。
- 表格数字列建议 `font-variant-numeric: tabular-nums`。

> 沿革：vue-element 的全局字重旧值 500、以及本小节旧版那条"含 PingFang SC / Hiragino Sans GB /
> Microsoft YaHei 的权威栈"都已退役——后者从未被任何一端按原样实现，保留在表外的参考写法里即可。

### 2.6 布局尺寸

规范值即三端现值（2026-09-25 逐端实测，落点如下）：

| 项 | 值 | 三端落点（现值 = 规范值） |
|---|---|---|
| 侧栏宽度 | **224px** | vben `framework/core/preferences/config.ts:73`；ele `src/core/preferences/config/default.ts:75`（消费点 `src/layouts/LeftLayout.vue:93`）；react `src/layouts/MainLayout/components/SiderMenu/index.tsx:99` 的 224 兜底 |
| 侧栏折叠 | **48px** | react 同文件 `:99`（`isCollapsed ? 48 : …`）；ele `src/layouts/LeftLayout.vue:57` 与 `src/layouts/MixLayout.vue:123` 的 `SIDEBAR_COLLAPSED_WIDTH = 48`；vben 48 ✓ |
| 顶栏高度 | **50px** | react `src/layouts/MainLayout/index.tsx:300`；ele `src/styles/_variables.scss:16` `$navbar-height: 50px`；vben 50 ✓ |
| 页签栏高度 | **38px** | vben `config.ts:78`；ele `core/preferences/config/default.ts:80`；react `core/preferences/config/default.ts:80`（chrome 形态） |
| 内容边距 | **16px** | 由各端页面容器统一施加；ele `src/styles/_layouts.scss:9-10` `.app-container { padding: 16px }` |

> **ele 那两个 SCSS 常量别当布局尺寸读**：`src/styles/_variables.scss:14-15` 的
> `$sidebar-width: 210px` / `$sidebar-width-collapsed: 54px` 只服务 logo 与菜单项图标对齐；
> 侧栏真正的展开/折叠宽度是上表的 224 与 48。
>
> **沿革（退役旧值）**：react 顶栏 56 / 折叠 60、ele 折叠 54、ele `app-container` 15px——这些是
> 2026-09 之前的现值，迁移已完成，不要再按"待办"读。

### 2.7 动效

| 项 | 规范 |
|---|---|
| 页面切换过渡 | `fade-slide`，进入 200–300ms / 离开 ≤200ms，ease-in-out |
| 路由进度条 | 顶部细进度条（react 已有 nprogress） |
| 内容区 loading | 骨架屏优先于 spinner |
| 侧栏折叠/浮层进出 | 150–250ms |
| 无障碍 | 全局动效开关（preferences.transition.enable）必须生效；有条件时尊重 `prefers-reduced-motion` |

---

## 3. 三端落地映射

### 3.1 vue-vben —— 参照实现（视觉基准）

机制：`preferences`（`framework/core/preferences`）→ `update-css-variables.ts` 写 `--primary` 等 HSL 三元组 + `html.dark`/`data-theme` → `useAntdDesignTokens()` / `useElementPlusDesignTokens()` 派生组件库 token。

- 本项目 override 仅 `app.name` / `accessMode`（`src/preferences.ts`），默认主题即本规范基准，**无强制迁移项**。
- 2.1 的语义色文字档 + 2.2/2.3 的次要档落点：`packages/styles/src/antd/index.css` 末尾的 `--gowind-*-text` 段
  （变量由 `hsl(var(--primary|success|warning|destructive))` 混色，跟随主题预设；规则带 `!important`，理由见 2.1 第 2 条）。
  **这一层是纯 CSS 且经 postcss 处理，注释只能写 `/* */`，写 `//` 会在 dev 直接 500（`Unknown word …`）。**
- 曾经的"唯一开放项"已关闭：2.3 的 react 系暗色已被采纳——`framework/base/design/design-tokens/dark.css:4-11`
  的 `--background` / `--background-deep` / `--popover` / `--foreground` 已是近黑蓝系，该文件头注释（2026-09-08）
  即写明"暗色中性色对齐设计语言规范（docs/design-language.md §2.3，近黑蓝系，与 react 端定稿一致）"。

### 3.2 react（antd 6）—— 映射机制与迁移清单

机制：preferences store（`src/core/preferences/store`）→ `useThemeConfig.ts` 组装 ConfigProvider token（`darkAlgorithm` + `darkThemeTokens` + `darkThemeComponents`）；自定义 CSS 一律引用 antd 生成的 `--ant-*` 变量，禁止裸色值。

| 本规范 | 落点文件:行（2026-09-25 实测） | 现值（= 规范值） |
|---|---|---|
| 主色 / 语义色 2.1 | `src/core/preferences/config/default.ts:91`（success `:92`、warning `:93`、destructive `:90`） | `colorPrimary: "hsl(212 100% 45%)"` |
| 圆角 2.4 | 同上 `:95`；大圆角在 `config/darkTheme.ts:67` | `radius: "8"`、`borderRadiusLG: 12` |
| 暗色分层 2.3 | `config/darkTheme.ts:36,37,39,41` | `colorBgBase/colorBgLayout: '#0B0F19'`、`colorBgContainer: '#111827'`、`colorBgElevated: '#1C2128'` |
| 主色 α（暗色） | `config/darkTheme.ts:86,102,103,104` | `activeShadow …0.12`、`rowHoverBg …0.08`、`rowSelectedBg …0.15`、`rowSelectedHoverBg …0.2`，全为 `rgba(0, 107, 230, …)` |
| 顶栏 50 | `src/layouts/MainLayout/index.tsx:300` | `height: 50` |
| 折叠 48 / 展开 224 | `src/layouts/MainLayout/components/SiderMenu/index.tsx:99` | `isCollapsed ? 48 : (sidebarConfig?.width ?? 224)` |
| 语义色文字档 2.1 + 次要档 2.2 | `src/styles/semantic-text.css`（由 `main.tsx` 引入） | `--gowind-*-text` = `color-mix(...)`，规则一律带 `!important`（2.1 第 2 条）；浅/暗两态各一组变量，深墨类规则只在 `html[data-theme='light']` 下生效 |
| 页签激活态文字 | `src/layouts/MainLayout/components/TabsBar/tabsbar.css` | 浅色走 `--gowind-primary-text`；暗色的 chrome/card 形态保持 `var(--ant-color-text)`（本地规则同样带 `!important`，否则被 2.1 那层的新页签规则盖掉） |

> **一处与 2.3 有意的偏差**：react 不覆写表头色——`darkTheme.ts:99-100` 注释说明
> `colorHeaderBg` 不是合法 antd Token（原值 `'#1F2937'` 已移除），表头由 `colorBgContainer/colorFillAlter`
> 派生。2.3 表里的"表头 `#1F2937`"对 react 端只是**观感目标**，不是可直接填的 token。

**沿革（退役旧值，别再去代码里找）**：主色 `#3B82F6`、`radius: "6"`、顶栏 56、折叠 60、暗色主色 α 的
`rgba(59,130,246,…)`。改主色仍要三处同步（token、暗色 α、本文档），且 `localStorage` 残留旧偏好属预期现象，
验证首屏前先清一次。

### 3.3 vue-element（Element Plus + vxe-table）—— 映射机制与迁移清单

机制：preferences（`src/core/preferences`，schema 与 vben 同构）→ `update-css-variables.ts` 运行时注入 `--el-color-primary(-light-N/-dark-2)` 全阶梯 + `--radius`；SCSS（`src/styles/vendors/_element-plus.scss`）为编译期兜底，两处值必须一致。

| 本规范 | 落点文件:行（2026-09-25 实测） | 现值（= 规范值） |
|---|---|---|
| 主色 2.1 | `src/core/preferences/config/default.ts:91` + `constants.ts:12` + `styles/vendors/_element-plus.scss:13` | `hsl(212 100% 45%)` / SCSS `#006BE6`（`info` 亦同值，见 `_element-plus.scss:24-27`） |
| 语义色 2.1 | `default.ts:92,93,90`（运行时注入） | success/warning/destructive 三个 HSL 与本表逐字一致 |
| 暗色 2.3 | `src/styles/_dark-mode.scss:33-36,42-45,54,72` | 大底 `#0B0F19`、表面/输入/抽屉 `#111827`、表头 `#1F2937`、浮层 `#1C2128`——旧 Arco 灰（`#0C0E13/#14171C/#14161a/#1e2026/#1f2329/#2e3440/#0D0F14`）已全部清除 |
| 浅色 canvas / 边框 2.2 | `styles/vendors/_element-plus.scss:31` / `:52` | `#F1F3F6` / `#E4E4E7` |
| 圆角 2.4 | `_element-plus.scss` + `--radius`（`default.ts:95` 为 `"0.5"` = 8px） | base 8px |
| 字重 2.5 | `src/styles/index.scss:46` | `--el-font-weight-primary: 400` |
| 顶栏 / 折叠 / 边距 2.6 | `_variables.scss:16`（50px）/ `LeftLayout.vue:57`、`MixLayout.vue:123`（48）/ `_layouts.scss:9-10`（16px） | 与 2.6 一致 |
| vxe 暗色 | `styles/vendors/_vxe-table.scss:180,183,185,187` | 表头 `#1F2937`、表体/布局底 `#111827`、斑马纹 `rgba(255,255,255,.03)` |
| 语义色文字档 2.1 + 次要档 2.2 | `styles/vendors/_element-plus.scss` 的 `--gowind-*-text` 段 | 把 EP 的 `--el-button-text-color` / `--el-tag-text-color` 指到文字档（`.el-button--{type}` 的 `is-link/is-text/is-plain` 三变体 + `.el-tag--{type}`），带 `!important`——unplugin-element-plus 把组件 CSS 追加在本文件之后，不带就输；次要档 `--el-text-color-secondary: #676E7C` |

**沿革（退役旧值）**：主色 `hsl(220 100% 55%)` / `#165DFF`、浅色 canvas `#F7F8FA`、浅色边框 `#E5E6EB`、
字重 500、折叠 54、`app-container` 15px，以及上面括号里那一整列 Arco 暗色灰。
运行时注入（`update-css-variables.ts`）与 SCSS 兜底两处必须同值，改一处就要同步另一处。

> **曾实测到的不一致（2026-09-28 已修代码）**：SCSS 兜底里的 success / warning 一度仍是 2.1 勘误前的
> `#57D1A0` / `#EF7A48`（`styles/vendors/_element-plus.scss` 的 `$colors` success / warning base），与规范值
> `hsl(144 57% 58%)`≈`#57D188` / `hsl(42 84% 61%)`≈`#EFBD48` 不同；运行时注入走的是规范值，
> 所以只有"运行时注入未覆盖到的场景"（首屏前 / 未执行主题脚本时）会露出旧色。现两处已同值。

---

## 4. 组件级视觉惯例（跨端一致）

- **页面容器**：三端各自容器（react `PageContainer` / ele `ProPage` / vben `Page`），但结构统一为：canvas 大底 → 搜索卡片（surface）→ 工具栏 → 表格卡片（surface + 12px 圆角 + 阴影）。
- **表格**：无边框 + 斑马纹可选；表头独立色（浅 `#F0F2F5` 系 / 暗 `#1F2937`）；行 hover 用主色 8% α；行高紧凑（≤40px，ele 30px 现状可保留）。
- **表格工具栏右侧图标钮（2026-10-08 定稿，react ProTable options 为基准 / ele 已对齐）**：无边框裸图标（hover 才铺 fill 底），颜色 = **主文本色阶**（react `colorText` / ele `--el-text-color-primary`，暗 `#F8FAFC` / 亮 `#1F2937`），不用 secondary——次级灰是为次要文字设计的，做图标唯一内容会显虚。**每枚图标必须带 Tooltip**（刷新/密度/列设置/全屏……）。图标名义 16px 是 antd 口径：antd 图标墨迹占 viewBox ≈90%，EP 图标仅 ≈75%，故 ele 的 EP 图标要放到 **18px** 补偿（等效墨迹 13.5px ≈ antd 16px），vben 若用 lucide 同理注意留白差异。
- **列表加载态（2026-09-29 定稿，react 基准 / ele / vben 三端均已落地）**：三态分工按"形状是否已知 + 耗时是否够长"判，不是骨架/转圈二选一——**首屏**（一行数据都没有）= 表格形状骨架屏；**表内刷新/翻页**（已有数据）= Spin；**请求失败** = 错误态 + 重试入口。
  - 出场阈值统一 **250ms**（骨架与 Spin 共用一个常量）：本地接口常在 100~200ms 返回，早于阈值出现的加载态比"什么都不显示"更闪。实测 201ms 自然加载：骨架 0 帧、Spin 0 帧。
  - 骨架行数（react 口径）= 该页 `pagination.defaultPageSize`；`pagination={false}` 的树表/抽屉没有 pageSize 可依，取常量 10。**另两端的实际口径见下面「以可见区为准」条**。
  - **骨架必须复用页面自己的 columns**（antd `<Table>` 只替换单元格 render），禁止自画灰块：react `/notification/deliveries`（`scroll.x=1670`、12 列）实测骨架态与真实态 `<th>` 的 x/宽/高逐列全等（2026-09-29 骨架接入 `scroll.y` 后两态同为 13 枚：`[41,170,47] [211,130,47] … [1451,260,47]` + 补位 `[1711,15,47]`，逐列等值）；表头底色两态同值（浅 `rgb(250,250,250)`、暗 `rgb(31,41,55)` = 上面「表格」条的 `#1F2937`）。
  - **表头那枚 15px 补位 `<th>` 由"表体有没有自己的纵向滚动条"决定，不由"骨架 vs 真实"决定**：`scroll.y` 生效后 `.ant-table-body` 一旦溢出就补这一列。骨架同样接了 `scroll.y`，实测其表体 `scrollHeight 973 > clientHeight 85` ⇒ 溢出 ⇒ 补位照样出现（先前记的"真实表比骨架多一枚"正是骨架不接 `scroll.y` 时代的产物，现已不成立）。
  - **骨架两向 scroll 都接**（2026-09-29 定稿，取代本条先前两版说法）：`ListTable` 把页面的 `scroll.x` 与 `scroll.y` 一并喂给 `TableSkeleton`。理由是那层限高 `.ant-table-body` 就是分页器落点的决定者——骨架缺它就只能按行数撑高，数据到位后整段跳动（实测跳动 **856px**：骨架 20 行不限高 pagerTop 1565 vs 真实表限高 132.667px pagerTop 709，container 652 / above 471）。接上之后同页复测两态 **pagerTop 590 = 590、above 385 = 385、`maxHeight 100px` = `100px`、pager 高 24 = 24**（margin 0/0 两态同）⇒ hook 的三个输入量逐项相等，由它自己的公式即得两态同高，跳动归零。口径注意：这组数取自后台标签页，`rAF` 不跑 ⇒ `useProTableScrollY` 停在初始 100px，**不是可见区里的 settle 值**；但 settle 与否两态共用同一个公式与同一组输入，所以等式不随之改变。
  - 已知残余差并保留：占位条高 24px（`Skeleton.Button size="small"`）vs 文本行盒 22px ⇒ 每行约 1.3px（实测两态 20 行 `scrollHeight` 973 vs 947 ⇒ 合计 26px）；这 26px 落在限高 body 内部被裁掉，不再推动分页器（旧记的 −43px 是"真实表未限高"时代的量，已作废）。表头位置不动，不为此钉死行高（骨架的诚实口径是"按 pageSize 猜行数"，不是"和结果一样高"）。
  - 占位条颜色取 antd token 渐变（浅 `rgba(0,0,0,.06)→(0,0,0,.15)→.06`、暗 `rgba(255,255,255,.08)`），**组件内不写死任何颜色**；本次浅色态与暗色态都以 computed style 数值对照给出（上面的 thBg/barBg），截图对照不可用——测量标签页 `visibilityState:hidden` 下 `take_screenshot` 不出图。
  - 首屏失败用 `Empty` + 「重试」而不是"暂无数据"（后者会被读成"查询成功但结果为空"）；已有数据后刷新失败在表格上方挂 `Alert type="error"` + 重试，**旧数据保留**（实测 rows 20→20，点一次重试即恢复）。
  - **列表页的 `request` 不许自己 catch**：错误一律抛给 `ListTable`，由它 `console.error` 带出原始错误对象，并把 `error.message` 直接显示在内联错误态里。此前 39 页中 34 页写的是 `catch → message.error → return {success:false}`——既是"只有 toast 没有日志"（违反全仓铁律 1），又让内联只能显示通用文案「加载失败」；改后 toast 不再与内联重复，原因常驻在表格上方且带重试入口（实测 `/notification/deliveries` 翻页失败：内联「网络连接错误,请检查网络设置后重试」+「重试」，toasts 0 条，控制台 `[RequestClient]` 与 `[ListTable]` 两行都带原始错误）。页面自己的**写操作**（增删改）仍可 `message.error`，那条路不经过表格。
  - 偏好设置 `transition.loading`（「页面切换 Loading」）自此有真消费者：控制首屏骨架，关掉后骨架 0 帧、Spin 接管（实测 `sk=0, spinSpinning=1`），不再是面板里的死配置。
  - **骨架行数以可见区为准，不是照抄 pageSize**：react 取 `defaultPageSize`，多出的行由接入的 `scroll.y` 那层限高 body 裁掉（可见行数因此自然等于真实可见行数，见上面「骨架两向 scroll 都接」条）；ele 的骨架卡是无滚动 flex 块，内容区实测 488px / 行高 39.8px ⇒ 铺 20 行会溢出裁切，保持 8 行；vben 的骨架画在 vxe 的 loading 遮罩里，遮罩多高由**空表表体**决定（实测 676×140）⇒ 挂载时同步量一次自身高度按 32px 折算（140 ⇒ 表头 + 3 行），`min(pageSize, 8)` 只是上限。
  - **vben 接入点 = `plugins/src/vxe-table/use-vxe-grid.vue` 一处**（39 个列表页共用，页面侧零改动）：`api.ts` 加 `queryLoading/queryError/hasLoaded` 三个 ref，`extends.ts` 只包 `proxyConfig.ajax.query` 写入它们，骨架/横幅/空态由该组件渲染。三处非显然的坑，改这块前先读：
    ① `init()` 必须**先 `extendProxyOptions` 再发首个 `reload`**（原顺序是先 reload 后包装）——VxeGrid 的 props 要到下一次渲染才换上包装函数，首屏那一次走未包装的原始函数，`hasLoaded`/`queryError` 都不被写入，三态直接失效；
    ② `proxyConfig.showLoading=false` **不足以**让 vxe 不点亮遮罩：`currLoading = isColLoading || isRowLoading || loading` 是它装列/载数据的内部状态，既不看 `showLoading` 也不看 250ms 阈值（实测快响应时遮罩仍然 `display:block`）。做法是把 `--vxe-ui-loading-background-color` 改 `transparent`、`#loading` 槽内容自己按 `delayedLoading` 决定，快响应时槽里什么都不渲染；
    ③ 槽里的容器要能拿到高度：vxe 把内容装进 `top:50% + translateY(-50%)` 的**零高盒**，骨架写 `height:100%` 会跟着塌成 0（实测 DOM 在、paint 面积 0，看起来"骨架没生效"），`style.css` 把 `.vxe-grid .vxe-table--loading > .vxe-loading--wrapper` 拉成 `top:0;height:100%;transform:none` 后两分支（骨架 / VbenLoading）才真正铺满表体。
  - vben 失败态落点（`/notification/deliveries` 实测，判一律取 painted 面积 + `visibility`，不看 DOM 是否存在）：快响应（<250ms）整轮骨架/Spin 均 0；慢首屏骨架 painted `676x140`、3 行、数据到位后归位；慢刷新（已有数据）骨架 0、`VbenLoading` 根节点 `676x140` 且 `invisible opacity-0` 已摘；刷新失败保留旧行（rows 恒 20）、横幅 painted `676x49` 落在 `top` 槽内、卡片自行增高 ~60px 而 `document.scrollHeight` 保持 772 不引页面滚动、点「重试」后横幅消失 `queryError` 归 null；首屏失败空态容器 painted `676x80` 显示原因 + 一个可见「重试」（vxe 同时挂着 `.vxe-table--empty-block` 的 `visibility:hidden` 克隆，按存在计数会读成 2）。
  - **测量环境的一条限制**：标签页 `visibilityState:hidden` 时**没有渲染帧 ⇒ `ResizeObserver` 回调、`requestAnimationFrame` 与 CSS transition 都不推进**（实测遮罩里 `clientHeight=140` 而 RO 计数 0、`opacity` 恒 0；react 侧同类后果是 `useProTableScrollY` 停在它自己返回的初始 `100px`，量到的表体高度**不是用户看到的那个**）。所以 vben 的骨架行数改用 `onMounted` 一次性同步测量，不用 RO——否则在后台标签里永远量不到，且量具自己也读不出真假；在后台标签读 react 表格高度时，报告里必须写明读的是"未 settle 的初始值"。
- **表单/抽屉**：输入控件底与表面同层（暗 `#111827` / 浅白），以 `rgba(255,255,255,.1)` 边框区分；focus 主色边框 + 3px 12% 柔光；抽屉遮罩 `rgba(0,0,0,.6)`，宽度基准 480。
- **按钮质感（2026-09-29 定稿，暗/浅两态一致）**：主按钮两态一律用 UI 库原生的扁平实色 + 原生 `0 2px 0` 底投影，**禁止叠顶部高光渐变与品牌色发光投影**（暗色的层次由 2.3 的"明度随层级递增 + 黑投影"表达，不靠发光；浅色的克制口径见 2.2 末）。**hover 态两态同机制：实底档改 `background-color`/`border-color`、文字/链接档改 `color`，禁止用 `filter: brightness()` 只给暗色加一层亮度**（暗色静止态常带 `!important`，会压掉库原生的 hover 背景色，于是 filter 成了暗色唯一的悬停反馈——既不对称又让"关不掉"）。落地：react 删 `pro-components-dark.css` 原三条质感规则；ele `_dark-mode.scss` 原 7 处 hover `filter: brightness(1.15)` 全部换成 `color-mix(in srgb, var(--dark-*) 85%, #ffffff)`；vben 无此类装饰。ele 实测：暗 `#006BE6` → hover `rgb(38,129,234)`，浅 `#006BE6` → hover `rgb(6,81,167)`（EP 原生 `--el-button-hover-bg-color`），两态 `filter` 均为 `none`；方向不同是故意的（暗色向亮、浅色向暗，见 2.3 层级模型）。
- **禁用态一律去实底（2026-09-29 定稿，基准 = react）**：语义色实底按钮进 disabled 必须换成中性档——浅 `rgba(11,15,25,.05)` 底 + `rgba(11,15,25,.25)` 字 / 暗 `rgba(248,250,252,.05)` 底 + `rgba(248,250,252,.25)` 字，**不许"仍是主色底只是变浅一档"**（用户读作"点了没反应"而不是"还不能点"）。
  踩坑实测（ele `/ai/chat` 发送按钮，空输入 vs 有输入两态对撞）：暗色下底色/文字/边框**三项零差异**（两态都是 `#006BE6` + `#FFFFFF`，cr 同为 4.95），唯一线索只剩 `cursor:not-allowed`；浅色下是 `#80B5F3` + 白（2.14），仍是一枚淡蓝药丸。
  基准端同页两态：react 暗 `#1D2432` + `#4B515C`（1.95）、浅 `#F5F5F5` + `#BFBFBF`（1.69），vben 暗 `#232A38` + `#4B515C`。改后 ele 实测暗 `#1D2332` + `#4B515C`、浅 `#F3F3F4` + `#C2C3C6`，与 react 同档。
  **两个来源要分别修**（只修一个会留下"变量已解析对、底色纹丝不动"的假修复）：暗色 = 我们自己的 `_dark-mode.scss` 给 `.el-button--{type}` 刷 `!important` 实底时没排除 `is-disabled`（已补 `:not(.is-disabled)`，primary/success/warning/danger 各一处，嵌套的 `&:hover` 随之一起排除）；浅色 = EP 原生 `--el-button-disabled-bg-color: <type>-light-5` 且前景留白字（在 `vendors/_element-plus.scss` 把 `--el-button-disabled-*` 三件套指到中性 rgba，走变量而非直接改 `background-color`，因为 EP 原生就从这个变量取）。
  禁用件按 WCAG 1.4.3 属 inactive 豁免，不参与 4.5 判定（`style-sweep.mjs` 的 `dis()` 已按 `.is-disabled/[disabled]` 豁免——改这条前先确认量具跟上，否则全页扫测会凭空多出几十条"违规"）。
  **边界**：`is-loading` ≠ `is-disabled`（EP 只把 loading 记进 `ariaDisabled`/`disabled` 属性，类名不带 `is-disabled`），实测发送中按钮仍是 `#006BE6` 实底 + spinner，"去实底"不会把加载态一起画灰；`is-plain/is-text/is-link` 的禁用态走 EP 自己的淡档（light-9 底 + light-5 字，是显式声明不是变量），本次未接管（全仓仅 1 处 `plain` 按钮：`system/script/script-log-dialog.vue:37`）。
- **认证页（登录/注册）**：画布深底 + 实底表面卡（24px 大圆角、主色柔影）；品牌插画带 vben 同款 float 动效（`translateY 0→-20px→0`，5s 循环，尊重 `prefers-reduced-motion`）。
- **页签栏**：chrome 形态、38px。选中页签走"温和配方"（2026-09-16 修订，基准 = vben）：暗色 = 中性灰底（fill ≈ 白 10%）+ 正常亮文字，浅色 = 主色 15% 底 + 主色文字；**禁用主色描边 / 发光阴影 / 底部指示线 / 字重加粗**（形状本身即指示）；悬停 = 中性微底，关闭按钮跟随文字色不用主色。
- **默认头像**（2026-09-16 统一）：三端统一使用橘猫插画 `default-avatar.png`（react/ele public 同文件，vben 经 `src/preferences.ts` 覆盖框架默认的 webp——带 Vben 品牌字样已弃用）；用于导航栏当前用户、通知、锁屏等无头像兜底；用户列表/详情的"姓名首字 + 底色"兜底保留（承载身份信息）。
- **侧边栏菜单交互态（2026-09-16 定稿，基准 = react antd Menu）**：悬停 = 中性灰遮罩（浅 `#F5F7FA` 系 / 暗 `rgba(255,255,255,.05~.08)`，8px 圆角）；选中 = 主色实底 + 白字（`--primary-foreground`）+ 8px 圆角药丸，不得用左侧竖条 / 淡色底 / 字重加粗来区分选中（vben 旧"选中与悬停同灰"、ele 旧"inset 蓝条 + light-9 淡底"均废弃）；父级展开链路只做文字/图标提亮，不铺底色。折叠后的弹出子菜单选中态同规则。横向顶部菜单暂不约束。
- **实底主色标"位置"，不标"次级动作"（2026-09-29 定稿，基准 = react）**：实底主色块留给状态标识（上一条的导航选中、列表/分页的当前选中）与页面级主行动（提交、发送）；
  **列表面板里的"新建 X"这类次级动作按钮一律中性描边**——它和导航选中同色同形时，用户分不清哪个是菜单高亮、哪个是按钮。
  踩坑实测（ele `/ai/chat` 左栏）：`<ElButton type="primary" plain>` 浅色下是淡底 `#E6F0FD` + 文字档 `#0162D2`（4.96 ✓），
  **暗色下 EP 把 plain 直接渲染成 `#006BE6` 实底 + 白字**（4.95），与侧边栏选中项、会话选中行叠成三枚一模一样的蓝药丸；
  改回默认按钮后三端同形（react 实测 `#FFFFFF` 浅 / `#1C2128` 暗，vben 为 `a-button` 默认态）。
  反向结论一并记下：**别用"淡底 + 主色文字"去降"面板内选中"这一档**——暗色淡底 `#121D29` 与面板 `#111827` 只差
  **1.04:1**（浅色 `#E6F0FD` vs 白 1.15:1），铺不出可辨的选中块，实底主色在这一档是必要的
  （把淡底加浓到文字对比仍 ≥4.5 的上限 ≈14% 主色：浅 4.67 / 暗 5.16，底色与面板差也只有 1.22:1 / 1.14:1）。
- **图表**：数据色板 = 主色阶梯（-300~-700）+ 语义色；枚举分类名本地化复用既有 i18n 命名空间，不新增同义 key。
- **空态**：文字性空态，不引入插画资源。
- **错误/兜底页（401/403/404/500/offline/coming-soon）**（2026-09-16 定稿）：插画填色一律走 `--fb-*` 插画语义变量（`primary/ink/paper/mist/mist-2/line/navy/navy-deep/skin/skin-light`），SVG 内禁用裸色值。亮色为"纸墨日光"原色；暗色为"月夜"版——装饰件压暗至画布上方一档（mist `#182136` / mist-2 `#22304a` / line `#33405e`）、藏青物件提亮保形（navy `#46547a` / navy-deep `#38456a`）、主色提亮一档（`color-mix(in srgb, 主色 78%, white)`）。主色锚点接线：react 由 ThemeProvider 写 `--app-color-primary` 到 `<html>`，ele 复用 `--primary-hsl`。插画后方垫主色柔光晕（radial-gradient `--fb-glow`，暗色更明显）；进场 fade+上浮 450ms 依次错峰 + 插画 6s 悬浮呼吸，动效尊重 `preferences.transition.enable` 与 `prefers-reduced-motion`。
- **全局搜索面板（2026-10-02 定稿，基准 = vben SearchPanel/global-search）**：三端同形——
  - **容器**：居中模态 600px（非 popover/抽屉）；头部 = 搜索图标 + 无边框大输入（同一 baseline，底部 1px 分隔线）；结果区限高 ~450px 内滚；底部键位提示条（`↵` 选择 · `↑↓` 切换 · `ESC` 关闭，muted 小字）与正文 1px 分隔。触发 = 顶栏胶囊条（放大镜 + "搜索" + `Ctrl K` kbd 徽标）与 Ctrl/Cmd+K，三端一致。
  - **结果行**：图标 + 标题一行为一项（路径不入行文，行紧凑、圆角 6px）；键盘/悬停选中 = **主色实底 + 白字**（同"侧栏菜单选中"惯例，禁止淡底 + 主色文字——暗色淡底与面板差仅 ~1:1 铺不出选中块）；行尾可带删除按钮（仅历史行）。
  - **分区**：无关键词 = "最近"历史（localStorage，上限 5，可单条删）；有关键词 = 本地菜单命中在前，**语义搜索独立小节**（12px muted 小节标题"语义搜索" + 语义行 = 标题 + 右侧灰字 route；搜索中显示"搜索中…"；失败 console.error 降级为仅本地，不弹错）。空态统一文案（"暂无最近搜索" / "未找到相关页面"），纯文字不配插画。
  - **本地化**：菜单标题在**建索引时翻译**（ele 教训：`meta.title` 是 i18n key，裸 key 入索引 = 中文永远搜不到）；搜索匹配口径 = 标题或路径 contains。

---

## 5. 治理规则

1. **本文档先行**：视觉值变更 = 改本文档 + 三端同步迁移 + 暗浅两态截图对照，一个 PR 完成。
2. **禁止硬编码**：组件内不得出现裸色值/裸圆角；一律走 token（antd token / `--el-*` / `--ant-*` / tailwind 主题桥接）。
3. **新增语义色**：先入 2.1 表，再三端同补，含色阶派生。
4. **preferences 是用户可调入口**：本表定义"默认值与基线"，不冻结用户的主题预设/圆角/明暗个性化；三端内置主题预设列表（violet/pink/…）应保持同名单。
5. **暗色规则沉淀**：结构属性两端通用，颜色按主题分支；antd v6 覆盖前先 querySelector 验证 DOM（v6 已移除 `.ant-select-selector` 等）。
6. **验证门禁**：视觉迁移 PR 必须跑三端 typecheck（react `npm run typecheck` / ele `npx vue-tsc --noEmit` / vben `pnpm run check:type`），并附暗/浅 × 三端对照截图。

---

## 6. 迁移顺序建议

> 三端迁移**已完成**（2026-09-25 逐端复核，落点见 §3.2 / §3.3）。本节保留为"未来跨端视觉变更"的
> 工序参考，不再是待办清单。

1. **react 先行**（行为基准端）：主色 + 圆角 + 顶栏/折叠尺寸，一次小 PR，浏览器暗/浅两态验证。
2. **vue-element 跟进**：主色/语义色 + 暗色底统一 + 尺寸，改动集中在 4 个样式/配置文件。
3. **vben 收尾**：按最终决议决定是否把暗色 default 主题对齐 react 系 4 个值（**已决**：采纳 react 系，见 §3.1 与 `dark.css:4-11`）。

> 维护记录：2026-09-08 首版定稿（基准取 vben 视觉语言 + react 暗色中性色定稿）；2026-09-13 勘误 2.1 表 success/warning ≈HEX，并补记 vue-element 端收尾迁移（暗色文字层次/抽屉输入同层化/Tag 与图表色板对齐 react/浅色 Arco 灰清除）；2026-09-16 新增 §4 错误/兜底页 `--fb-*` 插画语义变量与两态色板（react/ele 已迁移并暗浅两态实测，vben 维持 `--primary/--foreground` 现状）；2026-09-25 复核：确认 §2.6/§3.2/§3.3 的"现值 → 目标"清单**已落地**，改写为落点索引（逐条 file:line）+ 沿革，§2.5 字体栈改记三端实测值，§2.2 标注 react 浅色无 token 覆写，并补"三选一口径"说明；2026-09-28 三端逐页暗色实测（对比度探针，react/ele/vben 各 39–40 路由）：新增 2.1「语义色实底上的前景」条并把 react/vben 任务页"启动全部"按钮前景改深色墨，修 ele 端 SCSS success/warning 与运行时注入不同值（见 3.3 注），清理 ele 面包屑/AI 问数/分析卡的裸色值；2026-09-29 三端逐页浅/暗两态复测（react 39 路由、ele/vben 各 50 路由，
单次实测 5.4k~6.8k 个文本节点，探针口径：4.5，大字号/粗体档 3.0）：新增 2.1「语义色当文字用」文字档表
（三端各落一层覆盖，违规读数 react 浅 333→0 / 暗 251→0、vben 浅 484→0 / 暗 93→0、ele 浅 80→0 / 暗 38→0）；
2.2 次要档 `#6B7280`→`#676E7C` 并定稿占位符档的豁免边界；2.3 补暗色语义色的两条踩坑结论（浅色深墨规则在暗色是倒退）。
vben 端另修 `registerGlobComp.ts` 漏注册 Radio（AI 问数页 `a-radio-group` 整块不渲染，实测 console 的
`Failed to resolve component` 两条随注册消失）；同日按新 §4「实底主色标位置不标次级动作」条，把 ele AI 助手页
"新建对话"按钮从 `type="primary" plain`（暗色实测 `#006BE6` 实底 + 白字，与侧边栏选中同色同形）改回默认按钮，
`/ai/chat` 暗/浅两态复测各 52 个文本节点、违规 0。同日按新 §4「按钮质感」条删 react 暗色主按钮的高光渐变 +
品牌色发光投影（原 `pro-components-dark.css:513-524` 三条，含配套的 hover/active `filter: brightness`），
暗/浅两态同一按钮实测计算值对齐：`backgroundImage` 均 `none`、`filter` 均 `none`、`boxShadow` 同为库原生
`0 2px 0`（暗 `rgba(3,129,249,.21)` / 浅 `rgba(5,165,255,.1)`），差异只剩主色本身。
同日把 ele 端同族装饰一并收敛：`_dark-mode.scss` 7 处 hover `filter: brightness(1.15)`（实底 primary/success/warning/danger
四组 + 表格体 danger 文字档 + `.button-group` 内两处）改为 `color-mix(in srgb, var(--dark-*) 85%, #ffffff)` 的
`background-color`/`border-color`/`color`；stylelint 错误数与 HEAD 基线同为 118（本次零新增），`/opm/profile`
真 hover 实测暗 `#006BE6`→`rgb(38,129,234)`、浅 `#006BE6`→`rgb(6,81,167)`，两态 `filter` 均 `none`。
**顺带记一条排查教训**：这条尾巴最初是按 grep 命中 `filter: brightness`（`_dark-mode.scss:205`）报出来的，实测该
选择器 `html.dark .el-form--inline .button-group …` 在页面上匹配 **0 个元素**（全 src 检索 `button-group` 只命中这行
选择器自身，`/opm/users` 实测 `.button-group` 计数 0、`.el-form--inline` 计数 1），查询区实际类名是
`.pro-search__actions` —— 整块 `.button-group { … }` 是死代码，本次只把它里面的 filter 换掉、**未删块**，
要清理另开一次改动（grep 命中 ≠ 生效，先量匹配数）；同日补 §4「禁用态一律去实底」条：ele `/ai/chat` 发送按钮空输入态暗色实测与可点态三项（底色/文字/边框）零差异，
根因两处分别修——`_dark-mode.scss` 的 `.el-button--{type}` !important 实底补 `:not(.is-disabled)`、`vendors/_element-plus.scss` 把 `--el-button-disabled-*` 指到中性 rgba；
改后 ele 暗 `#1D2332`+`#4B515C` / 浅 `#F3F3F4`+`#C2C3C6`，与 react（`#1D2432`+`#4B515C` / `#F5F5F5`+`#BFBFBF`）同档，loading 态实测未被连带画灰（4/4 采样仍实底）。
同日按新 §4「列表加载态」条把 react 列表页的加载态三态化：新增 `src/components/common/ListTable/{index.tsx,TableSkeleton.tsx}`（首屏骨架 + 250ms 阈值 Spin + 失败态与重试），
39 个列表页只把 `<ProTable>` 换成 `<ListTable>`（import 与标签两处，`request`/`columns`/`actionRef` 用法不变，实测替换后页面上 `<ProTable` 计数 0、`<ListTable<` 计数 39），
`locales/{zh-CN,en-US}/_core/common.json` 各加 1 个 `button.retry`。**ele/vben 两端未移植**（按"react 先行、其余移植"工序，本轮只落 react）；
骨架/错误态的实测数值（逐列全等、分页器 43px 残余差、rows 20→20 保留）记在 §4 该条内，截图对照因测量标签页 `visibilityState:hidden` 不可用。
同日续：把 39 个列表页 `request` 里的 `try/catch → message.error → return success:false` 全部剥掉（其中 34 页此前只有 toast、没有日志，属违反全仓铁律 1），
错误改由 `ListTable` 一处 `console.error` 带原始对象并内联显示 `error.message`；顺手删两个零消费者死目录
`react/src/layouts/components/LoadingSkeleton/`（12 文件，其 `presets/TableSkeleton.tsx` 与本次骨架实现重复）与
`react/src/components/common/PageContainer/index.tsx`（**活的那份在 `react/src/layouts/components/PageContainer/`，两者同名勿混**）。
承上，「ele/vben 两端未移植」那句话后续由两笔补齐：
**ele**（`b9dd3182`）接在 `components/Pro/ProPage/index.vue` + `composables/useTableState.ts` 这一处接入点上（38 个列表页无需逐页改），
`transition.loading` 从此有消费者、表内 spinner 加 250ms 阈值（注入 900ms 延迟实测 mask 在 273ms/275ms 两把量具同值出现）、
失败态无数据整块换 `el-empty`+重试（实测首屏失败 → 内联「网络连接错误,请检查网络设置后重试」，点重试 rows 0→20）、有数据在表格上方挂 `el-alert` 且保留旧行；
delivery/rule 两页的 `listAction` 自 catch 一并剥掉；`locales/{en-US,zh-CN}/common.json` 加 `common.button.retry`；骨架保持 8 行（理由见 §4「以可见区为准」条）。
**vben**（2026-09-29 续）接在 `framework/effects/plugins/src/vxe-table/`（新增 `table-skeleton.vue`，改 `use-vxe-grid.vue`/`extends.ts`/`api.ts`/`style.css`，
`packages/locales` 两语各加 `common.loadDataFailed`/`common.retry`，`views/app/notification/{delivery,rule}/index.vue` 的 `query` 不再自 catch），
六个场景 live 实测通过、数值记在 §4 该条内；`apps/admin` 下 `npx vue-tsc --noEmit --skipLibCheck` 退出码 0
（阳性对照：往新文件塞一行类型错误后同一条命令报 2 处并指向该文件，确认它真在检查范围内，不是"看不见所以没错"）。
本轮踩到并修掉的两个假绿：① 骨架 `height:100%` 塌在 vxe 的零高遮罩盒里（DOM 在、paint 面积 0），先前"骨架 8 行且布局不动"的读数只数了 DOM 存在；
② `init()` 原先先 `reload` 后包 `query`，首屏那一次绕过包装函数 ⇒ `hasLoaded` 永为 false，一次冷加载出两段加载态。
