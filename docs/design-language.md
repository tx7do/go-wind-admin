# 设计语言规范（Design Language）

> **定位**：react / vue-element / vue-vben 三端共享的视觉语言权威值表。
> **唯一权威**：任何颜色、圆角、尺寸、动效的调整，先改本文档，再同步三端代码；禁止只改代码不改文档。
> **参照实现**：vben 端的主题体系（preferences → CSS 变量 → 组件库 token 派生）是视觉基准与架构参照；react / vue-element 按第 3 章映射表落地。

---

## 1. 基本决策与理由

| 决策 | 取值来源 | 理由 |
|---|---|---|
| 主题模型 | vben | HSL token 三元组 + CSS 变量下发 + 组件库 token 派生，三端 preferences schema 已同构 |
| 主色 / 语义色 | vben 默认值 | `hsl(212 100% 45%)` 深蓝更"企业级"，且 vben/语义色三端 config 本就同源 |
| 圆角体系 | vben | `--radius: 0.5rem`（控件 8px），卡片/浮层 12px |
| **暗色中性色** | **react 六轮定稿（2026-08-24）** | 近黑蓝 `#0B0F19` 系。vue-element 现行暗色（`#0C0E13/#14171C`）与之同族，**三端中两端已趋同**；vben 的浅炭暗色列为备选（见 2.3，切换成本 4 个值） |
| 布局尺寸 | vben（ele 顶栏已一致） | 侧栏 224 / 折叠 48 / 顶栏 50 / 页签 38 |

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

> **备选方案（vben 浅炭暗色）**：大底 `#14161A`（`hsl(220 13.06% 9%)`）+ 表面 `#1C1E23`（`hsl(222.34 10.43% 12.27%)`）。若观感偏好"更柔和不近黑"，整体切换只需替换上表 L0/L1 与浮层三个值 + 文字基色改 `#F2F2F2`，三端各改一处。

### 2.4 圆角

| 层级 | 值 | 说明 |
|---|---|---|
| 控件（按钮/输入/选择） | **8px** | vben 基准 `--radius: 0.5rem`；react 端 `radius: "6"→"8"` |
| 卡片/抽屉/弹窗 | **12px** | antd `borderRadiusLG`；Tailwind 语境 = `rounded-xl` |
| 登录/注册等认证面板 | **24px** | vben 基准 `rounded-3xl`（独立大面板，区别于页内卡片） |
| 胶囊 | 999px | Tag、开关类 |

### 2.5 字体

```
-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue',
Arial, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif
```

- 字号：控件 14 / 表格 13 / 卡片标题 16 / 页面标题 18；rem 基准 16。
- 字重：正文 400、标题与强调 600（vue-element 现全局 500，迁移时对齐本表；其自托管 Noto Sans SC 可保留作为 CJK 渲染增强，但字号字重须遵循本表）。
- 表格数字列建议 `font-variant-numeric: tabular-nums`。

### 2.6 布局尺寸

| 项 | 值 | 三端现状 |
|---|---|---|
| 侧栏宽度 | **224px** | 三端一致 ✓ |
| 侧栏折叠 | **48px** | vben 48 ✓；react 60、ele 54 → 迁移 |
| 顶栏高度 | **50px** | vben 50 ✓、ele 50 ✓；react 56 → 迁移 |
| 页签栏高度 | **38px** | 三端一致 ✓（chrome 形态） |
| 内容边距 | **16px** | 由各端页面容器统一施加；ele `app-container` 15px → 迁移 |

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
- 唯一开放项：若最终采纳 2.3 的 react 系暗色（而非 vben 浅炭），改 `packages/@core/base/design/src/design-tokens/dark.css` 中 default 主题的 `--background` / `--background-deep` / `--popover` / `--foreground` 四个值。

### 3.2 react（antd 6）—— 映射机制与迁移清单

机制：preferences store（`src/core/preferences/store`）→ `useThemeConfig.ts` 组装 ConfigProvider token（`darkAlgorithm` + `darkThemeTokens` + `darkThemeComponents`）；自定义 CSS 一律引用 antd 生成的 `--ant-*` 变量，禁止裸色值。

| 本规范 | 落点文件 | 现值 → 目标 |
|---|---|---|
| 主色 2.1 | `src/core/preferences/config/default.ts` | `colorPrimary: "#3B82F6"` → `"hsl(212 100% 45%)"` |
| 圆角 2.4 | 同上 + `useThemeConfig.ts` | `radius: "6"` → `"8"`（`borderRadiusLG: 12` 保持） |
| 暗色 2.3 | `src/core/preferences/config/darkTheme.ts` | 色板已即权威值 ✓；主色 α 三处随主色联动：`rowHoverBg/rowSelectedBg/activeShadow` 的 `rgba(59,130,246,…)` → `rgba(0,107,230,…)`，注释中 `accentBlue` 同步 |
| 顶栏 50 | `src/layouts/MainLayout/index.tsx`（内联 56px） | 56 → 50 |
| 折叠 48 | `src/layouts/MainLayout/components/SiderMenu/index.tsx` | 60 → 48 |

