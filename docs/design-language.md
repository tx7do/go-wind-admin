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
| **暗色中性色** | **react 六轮定稿（2026-08-24）** | 近黑蓝 `#0B0F19` 系。**三端均已落在该系**：react `core/preferences/config/darkTheme.ts:37,39`、vue-element `styles/_dark-mode.scss:33-36,54,72`、vben `packages/@core/base/design/src/design-tokens/dark.css:4-11`（该文件注释即写明对齐本规范 2.3）。vben 原"浅炭暗色"降为沿革，见 2.3 末 |
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

**色阶派生规则**：由基准色程序化生成 -50 ～ -900 阶梯（vben / vue-element 已有 `generatorColorVariables`；react 端由 antd `defaultAlgorithm/darkAlgorithm` 自动派生 hover/active）。交互态规则：**hover 取亮一阶、active 取暗一阶，禁止手工挑色**。EP 侧 `light-3/5/7/8/9`、`dark-2` 一律由脚本生成，不手写。

### 2.2 中性色 · 浅色模式

> **落地状态按端不同**（2026-09-25 复核）：vue-element ✓（`styles/vendors/_element-plus.scss:31` 页面画布
> `#F1F3F6`、`:52` 边框 `#E4E4E7`）、vue-vben ✓（`packages/@core/base/design/src/design-tokens/default.css:10`
> `--background-deep: 216 20.11% 95.47%` = `#F1F3F6`）、**react ✗**——react 端只有
> `core/preferences/config/darkTheme.ts` 一套 token 覆盖，**浅色模式没有任何 token 覆写**，
> 画布/边框走 antd `defaultAlgorithm` 的默认派生值，与本表并不严格相等。
> 本表因此对 react 是"目标值"而非"现值"；要 react 浅色严格对齐本表，需要新增一套 light tokens（未做，属新工作）。

| Token | 值 | ≈HEX | 说明 |
|---|---|---|---|
| canvas（页面画布） | `hsl(216 20.11% 95.47%)` | `#F1F3F6` | 大底，卡片在其上"浮起" |
| surface（卡片/容器） | `#FFFFFF` | — | 卡片、表格容器、顶栏、侧栏 |
| 边框 | `hsl(240 5.9% 90%)` | `#E4E4E7` | 通用边框（≈ zinc-200） |
| 文字-主 | `#1F2937` | — | 标题、重点（gray-800） |
| 文字-常规 | `#374151` | — | 正文（gray-700） |
| 文字-次要 | `#6B7280` | — | 辅助说明（gray-500） |
| 文字-占位 | `#9CA3AF` | — | placeholder（gray-400） |

灰阶统一走 Tailwind gray/slate 族（与暗色同一冷调体系）。浅色投影：卡片静止 `0 1px 2px rgba(0,0,0,.05)`，悬浮态 `0 4px 12px rgba(0,0,0,.08)`，克制不发光。

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
- **文字**：主 `#F8FAFC`、次 `#8B949E`、三级 `#6E7681`、placeholder `#9CA3AF`（提亮对齐 vben）；表单标签降一档用 `#DCE3ED`（值是主角）。
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
| vue-vben | `packages/@core/base/design/src/design-tokens/default.css:2-4` | 与 react 同族、仅大小写写法不同：`-apple-system, blinkmacsystemfont, 'Segoe UI', roboto, 'Helvetica Neue', arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'` |

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
| 侧栏宽度 | **224px** | vben `packages/@core/preferences/src/config.ts:73`；ele `src/core/preferences/config/default.ts:75`（消费点 `src/layouts/LeftLayout.vue:93`）；react `src/layouts/MainLayout/components/SiderMenu/index.tsx:99` 的 224 兜底 |
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

机制：`preferences`（`packages/@core/preferences`）→ `update-css-variables.ts` 写 `--primary` 等 HSL 三元组 + `html.dark`/`data-theme` → `useAntdDesignTokens()` / `useElementPlusDesignTokens()` 派生组件库 token。

- 本项目 override 仅 `app.name` / `accessMode`（`apps/admin/src/preferences.ts`），默认主题即本规范基准，**无强制迁移项**。
- 曾经的"唯一开放项"已关闭：2.3 的 react 系暗色已被采纳——`packages/@core/base/design/src/design-tokens/dark.css:4-11`
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

**沿革（退役旧值）**：主色 `hsl(220 100% 55%)` / `#165DFF`、浅色 canvas `#F7F8FA`、浅色边框 `#E5E6EB`、
字重 500、折叠 54、`app-container` 15px，以及上面括号里那一整列 Arco 暗色灰。
运行时注入（`update-css-variables.ts`）与 SCSS 兜底两处必须同值，改一处就要同步另一处。

> **实测到的不一致（待修代码，不是待修文档）**：SCSS 兜底里的 success / warning 仍是 2.1 勘误前的
> `#57D1A0` / `#EF7A48`（`styles/vendors/_element-plus.scss:16,19`），与规范值
> `hsl(144 57% 58%)`≈`#57D188` / `hsl(42 84% 61%)`≈`#EFBD48` 不同；运行时注入走的是规范值，
> 所以只有"运行时注入未覆盖到的场景"（首屏前 / 未执行主题脚本时）会露出旧色。按上面的"两处必须同值"规则，
> 这两行 SCSS 值应对齐 2.1。

---

## 4. 组件级视觉惯例（跨端一致）

- **页面容器**：三端各自容器（react `PageContainer` / ele `ProPage` / vben `Page`），但结构统一为：canvas 大底 → 搜索卡片（surface）→ 工具栏 → 表格卡片（surface + 12px 圆角 + 阴影）。
- **表格**：无边框 + 斑马纹可选；表头独立色（浅 `#F0F2F5` 系 / 暗 `#1F2937`）；行 hover 用主色 8% α；行高紧凑（≤40px，ele 30px 现状可保留）。
- **表单/抽屉**：输入控件底与表面同层（暗 `#111827` / 浅白），以 `rgba(255,255,255,.1)` 边框区分；focus 主色边框 + 3px 12% 柔光；抽屉遮罩 `rgba(0,0,0,.6)`，宽度基准 480。
- **认证页（登录/注册）**：画布深底 + 实底表面卡（24px 大圆角、主色柔影）；品牌插画带 vben 同款 float 动效（`translateY 0→-20px→0`，5s 循环，尊重 `prefers-reduced-motion`）。
- **页签栏**：chrome 形态、38px。选中页签走"温和配方"（2026-09-16 修订，基准 = vben）：暗色 = 中性灰底（fill ≈ 白 10%）+ 正常亮文字，浅色 = 主色 15% 底 + 主色文字；**禁用主色描边 / 发光阴影 / 底部指示线 / 字重加粗**（形状本身即指示）；悬停 = 中性微底，关闭按钮跟随文字色不用主色。
- **默认头像**（2026-09-16 统一）：三端统一使用橘猫插画 `default-avatar.png`（react/ele public 同文件，vben 经 `apps/admin/src/preferences.ts` 覆盖框架默认的 webp——带 Vben 品牌字样已弃用）；用于导航栏当前用户、通知、锁屏等无头像兜底；用户列表/详情的"姓名首字 + 底色"兜底保留（承载身份信息）。
- **侧边栏菜单交互态（2026-09-16 定稿，基准 = react antd Menu）**：悬停 = 中性灰遮罩（浅 `#F5F7FA` 系 / 暗 `rgba(255,255,255,.05~.08)`，8px 圆角）；选中 = 主色实底 + 白字（`--primary-foreground`）+ 8px 圆角药丸，不得用左侧竖条 / 淡色底 / 字重加粗来区分选中（vben 旧"选中与悬停同灰"、ele 旧"inset 蓝条 + light-9 淡底"均废弃）；父级展开链路只做文字/图标提亮，不铺底色。折叠后的弹出子菜单选中态同规则。横向顶部菜单暂不约束。
- **图表**：数据色板 = 主色阶梯（-300~-700）+ 语义色；枚举分类名本地化复用既有 i18n 命名空间，不新增同义 key。
- **空态**：文字性空态，不引入插画资源。
- **错误/兜底页（401/403/404/500/offline/coming-soon）**（2026-09-16 定稿）：插画填色一律走 `--fb-*` 插画语义变量（`primary/ink/paper/mist/mist-2/line/navy/navy-deep/skin/skin-light`），SVG 内禁用裸色值。亮色为"纸墨日光"原色；暗色为"月夜"版——装饰件压暗至画布上方一档（mist `#182136` / mist-2 `#22304a` / line `#33405e`）、藏青物件提亮保形（navy `#46547a` / navy-deep `#38456a`）、主色提亮一档（`color-mix(in srgb, 主色 78%, white)`）。主色锚点接线：react 由 ThemeProvider 写 `--app-color-primary` 到 `<html>`，ele 复用 `--primary-hsl`。插画后方垫主色柔光晕（radial-gradient `--fb-glow`，暗色更明显）；进场 fade+上浮 450ms 依次错峰 + 插画 6s 悬浮呼吸，动效尊重 `preferences.transition.enable` 与 `prefers-reduced-motion`。

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

> 维护记录：2026-09-08 首版定稿（基准取 vben 视觉语言 + react 暗色中性色定稿）；2026-09-13 勘误 2.1 表 success/warning ≈HEX，并补记 vue-element 端收尾迁移（暗色文字层次/抽屉输入同层化/Tag 与图表色板对齐 react/浅色 Arco 灰清除）；2026-09-16 新增 §4 错误/兜底页 `--fb-*` 插画语义变量与两态色板（react/ele 已迁移并暗浅两态实测，vben 维持 `--primary/--foreground` 现状）；2026-09-25 复核：确认 §2.6/§3.2/§3.3 的"现值 → 目标"清单**已落地**，改写为落点索引（逐条 file:line）+ 沿革，§2.5 字体栈改记三端实测值，§2.2 标注 react 浅色无 token 覆写，并补"三选一口径"说明。