注意：主色三处同步（token、暗色 α、文档），且改完要清 localStorage 旧偏好验证首屏；`localStorage` 残留旧 `#3B82F6` 属预期现象。

### 3.3 vue-element（Element Plus + vxe-table）—— 映射机制与迁移清单

机制：preferences（`src/core/preferences`，schema 与 vben 同构）→ `update-css-variables.ts` 运行时注入 `--el-color-primary(-light-N/-dark-2)` 全阶梯 + `--radius`；SCSS（`src/styles/vendors/_element-plus.scss`）为编译期兜底，两处值必须一致。

| 本规范 | 落点文件 | 现值 → 目标 |
|---|---|---|
| 主色 2.1 | `src/core/preferences/config/default.ts` + `constants.ts` + `_element-plus.scss` | `hsl(220 100% 55%)`/`#165DFF` → `hsl(212 100% 45%)` |
| 语义色 2.1 | 同上 | success/warning/danger → `hsl(144 57% 58%)` / `hsl(42 84% 61%)` / `hsl(348 100% 61%)` |
| 暗色 2.3 | `src/styles/_dark-mode.scss` + `_variables.scss` | 大底 `#0C0E13`→`#0B0F19`、画布/表格 `#14171C`→`#111827`、输入 `#14161a`→`#161F33`、浮层/抽屉 `#1e2026`→`#1C2128`、表头 `#1f2329`→`#1F2937`、边框 `#2e3440`→2.3 的 α 边框方案；菜单/顶栏 `#0D0F14`→`#111827` |
| 浅色 canvas 2.2 | `_element-plus.scss` `$bg-color.page` | `#F7F8FA` → `#F1F3F6` |
| 浅色边框 2.2 | 同上 | `#E5E6EB` → `#E4E4E7` |
| 圆角 2.4 | `_element-plus.scss` + `--radius` | base 8px ✓ 保持 |
| 字重 2.5 | `src/styles/index.scss`（`--el-font-weight-primary: 500`） | 500 → 400 |
| 顶栏/折叠/边距 2.6 | `LayoutNavbar` / `useLayout` / `_layouts.scss` | 顶栏 50 ✓；折叠 54 → 48；`app-container` 15px → 16px |
| vxe 暗色 | `src/styles/vendors/_vxe-table.scss` | 表体 `#14161a`→`#111827`、表头 `#1f2329`→`#1F2937`、hover/斑马纹按 2.3 α 方案换算 |

---

## 4. 组件级视觉惯例（跨端一致）

- **页面容器**：三端各自容器（react `PageContainer` / ele `ProPage` / vben `Page`），但结构统一为：canvas 大底 → 搜索卡片（surface）→ 工具栏 → 表格卡片（surface + 12px 圆角 + 阴影）。
- **表格**：无边框 + 斑马纹可选；表头独立色（浅 `#F0F2F5` 系 / 暗 `#1F2937`）；行 hover 用主色 8% α；行高紧凑（≤40px，ele 30px 现状可保留）。
- **表单/抽屉**：输入控件底与表面同层（暗 `#111827` / 浅白），以 `rgba(255,255,255,.1)` 边框区分；focus 主色边框 + 3px 12% 柔光；抽屉遮罩 `rgba(0,0,0,.6)`，宽度基准 480。
- **认证页（登录/注册）**：画布深底 + 实底表面卡（24px 大圆角、主色柔影）；品牌插画带 vben 同款 float 动效（`translateY 0→-20px→0`，5s 循环，尊重 `prefers-reduced-motion`）。
- **页签栏**：chrome 形态、38px、active 页签用表面色与内容区无缝衔接。
- **图表**：数据色板 = 主色阶梯（-300~-700）+ 语义色；枚举分类名本地化复用既有 i18n 命名空间，不新增同义 key。
- **空态**：文字性空态，不引入插画资源。

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

1. **react 先行**（行为基准端）：主色 + 圆角 + 顶栏/折叠尺寸，一次小 PR，浏览器暗/浅两态验证。
2. **vue-element 跟进**：主色/语义色 + 暗色底统一 + 尺寸，改动集中在 4 个样式/配置文件。
3. **vben 收尾**：按最终决议决定是否把暗色 default 主题对齐 react 系 4 个值。

> 维护记录：2026-09-08 首版定稿（基准取 vben 视觉语言 + react 暗色中性色定稿）；2026-09-13 勘误 2.1 表 success/warning ≈HEX，并补记 vue-element 端收尾迁移（暗色文字层次/抽屉输入同层化/Tag 与图表色板对齐 react/浅色 Arco 灰清除）。
