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
**危险实底按钮的沿革（2026-10-15 定稿）**：react/vben 曾按对比度把亮色 danger 实底按钮前景改成暗墨
（`#0B0F19` on `#FF3860` 5.46:1），用户实测裁决回退——暗墨坐在红色实底上观感刺眼，**亮/暗两态一律回
各 UI 库原生白字**，3.51:1 的数值取舍让位观感（按钮文案是短动作词，不构成阅读障碍）。**同日第二批：
徽标数字一并回退**——react（antd）/ele（EP）删除浅色徽标暗墨规则，恢复库原生白字；vben 本就无此规则。
暗墨（`--gowind-solid-ink`）只剩开关轨道这类**非按钮**小字；success/warning 实底不受本条影响（白字在那两档上 1.87/1.74 真不可读）。

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

> **结案一条旧开放项（2026-10-17）：暗色实心主按钮保持派生档 `#035EC7`，不钉回品牌 `#006BE6`。**
> 实测 react 暗色 `/permission/roles`：`.ant-btn-primary` 与 `.ant-pagination-item-active` 的 bg 都是
> `rgb(3,94,199)` = `hsl(212 97% 40%)`，即 `darkAlgorithm` 从品牌 `hsl(212 100% 45%)` 派生下来的一档
> （**不是仓里硬编码**，全仓搜 `035EC7` 只有注释）。算给它看：白字 6.12:1 → 钉品牌后 4.95:1，
> 分页那处 `#F8FAFC` 文字 5.85 → 4.73（仍过 AA 但贴线）；填充对 L1 `#111827` 反倒从 2.90 升到 3.59。
> 决定性理由是上面第 1 条已经确立的分工：**实底吃派生深档、文字另走反解提亮档**
> （`ThemeProvider/index.tsx:71-77` 把 `colorLink` 混到 `#4D97EE` 才有 5.89:1，`semantic-text.css:149` 同理）。
> 只钉按钮 + 分页两处，等于在同一套暗色里开两条"这里不吃派生色"的例外。
> 若日后确有"品牌色必须原样出现在暗色"的需求，**正确做法是抬暗色的 `colorPrimary` 输入本身**
> （明度 45%→50%，让派生填充正好落回品牌档），一处改动全暗色一致，代价是链接/选中/菜单/波纹要重测一批。

### 2.2 中性色 · 浅色模式

> **落地状态按端不同**（2026-09-25 复核，2026-09-29 补文字档一层，**2026-10-14 再订**）：vue-element ✓（页面画布**不是**由 `_element-plus.scss:34` 的 Sass map `$bg-color: ("page": #F1F3F6)` 落地的——它到不了运行时，实测 `--el-bg-color-page` 读回出厂值 `#F2F3F5`；生效档在该文件末尾的 `html:not(.dark)` 块里带 `!important`，同块补上浅色文字-主/常规两档 `#1F2937`/`#374151`；边框 `#E4E4E7` 那条 **2026-10-17 复测判定为从未落地**——`_element-plus.scss:55` 的 `$table: ("border-color": #E4E4E7)` 属于文件顶部那张 `@forward ... with (...)` map，而本端 EP 样式按**预编译 CSS** 引入（`bootstrap.ts:4` 的 `element-plus/dist/index.css` + unplugin 追加的单组件 css），map 值根本不参与编译，实测 `--el-border-color-lighter` 仍是出厂 `#EBEEF5`、预编译 `el-table.css` 里写的是 `--el-table-border-color: var(--el-border-color-lighter)`；卡片描边现由本端自建 `--gowind-card-border` 承担（见本节末"同一档在 vue-element 的落点"），**表格线仍差那一档，属待决**；原记行号 `:31`/`:52` 均已漂）、vue-vben ✓（`framework/base/design/design-tokens/default.css:10`
> `--background-deep: 216 20.11% 95.47%` = `#F1F3F6`；浅色 `--foreground` 本日起为 `215 28% 17%` = `#1F2937`，原先的 shadcn 出厂 `#323639` 属漏迁）、**react ✓**（2026-10-16 起）——此前 react
> 只有 `core/preferences/config/darkTheme.ts` 一套 token，**浅色全走 antd `defaultAlgorithm` 派生值**
> （实测画布 `#F5F5F5`、卡片边框 `#F0F0F0`、卡片投影 `none`），与本表不严格相等；现已新增
> `core/preferences/config/lightTheme.ts`（画布/容器/边框/大圆角）+ `styles/global.css` 的浅色卡片投影两档，
> 实测 `--ant-color-bg-layout` `#F1F3F6`、`--ant-color-border-secondary` `#E4E4E7`、`--ant-color-split`
> `rgba(10,10,37,.11)`（合成白底即 `#E4E4E7`，1.269）、卡片 `0 1px 2px rgba(0,0,0,.05)`、`borderRadiusLG` 12px。
> **文字档与 placeholder 不进 token 层**：主/常规/次要由 `styles/semantic-text.css` + `colorTextDescription`
> 承担；`colorTextPlaceholder` 刻意留 antd 档，理由见本节末注。
> 例外是认证页：`layouts/UserLayout/UserLayout.style.less` 的 `.light-mode` 大底本日起直写本表 canvas `#F1F3F6`
> （实测点主题按钮后 `backgroundColor = rgb(241,243,246)`，原先是 antd 默认底 `#f5f7fa`）——那是整页大底，不归 antd 派生链管。
> 但**文字档三端已接上同一组值**：react 在 `src/styles/semantic-text.css`（主 `#1F2937` / 常规 `#374151` 两档 2026-10-14 补，加原有次要档）、ele 走 `_element-plus.scss` 的 `--el-text-color-{primary,regular,secondary}`、vben 走 `--foreground`（主）/ `--muted-foreground`（次要）——**vben 结构上没有"常规"档的 token**，正文要么取主档要么取次要档，本轮未新增第四层。

| Token | 值 | ≈HEX | 说明 |
|---|---|---|---|
| canvas（页面画布） | `hsl(216 20.11% 95.47%)` | `#F1F3F6` | 大底，卡片在其上"浮起" |
| surface（卡片/容器） | `#FFFFFF` | — | 卡片、表格容器、顶栏、侧栏 |
| 边框 | `hsl(240 5.9% 90%)` | `#E4E4E7` | 通用边框（≈ zinc-200） |
| 文字-主 | `#1F2937` | — | 标题、重点（gray-800） |
| 文字-常规 | `#374151` | — | 正文（gray-700） |
| 文字-次要 | `#676E7C`（= `hsl(220 9.25% 44.51%)`） | — | 辅助说明、描述表标签。2026-09-29 由 `#6B7280`(gray-500) 提暗一档：旧值在白底 4.83 达标，但落在**实际容器底色**上就不够——antd 表头 `#F5F5F5` 4.43、vxe/EP 表头 `#F0F2F5` 4.31、画布 `#F1F3F6` 4.35；新值同底 4.70 / 4.57 / 4.61 |
| 文字-占位 | `#9CA3AF` | — | placeholder（gray-400），**不参与 4.5 判定**，见下方豁免边界；**仅浅色档**，暗色 2026-10-16 起为 `#6E7681`（§2.3） |

> **占位符档的豁免边界**（2026-09-29 定稿，三端一致）：`#9CA3AF` on 白 = 2.54，但它承载的是"尚未成为内容"的
> 提示语，不是正文，按 4.5 判会把每一张带筛选区的列表页都报成缺陷；antd 同档更浅（`rgba(0,0,0,.25)`）。
> 判据必须窄：**只有坐在表单控件里**（`.el-select/.el-input/.el-textarea/.el-date-editor/.ant-select/.ant-picker/…`）
> 且带 `is-transparent` 空态标记的 `*placeholder` 节点才算这一档。
> `.avatar-placeholder`（姓名首字，85px）、vxe 的"暂无数据"等**是内容文字，不豁免**，实测仍按 4.5 判。
> 选中后的值走 EP 的 `#606266`（去掉 `is-transparent`），同样要量。
>
> **暗色不能照抄浅色的这个值**（2026-10-16 定稿，react 端已落）：占位档的判据是它与**同一控件里已输入值**的分离度，
> 不是它对底色的对比度。浅侧 `#1F2937`/`#9CA3AF` 分离 **5.78 倍**；暗侧值文本 `#F8FAFC`（L1 上 16.96）配同一个
> `#9CA3AF`（6.99）只有 **2.43 倍**——提示语跳到和真内容一个优先级上，用户原话是"暗色模式下太显眼，亮色似乎无事"。
> 暗色改取已有的三级档 `#6E7681`（L1 上 3.86 / 浮层 `#1C2128` 上 3.52，分离度回到 **4.39 倍**），不新增 hex。
> 顺带一条现证更正：上面那句"antd 同档更浅"在 **react 浅色侧才是实测真相**——react 新增的浅色 token 里
> **不含 `colorTextPlaceholder`**（见本节开头"react ✓"那段末句），`:opm/users` 暗/亮两态各读 `::placeholder` 计算色，
> 亮态仍为 `rgba(0,0,0,.25)`（合成 `#BFBFBF`，白底 2.57、分离 6.06 倍），**从未真的画出过 `#9CA3AF`**；
> `#9CA3AF` 在 ele/vben 是写进变量的实值。这不是遗漏：占位档的判据是与已输入值的**分离度**，
> antd 的 6.06 倍比 `#9CA3AF` 的 5.78 倍更"退后"，把浅色钉成 `#9CA3AF` 等于在浅色侧重犯暗色那轮刚修掉的毛病。
>
> **暗色认证页是唯一被拉出豁免的一处**（2026-10-18 已落；**浅色不在其内**）：登录页与 MFA 挑战页的那几个字段没有
> `Form.Item label`，提示语就是字段的唯一可读名称——这时它要的是"读得出"而不是"退后"。所以只在这一个作用域把它
> 提到次要档（暗 `#8B949E`：L1 `#111827` 上 3.86 → **5.77**），全站 token 与 `styles/pro-components-dark.css:238`
> 那条 `!important` 档都不动。react 落点是 `layouts/UserLayout/UserLayout.style.less` 的
> `.user-layout-wrapper:not(.light-mode) .user-form-section …::placeholder`（比那条全局暗色规则多两级类名，
> 靠权重取胜而不是源码顺序）；字段名同时补 `aria-label`——着色救不了屏幕阅读器。
> **浅色那一半被用户判回原值**：理由不是"白底 1.84:1 够读"，而是①这条豁免边界本就是浅色先定的（上面那段：
> react 浅色 token **故意不含** `colorTextPlaceholder`，留 antd 更浅值），②用户这轮只点名暗色，"提一档、不改版式"
> 的范围不该顺带扩到浅色。**教训**：用 `--gowind-secondary-text` 这类**两态都解析**的变量做"局部修复"，
> 会把没被点名的那一态一起改掉——按主题收口的修复，选择器就要钉死在 `[data-theme]` / `.light-mode` 上。
> **代价（暗色）**：与已输入值的分离度 4.39 → **2.94**，靠"登录表单一开始必空 + 聚焦后占位符消失"兜；
> 这类字段一旦补上真 label，占位符就退回三级档。**ele / vben 的认证页同形问题待移植（暗色侧）。**

灰阶统一走 Tailwind gray/slate 族（与暗色同一冷调体系），**次要档是唯一的例外**：`#676E7C` 不在 gray/slate
的任一级上（gray-500 `#6B7280` 在淡底色上差 0.1~0.2，slate-500 `#64748B` 偏蓝过冷），按 2.1 的混色思路取冷灰。
**这条例外要反过来读**：辅助说明/版权/状态副文案一类的次要文字一律取 `--gowind-secondary-text`，组件里不得写
Tailwind 灰阶（`slate-400` 在白底 2.60、在 `#F5F5F5` 上 2.38，`zinc-500` 在暗色卡片 `#111827` 上 3.71，全过不了 4.5）；
坐在**浅底盒子**里的提示文字（如登录页验证码占位）连次要档都不够，按 2.1 取 `--gowind-solid-ink`。
**ele 端另有一层坑**：EP 出厂 CSS 里 `@layer` 计数为 0，`.el-icon { color: var(--color) }` 属 unlayered，
按层叠第一步就压过 Tailwind v4 的 `@layer utilities` ⇒ 写在 EP 组件根上的 `text-*` 颜色工具类**一律不生效**
（现证：同一 `<i class="el-icon">` 裸=白、加 `text-gray-400`=仍白、加 `text-[color:var(--el-text-color-secondary)]`=仍白、
写 `style="--color: rgb(1,2,3)"`= **`rgb(1,2,3)`**）。要改 EP 组件的颜色走它自己的 `--color` 钩子，别写 `text-*`。
浅色投影：卡片静止 `0 1px 2px rgba(0,0,0,.05)`，悬浮态 `0 4px 12px rgba(0,0,0,.08)`，克制不发光。

> **投影只表达层级，不表达"这是块卡片"**（2026-10-16 定稿，react 已落）：同一层级、没有叠放关系的并列容器
> 必须同档。此前 react 的列表页是**搜索卡浮、列表贴地**——搜索卡 `.ant-pro-table-search` 吃 Tailwind
> `shadow-md`（`0 4px 6px -1px rgba(0,0,0,.1), 0 2px 4px -2px rgba(0,0,0,.1)`，α 与 Y 偏移都约是静止档的 2/4 倍），
> 列表 `.ant-table` 一条投影都没有，两块中间还空 24px。读起来像"搜索卡压在列表上面"，而它们并没有这种关系。
> 现在两块在亮色同取静止档、暗色同为 `none`（暗色靠 L1 面 + 8% 白描边分离，黑投影压在 `#0B0F19` 上不显形——
> 原暗色那条 Tailwind `shadow-xl` 属于"写了不生效"的空规则，一并删）。
> 顺带把两处描边越界值收回 token：亮侧原写 Tailwind gray-200 `#E5E7EB`（规范是 zinc-200 `#E4E4E7`）、
> 暗侧原写 5% 白（正是本节上面点名的"卡片与大底糊成一片"那一档），现都走 `--ant-color-border-secondary`。
> 落点 `src/styles/pro-components-dark.css`。
>
> **盒子的载体是宿主卡片，不是表格**（同日第二轮修正，react 已落）：上一版把描边/圆角/投影打在 `.ant-table` 上，
> 而它坐在 `.ant-pro-table > .ant-pro-card > .ant-pro-card-body`（出厂 `padding: 0 24px`）里，
> 于是可见盒子比搜索卡**窄 48px**（亮 1072 vs 1120、暗 1056 vs 1104）——用户口中的"列表 card"其实是那块宿主卡片，
> 它本来就与搜索卡同起点同宽（两态实测 x/width 完全相等）、面也已是 L1/白，只是描边宽度 0：
> **是"没画出来的那块卡片"，不是"这里没有卡片"**。现在 border/radius/box-shadow 全部上移到宿主层，
> `.ant-table` 只保留 antd 自己的网格线。卡片体内缩从 24px 收到 §2.2 的 p-5(20px)，四边同一档——
> ProTable 在宿主 inline 上写死 `padding-block-start: 0`（工具条原本贴卡片顶边），描边上移后必须 `!important`
> 才收得到 20px，工具条自己补的那 8px `padding-top` 同步归零。
> 同轮两处收口：表头分隔线从 Tailwind gray-200 `#E5E7EB` 回 `--ant-color-border-secondary`（亮暗同一条规则），
> 并删掉一条实测被 antd 规则整体压过、从未生效的 `.ant-pro-table .ant-table-tbody > tr > td { border-bottom: 1px solid }`
> ——亮/暗两态的行分隔线一直是 antd 的 colorSplit（亮 `#F0F0F0`、暗 6% 白），不是 currentColor 派生的近白色。

> **同一档在 vue-element 的落点**（2026-10-17 已落，明暗两态 + 骨架态实测）：本端四张卡——`ProSearch/.pro-search`、
> `ProPage/.pro-page__content`，加骨架镜像 `ProPage/TableSkeleton.vue` 的 `.ts-search` / `.ts-table-card`——原先吃
> `--el-box-shadow-light`，运行时实测 `0 0 12px rgba(0,0,0,.12)`：**全向、无 Y 偏移，α 比 §2.2 的悬浮档还重**
> （.12 > .08），静止态读起来"发一圈灰光飘着"，层级语言正好是反的。现统一走本端自建
> `--gowind-card-shadow` / `--gowind-card-border`（定义在 `_element-plus.scss` 的 `:root` 与 `html.dark` 两块里，
> 与 `--gowind-*-text` 同族：base.css 无同名项，不需要 `!important`）——亮 `0 1px 2px rgba(0,0,0,.05)` + `#E4E4E7`、
> 暗 `0 1px 2px rgba(0,0,0,.36)` + 8% 白。三条支撑：
> ① **不复用 `--el-box-shadow-light`**：改完之后它剩下的使用者**并非"全是浮层"——本条原稿把聊天页和 `.el-card`
> 一起算成浮层，是错的**（2026-10-18 纠正）。逐处核过（读的是选择器与所在规则的上下文，不是量像素）：真浮层五处保持那圈 12px 晕不动
> （`.el-select-dropdown` 与 `.el-popper.is-light` `_element-plus.scss:380`、`.el-notification` `:422`、
> `NoticeDropdown/index.vue:165`、
> `LayoutTagsView.vue:1128` 右键菜单、`LeftLayout.vue:118` hover 展开的侧栏覆盖层）；**页面卡两处**——
> `pages/app/ai/chat/index.vue` 的 `.chat-card` 与 `_element-plus.scss:282` 的 `.el-card`。
> **`.chat-card` 已于 2026-10-18 改走 `--gowind-card-shadow` / `--gowind-card-border`**（明暗两态实测：
> 亮 `0 1px 2px .05` + `#E4E4E7`、暗 `0 1px 2px .36` + 8% 白）。它原先暗色实测是 `0 0 12px rgba(0,0,0,.72)`
> （EP 暗色把 `--el-box-shadow-light` 覆成 `#000000b8`），正是 ②里 `.el-card` 那一版点名"会在窄间距卡片间叠出黑缝"
> 的同一档，而会话列表卡与对话卡之间正是窄间距。**`.el-card` 亮色仍是那圈 12px 晕**（暗色已被 `_dark-mode.scss:110`
> 覆成静止档 ⇒ 同一组件两态不同档），属已知待决、本轮未批不动。
> **另记一条工序教训**：react 的聊天卡在 2026-10-17 就由 `b903f8515` 落到静止档，而本端同一轮只覆盖了四张 Pro 卡，
> 于是"文档说聊天页是浮层、react 代码把它当静止卡"这对矛盾当天就产生了、没人发现。**判"某处属于哪一档"要看那个元素
> 是不是页面容器，不能看它吃的是哪个变量**——按变量名归类，等于把"谁还在用旧值"当成"谁该用旧值"。
> ② **暗色那一档不是新发明**：`_dark-mode.scss:110` 的 `.el-card` 早已采用同一条值，注释原话是 EP 暗色
> `--el-box-shadow` 为 72% 黑晕、会在窄间距卡片间叠出"黑缝"——那一版已经判过它不对，只是 Pro 卡没跟。
> ③ **描边抬一档**：`--el-border-color-lighter` 亮 `#EBEEF5`、暗 6% 白，各比权威淡一档，暗侧那 6% 正落在
> §2.3 点名的"卡片与大底糊成一片"那一档。
> **暗色两端的档不同是有意的**：react 暗色靠 L1 面 + 8% 白描边分离、投影 `none`；ele 沿用本端 `.el-card`
> 那一版定稿的黑 α 轻投影（同一端内 `.el-card` 与 Pro 卡必须同档）。描边抬到 8% 白之后两者并存，
> 若下一轮实测判定暗色投影在 ele 也不显形，可一并收成 `none`——**待决，本轮不动**。
> 另两条待决：`.el-card`（EP 组件卡）亮侧仍走 12px 晕、暗侧走静止档，同一组件两态不同档；
> 骨架卡内部线 `.ts-thead` 与真表格线都还是 `#EBEEF5`（同一条 EP 变量），要抬到权威 `#E4E4E7` 得另做一轮。

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
  没有任何渲染文本取该值，真要用进正文得先提到次要档）、**placeholder `#6E7681`**（= 三级档，2026-10-16 由 `#9CA3AF` 下调；
  判据是它与自己控件里**已输入值**的分离度而非它对底色的对比度，见 2.2 末注的暗色补充；
  **例外**：字段没有真 label、占位符就是字段名时（认证页）上抬到次要档 `#8B949E`，判据与代价见 2.2「豁免边界」末段）；
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

> **卡片那一档的三端落点（2026-10-17 实测）**：react `borderRadiusLG: 12` → 搜索卡/列表卡实测 12；
> vben `.vxe-grid` 实测 12（控件走 `--radius: .5rem` = 8）；**ele 原先是 8**，与它自己的
> `.el-input__wrapper`（8）同档——卡片不比装的内容更圆，这是"显方"的真因，已抬到 12
> （`ProSearch/index.vue`、`ProPage/index.vue` 的 `__content`）。
> **骨架卡是真卡的镜像**：`ProPage/TableSkeleton.vue` 的 `.ts-search` / `.ts-table-card` 必须与真卡同值，
> 否则骨架→真实切换时圆角与内缩都会跳（本轮顺带把它那条漏改的 `padding: 16px 20px` 一起归到 20）。
> 内层表格面按 `proTableConfig.ts:13` 的 6px 保留，**外 > 内**这条阶梯不能反。
> 暗色 `_dark-mode.scss` 里那两条 `border-radius: 8px !important` 只作用于抽屉/弹窗内的
> `.el-input__wrapper` / `.el-textarea__inner`（`:340`、`:374`），不涉及卡片——所以卡片圆角没有暗色副本要跟。

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
| 侧栏宽度 | **224px** | vben `framework/core/preferences/config.ts:73`；ele `src/core/preferences/config/default.ts:75`（消费点 `src/layouts/LeftLayout.vue:93`）；react `src/layouts/MainLayout/components/SiderMenu/index.tsx:115` 的 224 兜底 |
| 侧栏折叠 | **48px** | react 同文件 `:115`（`isCollapsed ? 48 : …`）；ele `src/layouts/LeftLayout.vue:57` 与 `src/layouts/MixLayout.vue:123` 的 `SIDEBAR_COLLAPSED_WIDTH = 48`；vben `layout-sidebar.vue:88` 的 `collapseWidth: 48` ✓ |
| 顶栏高度 | **50px** | react `src/layouts/MainLayout/index.tsx:301`；ele `src/styles/_variables.scss:16` `$navbar-height: 50px`；vben 50 ✓ |
| 页签栏高度 | **38px** | vben `config.ts:78`；ele `core/preferences/config/default.ts:80`；react `core/preferences/config/default.ts:80`（chrome 形态） |
| 内容边距 | **16px** | 由各端页面容器统一施加；ele `src/styles/_layouts.scss:9-10` `.app-container { padding: 16px }` |
| 表格 → 分页器 | **16px** | react `src/styles/pro-components-dark.css` 的 `.ant-pro-table .ant-pagination { margin: 16px 0 0 }`（2026-10-16 起；**antd 自带的 `margin: 16px 0` 在本项目落不了地**，见下方分页器条注）；ele `ProPagination/index.vue` 的 `padding: 16px 0 0`（暗色在 `_dark-mode.scss` 有一条同名 `!important` 覆写，两处必须一起改）；vben `framework/effects/plugins/vxe-table/style.css` 新增 `.vxe-grid--pager-wrapper { padding-top: 16px }`。**分页器到卡片底边不另写一份**：由该卡自己的内缩唯一提供（react/ele 20px、vben 8px 密度的 `p-2`），所以"上下对称"从来不是目标——上 16 是同级块间距、下 20 是卡片内缩 |
| 搜索卡内缩 | **20px 四边，且只有一个来源** | react `src/styles/pro-components-dark.css`：`.ant-pro-table-search { padding: 20px }` 配上把 pro-components 自带的查询表单 `padding: 24px` 归零（2026-10-17 起，机制见下方"搜索卡打薄"条）；ele `ProSearch/index.vue` 的 `.pro-search { padding: 20px }`（EP 的 `ElForm` 自身不带 padding，本端只有一个内缩来源、无需归零）+ `ProPage/index.vue` 的 `.pro-page__content { padding: 20px }`；**vben 没有独立搜索卡**（搜索表单在 vxe-grid 的 `#form` 插槽里，与表格同处一张卡），这一档对它只意味着"内容与表格文字同一条竖线"，见同条注 |
| 搜索卡 → 列表卡间距 | **16px**（ele 已知分叉 12px，见右） | react 同文件的 `.ant-pro-table .ant-pro-table-search { margin-bottom: 16px }`；vben 结构上不存在这一对卡片（卡内表单块用 `pb-4` = 16px，本来就同值）；**ele `ProPage/index.vue` 现为 `gap: 12px`**——2026-10-17 用户实测反馈"16px 让两块离得太远、压缩列表数据显示空间"，先收到 8px 又判"两块贴一块"，落 12px。**这一档 ele 目前不满足本表**：是否把 12px 反向同步到 react/vben（或把 ele 收回 16px）属待决，改之前先修本行 |

> **结案一条：`sidebar.collapsedShowTitle` 在 react/ele 不实现，48px 这条宽度也不改（2026-10-17 决定）**。
> 先纠正一句早前的误记：**vben 并没有为这个开关加宽轨道**——它的折叠轨仍是 48
> （`framework/ui-kit/layout-ui/components/layout-sidebar.vue:88` `collapseWidth: 48`），
> 标题是靠把菜单项**竖排**塞进 48 的：`menu-ui/components/menu.vue:734-755`
> （`.is-collapse-show-title { padding: 32px 0 }` + `__name` 12px 放在图标下），
> 代价是单项高度从 ~40px 涨到 ~100px、一屏可见项数腰斩。
> react/ele 这边是几何不可能：实测折叠态 antd 给 `.ant-menu-title-content` 的宽度是 **0**
> （48 − 左右 pad 后不够放图标以外的任何东西），所以开关开了与没开一样。
> 三条出路的代价都排在收益后面：竖排（要覆盖 antd 折叠菜单的硬约束，升级易坏 + 可见项腰斩）、
> 加宽到 ~64（要把本条 48 拆成两档且三端跟）。**决定：48 保持原样，不引入竖排。**
> 留在面板上那个在 48px 下永远不生效的开关属于已知瑕疵，**"在 react/ele 偏好面板里隐藏/置灰该项"这一条
> 尚未获批、未动代码**——下一轮若要处理，那是唯一认可的修法。

> **搜索卡打薄：内缩只能有一条来源**（2026-10-17 定稿；react 先落，同日跟到 ele，vben 按结构差异只跟"内容对齐"这一半，三端均已实测）。
> 现象是"搜索栏和列表卡之间太空、搜索栏自己太厚"。实测拆开两处账：
>
> - **厚度**：pro-components 给查询表单写了 `.ant-pro-query-filter.ant-pro-query-filter { padding: 24px }`
>   （双类提权），与我们的卡 padding 叠成上 20+24 / 下 24+4；单行搜索真正的内容只有 44px，卡高却到 117px
>   （62% 是空白）。归零后 85px（暗色与亮色在 `/permission/menus` 同值），双行展开的 `/permission/roles` 141px。
> - **内容没对齐**：上一轮把两块卡片的**边缘**对齐后，边缘齐、内容仍劈叉——搜索标签落在 45px，
>   而列表卡工具条与表头单元格在 21px。归零 24px 后三处一并变成 21px（含 1px 描边）。
>
> 那条归零规则的**选择器必须同样写双类**，这与"构建期 CSS vs 运行时注入 `<style>`"的顺序无关：
> 实测在页面末尾追加单类 `.ant-pro-query-filter{padding:33px}` 仍压不过 pro 的双类（computed 依旧 0px）——
> 输的是特异度（0,1,0 < 0,2,0）。凡覆盖 pro/antd 自带样式，先读它自己的选择器有几个类，别照抄单类。
>
> **另两端跟法（2026-10-17 实测）**：
>
> - **ele**（`/permission/roles`，明暗两态同值）：改前搜索字段在 21px、列表工具条/表头在 13px——
>   与 react 同一类毛病（只是方向相反：这里浅的是列表侧）。`ProSearch` 16px 20px→20px、
>   `ProPage__content` 12px→20px、`ProPage` gap 12px→16px 之后，字段 / 工具条 / 表头三处一起落到 21px。
>   EP 的 `ElForm` 不带自己的 padding，所以本端没有"第二条内缩"要归零。
> - **vben**：搜索表单不在单独的卡里（vxe-grid 的 `#form` 插槽与表格同属一张卡），所以"两张卡对齐"这一半
>   无对象可跟；跟的是同一句话的另一半——**内容落在同一条竖线**。网格卡只有 `p-2`(8px) 一层内缩，
>   而表头/单元格文字额外吃 `--vxe-ui-table-cell-padding-small`(6px)（`size:'small'`，
>   `src/adapter/vxe-table.ts:33`），实测表单在 8px、单元格文字在 14px；给表单容器补 `px-1.5`(6px) 后
>   两者同到 224（页面绝对 x）。**vben 的密度没有跟着改成 20px**：它整站是 8px 尺度的紧凑卡，
>   把 react/ele 的卡片内缩搬过去等于重排它的信息密度，那一档要单独定夺。

> **ele 那两个 SCSS 常量别当布局尺寸读**：`src/styles/_variables.scss:14-15` 的
> `$sidebar-width: 210px` / `$sidebar-width-collapsed: 54px` 只服务 logo 与菜单项图标对齐；
> 侧栏真正的展开/折叠宽度是上表的 224 与 48。
>
> **沿革（退役旧值）**：react 顶栏 56 / 折叠 60、ele 折叠 54、ele `app-container` 15px——这些是
> 2026-09 之前的现值，迁移已完成，不要再按"待办"读。
>
> **分页器那条为什么必须自己写（2026-10-16 现证，react）**：antd 的表格分页默认间距在
> `es/table/style/pagination.js:13`，选择器是 `:where(.css-hash).ant-table-wrapper .ant-table-pagination`
> （`margin: 16px 0`）。本项目开 `cssVar` 后，`.ant-table-wrapper` 上挂的是 `ant-table-css-var css-var-*`，
> **没有那个 hash 类**（实测祖先链），`:where(.hash)` 前缀整条落空 ⇒ 分页器实际吃 antd 基础重置
> `:where(.hash).ant-pagination { margin: 0 }`，上下 margin 实测都是 `0px`。
> 于是 react 端原先那条 `margin-top: 0 !important`（挂在"分页器圆角（如果有）"的注释下，圆角本身也是空操作——
> 底色早被设成 transparent）看着像"它把 16px 改坏了"，实际是**没有默认值可恢复**，删掉它间距仍是 0。
> 这条机制对任何"以为 antd 会给"的间距/尺寸都成立：在 cssVar 模式下，凡选择器以
> `:where(.hash).<父类>` 起头的默认样式，都要先实测祖先元素上到底有没有 hash 类，别按"框架自带"推定。
>
> **"表格 → 分页器 16px" 的另两端（2026-10-16 记为同步债，2026-10-17 已跟完并实测）**：
> 规范值先落 react，另两端当时的现状是 **ele 上 12px / 下 24px**（`ProPagination/index.vue` 的
> `padding: 12px 0` 与容器 12px 各写一份）、**vben 上 0px / 下 8px**（网格卡 `p-2 pt-0`，
> `style.css` 只改 `.vxe-pager` 颜色不管间距）。跟法与两处必须一起动的暗雷：
>
> - **ele 的间距有三个来源，其中一个在 JS 里**：`ProTable/index.vue` 的高度计算写死
>   `pagerEl.offsetHeight + 20 // 表格与分页器之间的间距`，而 `offsetHeight` 本来就含分页器自己的
>   padding——于是 CSS 的 16px 与 JS 的 20px 叠成实测 36px 的空档。改法是 JS 只减 `offsetHeight`，
>   间距唯一由 `.pro-pagination` 的 `padding-top` 提供（实测盒到盒 0、文字到表格 16）。
> - **ele 暗色有一条同名 `!important`**：`_dark-mode.scss` 的 `.pro-pagination, .pagination-container`
>   块里重复声明了这份 padding，只改 scoped 组件会在亮色生效、暗色回退，两处必须同步（该块想画的
>   `border-top-color` 至今仍是空规则——全仓没给它 border-width/style，真表格没有分隔线）。
> - **vben 的 `.vxe-grid--pager-wrapper` 是 vxe 只出 class、不出样式**：vxe 的 `grid.scss` 没有对应规则，
>   `.vxe-pager` 也只管高度，所以补 `padding-top: 16px` 不会与框架值打架（改前实测上 0 / 下 8）。
> - 顺带一条口径纠正：本节旧版写过"分页器下方 16px 由卡片 body 的 padding-bottom 提供，上下对称"——
>   卡片统一成 20px 内缩后那句的"对称"已不成立，也不该成立：**上方 16 是同级块之间的间距，
>   下方 20 是卡片内缩，两个语义各走各的档**，把下方也写成 16 等于又开第二条内缩来源。

### 2.7 动效

| 项 | 规范 |
|---|---|
| 页面切换过渡 | `fade-slide`，进入 200–300ms / 离开 ≤200ms，ease-in-out |
| 路由进度条 | 顶部细进度条（react 已有 nprogress） |
| 内容区 loading | 骨架屏优先于 spinner |
| 侧栏折叠/浮层进出 | 150–250ms |
| 无障碍 | 全局动效开关（preferences.transition.enable）必须生效；有条件时尊重 `prefers-reduced-motion` |

> **react 端路由过渡的落点（2026-10-16 起）**：`layouts/MainLayout/index.tsx` 用带 key 的 `.page-transition`
> 包裹 `<Outlet/>`，按 `preferences.transition.name` 挂 `.transition-{fade,fade-down,fade-slide,fade-up}`，
> 具体值在 `styles/global.css`（进入 240ms `ease-in-out`、位移 8px、**只做起不做落**——ProLayout 的 Outlet
> 没有可挂的退出相位；`enable=false` 与 `prefers-reduced-motion` 都走 `animation: none`）。
> 填充档用 `backwards` 不用 `both`：`both` 会把终帧的 `transform` 永久留在元素上，非 none 的 transform
> 让内容包裹层长期变成 containing block，页内 `position: fixed` 元素会被改挂到它下面（实测改回
> `backwards` 后动画结束 `transform: none`）。
> 这之前 `preferences.transition.{enable,name}` 在路由层从未被消费（只有错误页 `Fallback` 读它），
> 偏好面板里的四档选择器是个显示"已生效"的死开关。入场位移取 8px 而非偏好面板预览动画的 20px：
> 那个尺度属于 36px 小方块，搬到整屏内容上会读成跳帧。

> **侧栏折叠：一条时钟，而不是四条**（2026-10-16 定稿，react 已落）。"生硬"不是曲线选错，而是同一次折叠被拆成
> 互不同步的多个时钟——折叠前实测（`/opm/users` 逐元素读 computed transition）：
>
> | 元素 | 属性 | 原时长 | 原缓动 |
> |---|---|---|---|
> | 侧栏容器 | `width` | 300ms | `cubic-bezier(.4,0,.2,1)` |
> | `.ant-menu` UL | `width` | 300ms | `cubic-bezier(.2,0,0,1)`（antd "Magic cubic"，与上一条复合成双缓动） |
> | 菜单项 / 分组标题 | `padding` | **100ms** | `cubic-bezier(.215,.61,.355,1)`（`antd/es/menu/style/vertical.js:109`） |
> | 菜单项 | `background-color`/`border-color` | 300ms | ease |
> | 标签 `.ant-menu-title-content` | `opacity`/`margin` | 300ms | `cubic-bezier(.645,.045,.355,1)` |
> | 图标 | `font-size` | 200ms | — |
>
> 于是几何（padding）100ms 就跑完、面板宽度还要再走 200ms，标签更滞后——读起来像"先弹一下再慢慢追"。
> 现在参与折叠的属性（容器 `width`、Logo 区 `padding`、站名 `opacity`、菜单项 `padding`、标签 `opacity`/`margin`、
> 图标 `font-size`/`margin`、箭头 `transform`）统一 **240ms + `cubic-bezier(.4,0,.2,1)`**（§2.7 允许档 150–250ms 内），
> UL 的 `width` 过渡直接关掉（面板宽度由容器决定，第二层只会叠加缓动）；
> hover 反馈（`background-color`/`border-color`/`color`）保留 antd 的 200ms，它不属于折叠。
> 落点：`layouts/MainLayout/components/SiderMenu/index.tsx`（容器与 Logo 区 inline）+ `styles/pro-layout-overrides.css`
> （菜单侧，作用域收在 `.app-sider` 内）。两条必须一起改的坑：
> ① **`padding` 不能从菜单项的过渡里摘掉**——折叠态图标的定位就靠这个值（48px 轨道上实测 `padding-left` 11.67px）；
> ② 站名原本随折叠**条件卸载**（面板还有 220px 宽时一帧消失，是同一类瞬时跳变），现改为常驻、
> 靠 flex 收缩 + 同档淡出，Logo 区 `padding` 也从 `0 16px → 0`（图标靠 `justify-content: center` 归位）
> 改成 `0 16px → 0 8px`（48 − 32 = 16，图标落在轨道正中）并让这条 padding 一起上时钟。
> 全局动效开关（`preferences.transition.enable`）与 `prefers-reduced-motion` 都把这条折叠降到 `none`；
> `.transition-disabled` 只挂在内容区 `.page-transition` 上，侧栏是它的兄弟节点，所以开关状态由 SiderMenu 自己落到 wrapper（`.sider-motion-off`）。
>
> **帧级复验（2026-10-18，`/dashboard/analytics` 暗色，逐帧 rAF 采样 + 每一次 `transitionend` 的 `elapsedTime` 逐元素登记）**：
> 折叠与展开各一次，取每个 (元素, 属性) 的最大 elapsed —— 容器 `width` **240ms**、Logo 区与分组标题
> `padding-left`/`padding-right` **240ms**（×18 = 9 个菜单项 × 两向）、站名与标签 `opacity`/`margin-left` **240ms**、
> 图标 `font-size` **240ms**、箭头 `opacity` **240ms**。**没有一个属性落在 100ms 或 300ms 上**，上表那四条时钟已合成一条。
> 宽度逐帧序列 48 → 74.1 → 195.3 → 221.1 → 224（约 148ms 起、340ms 落定，单调、无平台段、无回弹）。
> 首轮另见 `transform` 300ms 与 `height` 200ms 各一次，复跑两轮均为 0 次 ⇒ 不属折叠参与者（submenu 箭头/子容器挂载的一次性动画），不处理。
>
> 另：点击**展开分组**（submenu）本身没有横向位移——只有箭头 `transform` 与子容器 height 动画，子项挂载即
> 是最终的 `padding-left: 48px`。所以"菜单项向右展开"的观感来自侧栏折叠，不是分组展开。

---

## 3. 三端落地映射

### 3.1 vue-vben —— 参照实现（视觉基准）

机制：`preferences`（`framework/core/preferences`）→ `update-css-variables.ts` 写 `--primary` 等 HSL 三元组 + `html.dark`/`data-theme` → `useAntdDesignTokens()` / `useElementPlusDesignTokens()` 派生组件库 token。

- 本项目 override 仅 `app.name` / `app.accessMode` / `app.defaultAvatar`（`src/preferences.ts`，2026-10-17 复核；
  头像是为了与 react/ele 用同一张 `/default-avatar.png`，与视觉规范无关），默认主题即本规范基准，**无强制迁移项**。
- 2.1 的语义色文字档 + 2.2/2.3 的次要档落点：`framework/styles/antd/index.css` 末尾的 `--gowind-*-text` 段
  （变量由 `hsl(var(--primary|success|warning|destructive))` 混色，跟随主题预设；规则带 `!important`，理由见 2.1 第 2 条）。
  **这一层是纯 CSS 且经 postcss 处理，注释只能写 `/* */`，写 `//` 会在 dev 直接 500（`Unknown word …`）。**
- 曾经的"唯一开放项"已关闭：2.3 的 react 系暗色已被采纳——`framework/base/design/design-tokens/dark.css:4-11`
  的 `--background` / `--background-deep` / `--popover` / `--foreground` 已是近黑蓝系，该文件头注释（2026-09-08）
  即写明"暗色中性色对齐设计语言规范（docs/design-language.md §2.3，近黑蓝系，与 react 端定稿一致）"。

### 3.2 react（antd 6）—— 映射机制与迁移清单

机制：preferences store（`src/core/preferences/store`）→ `useThemeConfig.ts` 组装 ConfigProvider token（`darkAlgorithm` + `darkThemeTokens` + `darkThemeComponents`）；自定义 CSS 一律引用 antd 生成的 `--ant-*` 变量，禁止裸色值。

| 本规范 | 落点文件:行（2026-10-17 复核，行号随本轮布局改动重测） | 现值（= 规范值） |
|---|---|---|
| 主色 / 语义色 2.1 | `src/core/preferences/config/default.ts:91`（success `:92`、warning `:93`、destructive `:90`） | `colorPrimary: "hsl(212 100% 45%)"` |
| 圆角 2.4 | 同上 `:95`；大圆角在 `config/darkTheme.ts:72` | `radius: "8"`、`borderRadiusLG: 12` |
| 暗色分层 2.3 | `config/darkTheme.ts:41,42,44,46`；折叠弹层另补 `:129` | `colorBgBase/colorBgLayout: '#0B0F19'`、`colorBgContainer: '#111827'`、`colorBgElevated: '#1C2128'`、`Menu.darkPopupBg: '#1C2128'`（antd 出厂 `#001529` 藏青，见 §4「侧边栏菜单交互态」末注） |
| 主色 α（暗色） | `config/darkTheme.ts:91,107,108,109` | `activeShadow …0.12`、`rowHoverBg …0.08`、`rowSelectedBg …0.15`、`rowSelectedHoverBg …0.2`，全为 `rgba(0, 107, 230, …)` |
| 顶栏 50 | `src/layouts/MainLayout/index.tsx:301` | `height: 50` |
| 折叠 48 / 展开 224 | `src/layouts/MainLayout/components/SiderMenu/index.tsx:115` | `isCollapsed ? 48 : (sidebarConfig?.width ?? 224)` |
| 语义色文字档 2.1 + 次要档 2.2 | `src/styles/semantic-text.css`（由 `main.tsx` 引入） | `--gowind-*-text` = `color-mix(...)`，规则一律带 `!important`（2.1 第 2 条）；浅/暗两态各一组变量，深墨类规则只在 `html[data-theme='light']` 下生效 |
| 页签激活态文字 | `src/layouts/MainLayout/components/TabsBar/tabsbar.css` | 浅色走 `--gowind-primary-text`；暗色的 chrome/card 形态保持 `var(--ant-color-text)`（本地规则同样带 `!important`，否则被 2.1 那层的新页签规则盖掉） |

> **一处与 2.3 有意的偏差**：react 不覆写表头色——`darkTheme.ts:105` 注释说明
> `colorHeaderBg` 不是合法 antd Token（原值 `'#1F2937'` 已移除），表头由 `colorBgContainer/colorFillAlter`
> 派生。2.3 表里的"表头 `#1F2937`"对 react 端只是**观感目标**，不是可直接填的 token。

**沿革（退役旧值，别再去代码里找）**：主色 `#3B82F6`、`radius: "6"`、顶栏 56、折叠 60、暗色主色 α 的
`rgba(59,130,246,…)`。改主色仍要三处同步（token、暗色 α、本文档），且 `localStorage` 残留旧偏好属预期现象，
验证首屏前先清一次。

### 3.3 vue-element（Element Plus + vxe-table）—— 映射机制与迁移清单

机制：preferences（`src/core/preferences`，schema 与 vben 同构）→ `update-css-variables.ts` 运行时注入 `--el-color-primary(-light-N/-dark-2)` 全阶梯 + `--radius`；SCSS（`src/styles/vendors/_element-plus.scss`）为编译期兜底，两处值必须一致。

> **上面这半句"编译期兜底"2026-10-17 实测推翻了一半**：本端 EP 样式走**预编译 CSS**（`bootstrap.ts:4` 引
> `element-plus/dist/index.css`，unplugin 再按需追加 `theme-chalk` 的单组件 css），所以 `_element-plus.scss:9`
> 那句 `@forward "…/common/var.scss" with (…)` 里的 `$bg-color` / `$box-shadow` / `$fill-color` / `$table` 四张 map
> **不参与任何编译、到不了运行时**：实测 `--el-box-shadow-light` = 出厂 `0px 0px 12px #0000001f`（map 写的是
> `0 2px 8px .06`）、`--el-fill-color-light` = 出厂 `#f5f7fa`（map 写 `#F2F3F5`）、`--el-bg-color-page` 在该文件
:553 那条 `html:not(.dark) … !important` 补上之前也读回出厂 `#F2F3F5`（map 写 `#F1F3F6`，2026-10-14 实测）。**只有该文件里显式写进 `:root` / `html:not(.dark)` / `html.dark` 的自定义属性才生效**，
> 且因 base.css 追加在本文件之后、要压过它得带 `!important`（自建 `--gowind-*` 名字除外）。改 EP 出厂值前先想清这条，
> 改 map 是空转。

| 本规范 | 落点文件:行（2026-10-17 复核） | 现值（= 规范值） |
|---|---|---|
| 主色 2.1 | `src/core/preferences/config/default.ts:91` + `constants.ts:12` + `styles/vendors/_element-plus.scss:13` | `hsl(212 100% 45%)` / SCSS `#006BE6`（`info` 亦同值，见 `_element-plus.scss:29-31`） |
| 语义色 2.1 | `default.ts:92,93,90`（运行时注入） | success/warning/destructive 三个 HSL 与本表逐字一致 |
| 暗色 2.3 | `src/styles/_dark-mode.scss:33-36,42-45,54,72` | 大底 `#0B0F19`、表面/输入/抽屉 `#111827`、表头 `#1F2937`、浮层 `#1C2128`——旧 Arco 灰（`#0C0E13/#14171C/#14161a/#1e2026/#1f2329/#2e3440/#0D0F14`）已全部清除 |
| 浅色 canvas 2.2 | `_element-plus.scss:553`（`html:not(.dark)` 块，带 `!important`） | `#F1F3F6`。**不是** `:34` 的 Sass map——见上面那条注；浅色**边框** `#E4E4E7` 同理从未由 `:55` 落地，卡片描边现走 `--gowind-card-border`，表格线仍是出厂 `#EBEEF5`（待决） |
| 圆角 2.4 | `_element-plus.scss` + `--radius`（`default.ts:95` 为 `"0.5"` = 8px） | base 8px |
| 字重 2.5 | `src/styles/index.scss:46` | `--el-font-weight-primary: 400` |
| 顶栏 / 折叠 / 边距 2.6 | `_variables.scss:16`（50px）/ `src/layouts/LeftLayout.vue:57`、`src/layouts/MixLayout.vue:123`（48）/ `_layouts.scss:9-10`（16px） | 与 2.6 一致 |
| 搜索卡内缩 / 卡间距 / 表格→分页 2.6（2026-10-17 跟齐 react） | `src/components/Pro/ProSearch/index.vue`（`padding: 20px`）、`Pro/ProPage/index.vue`（`gap: 16px` + `__content { padding: 20px }`）、`Pro/ProPagination/index.vue`（`padding: 16px 0 0`，暗色副本在 `_dark-mode.scss:763` 的 `!important`，两处必须一起改） | 实测字段/工具条/表头同落 21px；间距来源**还有一条藏在 JS**：`Pro/ProTable/index.vue` 的高度计算原先 `offsetHeight + 20` 与 CSS padding 叠成 36px，现只减 `offsetHeight`（详见 §2.6 该条注） |
| vxe 暗色 | `styles/vendors/_vxe-table.scss:180,183,185,187` | 表头 `#1F2937`、表体/布局底 `#111827`、斑马纹 `rgba(255,255,255,.03)` |
| 卡片圆角 2.4（2026-10-17 由 8 抬到 12，明暗两态实测） | `ProSearch/index.vue`、`ProPage/index.vue` 的 `__content`；骨架镜像 `ProPage/TableSkeleton.vue` 的 `.ts-search` / `.ts-table-card` | 12px（控件仍 8、内层表格面按 `proTableConfig.ts:13` 的 6px；实测外 12 > 输入 8 > 按钮 6 > 表格面 6） |
| 卡片静止档投影 + 描边 2.2 / 2.3（2026-10-17 落地，明暗两态 + 骨架态实测） | 变量定义 `_element-plus.scss` 的 `:root`（亮）与 `html.dark`（暗）里那两条 `--gowind-card-*`；使用者同上四张卡（`ProSearch/.pro-search`、`ProPage/.pro-page__content`、`TableSkeleton/.ts-search`、`.ts-table-card`） | 亮 `0 1px 2px rgba(0,0,0,.05)` + `#E4E4E7`、暗 `0 1px 2px rgba(0,0,0,.36)` + `rgba(255,255,255,.08)`。**不再吃 `--el-box-shadow-light`**（出厂 12px 全向晕，α 比悬浮档还重），改后它剩下的 7 处使用者全是浮层；`.el-card` 亮侧仍走那条晕，待决。理由与"暗色两端不同档是有意的"见 §2.2 末注 |
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

- **页面容器**：三端各自容器（react `PageContainer` / ele `ProPage` / vben `Page`），结构统一为：canvas 大底 → 搜索卡片（surface）→ 工具栏 → 表格卡片（surface + 阴影）。**两处口径本轮核准**：① 阴影按 §2.2 只在亮色铺卡片静止档，react 暗色实测 `none`（§2.3 暗色投影须黑 α，白 α 那条已从 `chat-page.css` 清掉）；ele 2026-10-17 起四张卡同取该档、暗色沿用本端 `.el-card` 的黑 α 轻投影——**暗色一端 `none`、一端有轻投影，是两端各自内部一致造成的差异，不是漂移**，理由见 §2.2 末注；② 卡片圆角曾是 ele 离群（它写 8、react/vben 写 12），**2026-10-17 已抬 ele 到 §2.4 的 12px 档并实测明暗两态**，落点与"骨架卡是真卡镜像"这条约束见 §2.4 注。
- **表格**：无边框 + 斑马纹可选；表头独立色（浅 `#F0F2F5` 系 / 暗 `#1F2937`）；行 hover 用主色 8% α；行高紧凑（≤40px，ele 30px 现状可保留）。**斑马纹值表（2026-10-18 定稿，明暗两态实测）**：ele/vben 的 vxe 全局默认开（ele `proTableConfig.ts:12` `stripe: true` 是共享 ProTable 默认值），react 原先全端唯一没条纹的一端，同日起 `ListTable` 加 `striped` **默认开**（`rowClassName` 挂 `ant-table-row-striped`，页面可显式关）。色值：亮 `var(--ant-color-fill-quaternary)`（≈`#FAFAFA`，与 vxe 亮档 `#F8F8F9` 同档，token 优先）；暗 `rgba(255,255,255,.03)` = §3 vxe 暗色斑马纹既有权威值——antd 同名 token 暗色是 `.06`、重一倍，故暗色显式落规范值不走 token。占位行（暂无数据）不吃条纹；固定列（sticky）的半透明条纹会透出压在下层的文字，与行 hover 透字同一处理：容器底色打底 + 条纹色走 `background-image`，hover 时 `background-image: none` 清掉（react 落点 `pro-components-dark.css`）。
- **表格工具栏右侧图标钮（2026-10-08 定稿，react ProTable options 为基准 / ele 已对齐）**：无边框裸图标（hover 才铺 fill 底），颜色 = **主文本色阶**（react `colorText` / ele `--el-text-color-primary`，暗 `#F8FAFC` / 亮 `#1F2937`），不用 secondary——次级灰是为次要文字设计的，做图标唯一内容会显虚。**每枚图标必须带 Tooltip**（刷新/密度/列设置/全屏……）。图标名义 16px 是 antd 口径：antd 图标墨迹占 viewBox ≈90%，EP 图标仅 ≈75%，故 ele 的 EP 图标要放到 **18px** 补偿（等效墨迹 13.5px ≈ antd 16px），vben 若用 lucide 同理注意留白差异。
- **列表操作列（2026-10-21 定稿，react 基准 / ele / vben 三端已全量落地）**：
  - **行内按钮一律「图标 + 文本」的 link 弱化形态**，禁止纯图标（要靠 Tooltip 猜语义、暗色下高亮图标凌乱）也禁止纯文本（缺视觉锚点）。三端形态：react `Button type="link" size="small" icon={<XOutlined />}`（危险钮走 `danger`，不走内联颜色 style）；ele ToolCell `ElButton link size="small"` + `SvgIcon 14`；vben `a-button type="link" :icon="h(LucideX)"` + 文本子节点（vben 全局 `.ant-btn` flex 基座自带 8px 间距）。
  - **按钮数 ≤3 全部行内；>3 行内只留前 3 个，其余进「更多」下拉**——下拉选项同款「图标 + 文字」，危险项红墨（antd Menu `danger` / ele `--el-color-danger` + hover 钉色）。「更多」触发 = 文本「更多」+ 下箭头（react `MoreOutlined` / ele `lucide:chevron-down` / vben `LucideChevronDown`，i18n 键 react `common:button.more` / ele `common.button.more` / vben `ui.button.more`）。
  - **下拉里放不下 Popconfirm**：溢出到「更多」的确认类操作（重启/删除等）改弹窗确认——react `App.useApp().modal.confirm`（带 `okButtonProps:{danger:true}`）、vben `Modal.confirm`（`okType:'danger'`）、ele 下拉项复用页面 `@operate` 侧既有确认流（确认逻辑在页面 handleOperate，不随入参形态变化）。行内 ≤3 时确认方式不变（Popconfirm 原样保留）。
  - **列宽随形、宁大勿小**（换行即失败）：2 钮 ≈150、3 钮 ≈200-220、溢出后（3 钮 + 更多）≈260，文案每多 2 个中文字符 +28。ele/vben 的 vxe 列已有 width 就调 width，用 minWidth 就调 minWidth。
  - **三端的溢出都是组件单点（2026-10-21 组件化，页面只声明描述符）**：ele 共享 ToolCell（`ProTable/cellRendererRegistry.ts`，`col.buttons` 声明 name/label/icon/type/visible/auth，对 visible()+权限码判活后的按钮数做 >3 判定）；react `components/common/TableActions`、vben `components/TableActions.vue` 描述符同形（`key/label/icon/onClick/danger/success/confirm{title,description,okLoading}/tooltip/hidden`，行参数在点击时回传 `onClick(row)`，行级条件用 `hidden` 在计溢出前剔除）。语义对齐、形态随端（ele 随 EP `type`，antd 系用 `danger` prop）。react/vben 新页操作列一律写描述符，不再手写 Button/Popconfirm/Dropdown；样板：react `system/language`（≤3）/ `system/task`（溢出），vben 同两页。vben 行内标签用框架层纯文本键 `common.edit/common.delete` 与 `ui.button.detail/download/more`——**`ui.button.edit/delete/view` 是带 `{moduleName}` 的抽屉标题键，禁止当行内按钮文本**。
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
  - **浅色这一档 react 曾长期写成中性灰（2026-10-18 收口）**：`tabsbar.css` 的 chrome/card 选中底是
    `rgba(128,128,128,.12)`，压白栏合成 **#F0F0F0**——与页面画布 #F1F3F6 只差 **1.008:1**（读出"白栏被抠了个洞"而不是
    "这页被选中"），且与 hover 的 `.08`（#F5F5F5）同色相、只差 α，选中与悬停糊成一片。ele 早已按本条走 `color-mix`
    （`LayoutTagsView.vue:1004`，注释还解释过为什么不能用 `primary-light-N`），react 是唯一偏离端。现浅色改吃
    `color-mix(in srgb, var(--ant-color-primary) 15%, var(--ant-color-bg-container))` → 实测 **#D9E9FB**、对栏
    **1.24**（原 1.14）；描边退回 `transparent`（底色已是标记通道，灰圈压淡蓝底只会再叠一层脏）；card 的字重 500 一并收平
    （暗色那档早已覆成 400，留着会让同一风格两态字重不同，且本条明令禁用字重加粗）。**暗色零改动**：文件下方那组
    `[data-theme='dark']` 的 `!important` 仍胜出，实测 #262D3A + 8% 白描边 + 字重 400 逐值与改前一致。
    **一条要记住的代价**：淡蓝底把主色文字档 `--gowind-primary-text` 从灰底上的 5.00:1 压到 **4.63:1**——仍过 §2.1 的
    4.5 底线，但只剩 0.13 余量 ⇒ **别再往上调淡底的比例**（20% 即破线）；要更强的标记，走文字/图标档，不要走底。
- **默认头像**（2026-09-16 统一）：三端统一使用橘猫插画 `default-avatar.png`（react/ele public 同文件，vben 经 `src/preferences.ts` 覆盖框架默认的 webp——带 Vben 品牌字样已弃用）；用于导航栏当前用户、通知、锁屏等无头像兜底；用户列表/详情的"姓名首字 + 底色"兜底保留（承载身份信息）。
- **侧边栏菜单交互态（2026-09-16 定稿，基准 = react antd Menu）**：悬停 = 中性灰遮罩（浅 `#F5F7FA` 系 / 暗 `rgba(255,255,255,.05~.08)`，8px 圆角）；选中 = 主色实底 + 白字（`--primary-foreground`）+ 8px 圆角药丸，不得用左侧竖条 / 淡色底 / 字重加粗来区分选中（vben 旧"选中与悬停同灰"、ele 旧"inset 蓝条 + light-9 淡底"均废弃）；父级展开链路只做文字/图标提亮，不铺底色。折叠后的弹出子菜单选中态同规则。横向顶部菜单暂不约束。
- **"展开子菜单不铺底色"的落点（2026-10-18 补齐，此前只有 react 违规）**：react 侧栏曾由 `styles/pro-layout-overrides.css` 给
  `[data-theme='dark'] .ant-menu-sub.ant-menu-inline` 铺 `rgba(0,0,0,.22)` + 8px 圆角 + `padding:4px 0`（2026-08-24 暗色色板那一轮写的，
  早于上一条定稿、之后没回头对齐）。实测合成 `#0D131E` vs 侧栏 `#111827` = **1.049:1**——小到起不到分组作用、又足够被眼看成"不一致"；
  且它只命中暗色、只命中展开态（折叠弹层是 `.ant-menu-vertical`，压根没进选择器），三端里 ele/vben 的子菜单都与侧栏同色。已删该规则。
  **收平必须走 token，不能只删 CSS**：antd 出厂 `darkSubMenuItemBg: '#000c17'`（`es/menu/style/index.js:449`）比原规则更黑，
  规则一撤就露出藏青井（实测删规则未改 token 时 `.ant-menu-sub` = `rgb(0,12,23)`）。落点 = `config/darkTheme.ts` 的
  `Menu: { darkSubMenuItemBg, subMenuItemBg }` + `config/lightTheme.ts` 的 `Menu: { subMenuItemBg }`，两态均 `transparent`。
  **两个键不是冗余**：cssVar 模式下它们喂不同变量——`.ant-menu-dark`（侧栏暗色走这个，`SiderMenu/index.tsx` 的 `theme=isDark?'dark':'light'`）
  读 `--ant-menu-dark-sub-menu-item-bg` ← `darkSubMenuItemBg`；`.ant-menu-light` 读 `--ant-menu-sub-menu-item-bg` ← `subMenuItemBg`。
  只写后者时暗色侧栏纹丝不动（实测 `--ant-menu-dark-sub-menu-item-bg` 仍 `#000c17`）。
  同族的 `darkItemBg`/`darkPopupBg` 出厂都是 `#001529`（藏青）。根 UL 靠 `SiderMenu/index.tsx:229` 的 inline `background:'transparent'` 盖住了，
  而**折叠态弹出的子菜单浮层没有这层保护**（2026-10-18 实测确认后已修）：让 Menu 真正进入 `ant-menu-inline-collapsed` 需要
  `sidebar.expandOnHover=false` + `collapsed=true`（自动模式下点底部折叠按钮只改偏好，宽度由 hover 决定——此前"点了没折叠"是这条，不是缺陷），
  此时 portal 里承载弹出的 `.ant-menu-submenu-popup > .ant-menu.ant-menu-sub.ant-menu-vertical` 计算底色 = `rgb(0,21,41)` = 出厂藏青；
  它的圆角 12 与投影 `rgba(0,0,0,.45) 0 6px 16px + rgba(0,0,0,.3) 0 3px 6px` 本就分别合 §2.4 与 §2.3「投影必须用黑」，同层选中项也已是
  `#035EC7` 实底 + `#F8FAFC` 字 ⇒ **露馅的只有浮层容器底一档**。这一档与 §2.3 浮层 `#1C2128` 的差不是可读性（`#F8FAFC` 在其上 17.61 vs 15.38）
  而是**色系归属**：藏青比本项目的暗色蓝黑系更暗、更偏蓝，和抽屉/下拉摆在一起会分出"两套黑"。
  **落点（2026-10-18，react）**= `config/darkTheme.ts` 的 `Menu.darkPopupBg: '#1C2128'` 一行；`darkItemBg` 未动——查过 antd 源码，
  `darkPopupBg → popupBg`、`darkItemBg → itemBg` 是两个键，绘制弹层的那条规则现读
  `:where(…).ant-menu-dark.ant-menu-submenu-popup > .ant-menu { background-color: var(--ant-menu-dark-popup-bg) }`，只由前者驱动。
  仍缺的一环：可见态没有截图——内嵌面板 `visibilityState=hidden` 时那层 popup 停在 `ant-menu-submenu-hidden`（`display:none`），
  合成 hover 与直调 fiber 的 `onMouseEnter` 都唤不开它，所以改前底色取自"antd 用来显示它的那个元素"的计算样式、改后由那条 var 与规则双向对上。
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
- **错误/兜底页（401/403/404/500/offline/coming-soon）**（2026-09-16 定稿）：插画填色一律走 `--fb-*` 插画语义变量（`primary/ink/paper/mist/mist-2/line/navy/navy-deep/skin/skin-light` + 暖色件 `coral/flame/flame-core/star`，2026-10-13 补），SVG 内禁用裸色值。亮色为"纸墨日光"原色；暗色为"月夜"版——装饰件压暗至画布上方一档（mist `#182136` / mist-2 `#22304a` / line `#33405e`）、藏青物件提亮保形（navy `#46547a` / navy-deep `#38456a`）、主色提亮一档（`color-mix(in srgb, 主色 78%, white)`）；暖色四枚按 `paper`/`skin` 先例**两态同值**（`coral #FF6B6B` / `flame #FF9800` / `flame-core #FFEB3B` / `star #FFD700`，暗色块不覆写），暗色是否随"月夜"压暗另属 §5.1 决议。主色锚点接线：react 由 ThemeProvider 写 `--app-color-primary` 到 `<html>`，ele 复用 `--primary-hsl`。**插画里的 `<text>` 属"文字"，取 `--fb-ink`（两态各自取值，正逢其"线稿/数字 = 文字主色"的定义）或 `--gowind-*-text`，不得取 `--fb-navy`/`--fb-coral` 这类物件色**——2026-10-13 实读：`--fb-coral` 作离线页 `×` 在亮色只有 2.55（暗色 6.9），`--fb-navy` 作 Coming Soon 标题在暗色只有 2.56（亮色 9.57），两处都低于 §2.2 的大号/非文字 3.0 档；改墨色后 `×` = 亮 5.6 / 暗 5.94，标题 = 亮 13.46 / 暗 18.3。暖色四枚的两态同值决议（§5.1）不受影响，它们继续只管形状。插画后方垫主色柔光晕（radial-gradient `--fb-glow`，暗色更明显）；进场 fade+上浮 450ms 依次错峰 + 插画 6s 悬浮呼吸，动效尊重 `preferences.transition.enable` 与 `prefers-reduced-motion`。
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
6. **验证门禁**：视觉迁移 PR 必须跑三端 typecheck（react `npm run typecheck` / ele `npx vue-tsc --noEmit` / vben `pnpm run typecheck`），并附暗/浅 × 三端对照截图。

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
2026-10-13 更正 §4「react/ele 已迁移」那句的覆盖面：react `Icon401/IconComingSoon/IconOffline` 与 ele `icon-401.vue` 当时各留了暖色裸 hex（react 14 处、ele 1 处），本次具名化为 `--fb-coral/flame/flame-core/star` 并按 `paper`/`skin` 先例取**两态同值**（暗色块不覆写 ⇒ 亮/暗渲染逐字节不变，"暗色要不要压暗"仍留作 §5.1 决议）。react 侧 `/coming-soon`、`/401`、`/offline` 暗态实读：SVG 节点 `var(--fb-*)` 引用 15/15、14/15、12/13，暖色件命中 10、1、3 枚与源码逐一对应，未解析计数 0（检测器以"插一枚不存在的令牌必读回 `rgb(0, 0, 0)`"验活）；`npm run typecheck` 退出码 0。ele 侧那 1 处同构替换**未过门禁**——本机 `frontend/admin/vue-element` 无 `node_modules`，`vue-tsc` 跑不起来。

2026-10-13 同日第二笔：把 react 端「次要文字写成 Tailwind 灰阶」的六处收进 §2.2 的次要档与 §2.1 的实底暗墨（`AuthLayout:106,120`、`UserLayout:143`、`login:174`、`MainLayout/Footer:9,16`、`common/Loading:67,73,114` 与文字色新增的 `textColor`）。量法：页面内把计算色（Chrome 对 Tailwind v4 灰阶报 `oklch(...)` 而非 hex）转 sRGB 再算 WCAG 比值，读色前先 `transition:none` + 强制回流（后台标签页没有绘制帧，否则读到切换前的旧值）；量具两点验活 `白/黑=21.00`、`#FF3860/白=3.51`，第一版量具漏了 gamma 编码那一步、把 `slate-400` 报成 `#4D5B7C`，正是这两点把它照出来。实测（浅/暗两态，逐元素取实际落底）：`/auth/forgot-password` 品牌副标 2.38→4.70（落 `#F5F5F5`）、表单描述 2.60→5.12（白底）——这两处**原本不合格**；暗色同两处 7.36→6.23、6.82→5.77（原本达标，本次只改归档）；auth 页版权行暗色 3.71→5.77（`zinc-500` 落 `#111827`）、浅色 5.74→5.12；登录页验证码占位文案坐在**浅底盒子**里（暗色 `bg-white`、浅色 `bg-black/[0.03]` 合成 `#F7F7F7`），次要档在浅底上仍只有 2.43/2.60，故取 `--gowind-solid-ink`：2.43→17.88、2.60→19.15。**两处按构造改、未实测**：① `Footer.tsx` 的版权行只在偏好里填了 ICP 才渲染（`config/default.ts:39` 默认 `icp:""`），同处 `hover:text-blue-600` 与本日 §4 那条同因——工具类在 `@layer utilities`、antd 的 `:where(…) a{color:…}` 不属任何 layer，层叠第一步就输，hover 从不生效，换成 `hover:opacity-80`；② `Loading.tsx` 是冷启动全屏遮罩，只在路由包懒加载的那几百毫秒存在，轮询探针没抓到，而本机在跑的 :7788 二进制早于 captcha RPC（`GET /admin/v1/captcha` 实测 404），验证码取不到 ⇒ 走不进登录后的页面。Loading 的数字用**实测到的 CSS 变量值 + 组件自己声明的遮罩 alpha**算（暗色 `rgba(0,0,0,.85)` 叠实测 `#111827`、浅色 `bg-container/95` 叠实测 `#F5F7FA`）：主文案 暗 3.71→6.11、浅 4.93→5.71，副文案 浅 2.53→5.11、暗 7.23→5.97。`npm run typecheck` 退出码 0。**同一条 §2.2 规则在另两端尚未落**：vben 尚有 3 处（`ai/provider/ai-provider-drawer.vue:183` `text-gray-400`、`system/script/index.vue:271` `text-neutral-400`、`:317` `text-neutral-500`），全在登录后的页面上，本机同样量不到，按"react 先行、其余移植"记为待移植；ele 无 Tailwind 灰阶写法（0 处），其硬编码 `#909399` 一类走 `_element-plus.scss` 那条已有档。

2026-10-13 同日第三笔（整页扫描器 + 两处真不合格）：**先纠正上一笔留下的一个假绿面**——那轮写的"整页对比扫描 0 失败"是量具盲区造成的：兜底页文字带 450ms 入场动画，而后台标签页没有绘制帧 ⇒ 动画永远停在起点，`getComputedStyle().opacity` 读到 0，扫描器按"不可见"把它们全部跳过（`/401` 那页报 `scanned:1` 而真实文字有 3 条）。补法是读前 `document.getAnimations().forEach(a => a.finish())`（另加全局 `transition:none`），A/B 现证：导航后 120ms `.fallback-title` opacity=0、`finish()` 后 =1。无限循环的那枚 6s 呼吸动画 `finish()` 不生效（规范对无限时长抛错），但它不动 opacity，无影响。转换器重播时又被两道对照各照出一个 bug：① 6 位 hex 用了 8 位的位移解法（`#ffffff` 读成纯蓝，白/黑从 21.00 变 16.75）；② gamma 方向写反（hex/rgb 分支把已编码值再套一次 `1.055·c^(1/2.4)−0.055`，`slate-400` 报成 `#465C7B`；`oklch` 分支则相反地少了这一步）。修完后按六个外部已知值验：白/黑 21.00、`#777777`/白 4.48、`#767676`/白 4.54、`#FF0000`/白 4.00、`#0000FF`/白 8.59、`oklch(70.4% .04 253.3)`→`#8FA2B9`/白 2.63。**扫描器本身也必须能红**：注入 `#777777`/白 的 div 报 4.48、SVG `<text fill="#888888">` 报 3.25、插一枚 `fill="#123456"` 属性被点到——三条对照都到位后才采信其"0 失败"。裸色检测踩的坑值得记：`getComputedStyle(svg).fill` 把 `#123456` 序列化成 `rgb(18,52,86)`，所以对 computed 值写 `#hex` 正则是**结构性死闸**（报 0 且永远不红），必须按**书写值**（属性 / 内联 style）判定。

本轮改的三处（都属"文字误取了别的档的色"，未动任何权威取值）：① `pages/core/auth/login/index.tsx:199,208` 的登录页副标写的是 `text-[color:var(--ant-color-text-tertiary)]`，暗色解析为 `#6E7681` 落在 `#111827` 面板上只有 **3.86 < 4.5**（这一处上一笔漏了：只改到 `:174` 的验证码文案，同文件这两行没进清单）→ 换成 `--gowind-secondary-text`（暗 `#8B949E`）= **5.77**；归属用双向换类现证（回退旧类 → 3.86，恢复新类 → 5.77）。② `IconOffline` 的 `<text>×</text>` 取插画物件色 `--fb-coral`：亮色 **2.55 < 3.0**（暗色 6.9 本来就好）→ 取 `--gowind-danger-text`（它本来就是"错误标记"的语义）= 亮 **5.6** / 暗 **5.94**。③ `IconComingSoon` 的 `<text>Coming Soon</text>` 取 `--fb-navy`：暗色 **2.56 < 3.0**（亮色 9.57）→ 取同色板里管文字的 `--fb-ink` = 亮 **13.46** / 暗 **18.3**。

覆盖面（现量、带分母）：公开路由从 7 条补到 **9 条**（新增 `/auth/mfa-challenge`、`/auth/sso/callback`——枚举取自 `src/router/config/{auth,error-routes}.tsx`，上一轮的 7 条是自己数的、不是从路由拿的），亮/暗两态各扫一遍，失败 0；逐页文字分母 亮=暗：登录 11、忘记密码 11、MFA 8、SSO 回调 7、401/403/500/offline 各 3、coming-soon 1 + SVG 文字 1；`skip` 里 opacity/hidden/unparsable 全为 0（这三个桶先前非 0 就是漏读的位置）。**SVG 里的形状与 `<path>` 数字不在判据内**（只判 `<text>/<tspan>`），401/403/500 三页的 `svgt` 计数为 0 即因此——它们的"401"是描出来的形状，不是文字。裸 hex 归属也纠正一次：先前看到"每页 62 处 `#FFF×30` 等"以为是本仓插画，实为**兔小巢第三方组件**（`button.tsqd-open-btn` 内两份 31 枚拷贝）；排除第三方后 react 登录页自有 10 枚（`#0D74F2`×8 + `#3B82F6` + `#70B3FF`，登录页插画的具名化还没做），401/403/500/offline/coming-soon 五页自有裸 hex **0 枚**、`var(--fb-*)` 引用分别 14/32/45/12/15。`npm run typecheck` 退出码 0，并给它补了阳性对照（往 `IconComingSoon.tsx` 塞一行 `const x: number = 's'` ⇒ 报 3 条、其中 2 条指向该文件），证明这 0 是"看得见这些文件"的 0，对照行已撤。**移植面**：ele/vben 的错误页插画里没有 `<text>` 节点（按名 grep，两端 `pages/core/error` 与 `framework/**/fallback/icons` 各 0 命中），所以本轮这条"文字取墨不取物件色"的规则在另两端**没有可移植的具体目标**，但规则本身已写进上面 §4 那条，三端新画插画时按它走；vben 的 3 处 §2.2 灰阶与 ele 的门禁不可跑，仍按上一笔记录挂着。

2026-10-14 第四笔（把上一笔挂着的两项落到，并更正其中一个前提）：**先更正**——那句"ele 无 Tailwind 灰阶写法（0 处）"是错的，实为 **2 处**，都在 `src/components/Pro/{ProForm,ProSearch}/index.vue` 的 `<ElIcon class="text-gray-400">`（当时只点了 pages 面）。但这两处**从来就不是对比度缺陷**：上面对 EP 组件根写 `text-*` 一律失效，图标实际按继承的标签色画，够；问题在"声明与渲染不符"。改走 EP 自己的 `--color` 钩子（`style="--color: var(--el-text-color-secondary)"`），实测该变量浅 `#676E7C`（落 `.el-card` 白底 **5.12**）/ 暗 `#8B949E`（落实测 `rgb(17,24,39)` **5.77**）——既落 §2.2，也让写的东西真画出来。
**vben 3 处**（`ai-provider-drawer.vue:183` `text-gray-400`、`system/script/index.vue:271` `text-neutral-400`、`:317` `text-neutral-500`）统一换 `text-muted-foreground`：公开 `/auth/login` 上按同类元素注入读计算色，4 类 × 3 底面（`--card`/`--background`/`--popover`）× 2 态 = 24 读，旧值现形为 `#9CA3AF` 浅 **2.54**（暗 6.99 达标）、`#A3A3A3` 浅 **2.52**、`#737373` 暗 **3.74 / 落 popover 3.41**（浅 4.74 达标）——**每处至少在一种态下 < 4.5**，三处都属真不合格；新值 `text-muted-foreground` 实测解析浅 `#676E7C` = 5.12、暗 `#A1A1AA` = 6.92（popover 6.31），两态全面达标。转换器在本页重新验活（`#767676`/白 4.54、`#777777`/白 4.48、`#0000FF`/白 8.59、白/黑 21.00 四点等值）。CSSOM 扫描这轮先**报了一场假空**：`if (r.cssRules) recurse; else 看 selectorText` 这种写法的闸在 Chrome 支持 CSS 嵌套后恒为真（普通 `CSSStyleRule` 也带 `cssRules`，长度为 0），于是 1716 条规则里 `selectorText` 命中 0 条——判据改成"先看 `selectorText`，`cssRules.length` 才下钻"，并往页面插一枚 `.ant-table-cell{color}` 假规则验证检测器**能红**（抓到）后才采信。顺带一条事实纠正：vben 端 `.text-gray-400` / `.text-muted-foreground` 两条规则**都不在任何 `@layer` 里**（本仓 vben 用 Tailwind v3 输出，`--tw-text-opacity` 那套），所以这端的工具类不会被 antd 的 unlayered 规则挤掉。
**ele 一处真不合格**：`analytics/analytics-trends.vue:99` 的 yAxis `axisLabel.color` 写 `isDark ? "#8c8c8c" : "#909399"`，`#909399` 在实测 `.el-card` 白底只有 **3.08 < 4.5**（暗色那支 5.28 达标）→ 改 `getChartSecondaryText()`（`src/utils/chart-palette.ts` 新增，读 `--el-text-color-secondary` 的实际值：canvas 不继承 CSS 变量，只能建 option 时取，沿用同文件 `getChartPrimary()` 的先例）⇒ 浅 5.12 / 暗 5.77；同文件 `:80` 的 xAxis `#CFD3DC`/`#606266` 实测暗 11.83 / 浅 6.11 都达标，未动。

顺带修掉一个挡住实测的工程缺陷（不属样式，但 vben 起不来）：`pnpm dev` / `pnpm build` 自 2026-10 的"workspace 归零"提交（`b5be60037`）起加载不了 `vite.config.mts`——`build-config/node-utils/src/index.ts` 仍 `export { rimraf } from 'rimraf'`，而删 `internal/node-utils/package.json` 时 rimraf 掉出了依赖集（catalog 里 `^6.0.1` 还在，无人声明）。红/绿现证：把 `node_modules/rimraf` 移开 ⇒ `ERR_MODULE_NOT_FOUND: 'rimraf'`；删掉这行死再导出（全仓 `rimraf` 引用除该行 **0** 处）⇒ `VITE v6.4.3 ready` + `/` 回 200。门禁为什么没抓到：vben `tsconfig.json` 的 `include` 只有 `src/**` 和 `framework/**`，`build-config/**` 不在程序里（`vue-tsc --listFiles` 现数：改过的两个 `.vue` 各 1、`build-config/node-utils/src/index.ts` 0）。为什么不加回依赖：试过把 `"rimraf": "catalog:"` 写进 `devDependencies` 再 `pnpm install`——它顺带把 `@intlify/shared` 从 11.4.13 漂到 **12.0.0-alpha.4**（lock diff 30 加 / 8 删），撞本仓"工具链版本钉死"的铁律，package.json 与 lock 已回滚（回滚后 `git status` 只剩本次改动文件）。

门禁与仪器自证：vben `pnpm run typecheck` 用对照跑——临时 `src/qoder-gate-control.ts` 塞一行类型错误 ⇒ 报 1 条并指向该文件、EXIT=2，且全量输出里 `error TS` 计数 **恰好 1** ⇒ 本树除对照外 0 错；对照文件已删。ele `npx vue-tsc --noEmit` 在改动树上 EXIT=0 / `error TS` 0，随后往 `analytics-trends.vue` 注入同样一行 ⇒ 报 `(19,7) TS2322`、RC=2；撤行后按 sha256 前 16 位 `793a18f2bcbd` 与 numstat 2/2 回证逐字节还原。**ele 端门禁本次是第一次跑起来**：本机原先没有 `node_modules`（故上一笔的 2 个 ele 文件从未过门禁），`pnpm install --frozen-lockfile` 后 `vue-tsc` 可用；起 `npx vite` 另缺 `@rolldown/binding-win32-x64-msvc`（lockfile 里有、首装漏）⇒ `pnpm install --force` 补齐后 :15778 起服（vite 8.0.13，`/login` 公开可达，页面里 5 枚 `.el-icon`）；上面所有 ele 读数取自这台自起服务器，:5777 那台经内容比对**不是本仓**（React 应用、`/src/main.tsx`）。
**仍未实测的面**：三处 vben 改动坐在登录后的抽屉/表格/弹窗里，而本机 :7788 的在跑二进制 `GET /admin/v1/captcha` 今日复测**仍 404** ⇒ 走不进登录后页面；类→计算色的映射是在公开登录页上按同类元素量的，**宿主页面本身没进过**。ele 的两枚图标同理（`--color` 钩子在合成元素上现证，ProForm/ProSearch 的表单标签区在登录后）。

2026-10-14 第五笔（三端品牌插画具名化 + 四处色板偏离落到实测）：**先纠正一处取证前提**——本轮最初的 react 读数取自 `:15889`，那台经进程命令行现证是 `D:\GoProject\go-wind-iot\frontend\admin` 的 vite（同族不同仓），`:7788`/`:7789` 同样不是本仓后端（`D:\CocosProject\tgqp\backend\bin\admin.exe`）。改在 `:15888` 自起本机 react dev 后，下面所有 react 读数都是重跑过的；起服两处坑：默认 `node` 是 v20.10.0，Vite 8 因 `node:util` 无 `styleText` 直接崩，需 PATH 前置 nvm 的 20.20.0；且该 dev server 只绑 `::1`，`curl 127.0.0.1:15888` 恒 `000`（看着像服务挂了）。**判据从此按进程命令行认端口，不按端口号记归属。**

四处"写在代码里但没画出来 / 画出来但脱离色板"的实测缺陷：① vben `design-tokens/default.css` 的 `accent-color: var(--primary)` 是**整条无效**——`--primary` 存的是不带 `hsl()` 的三元组，登录页实测 `getComputedStyle(html).accentColor === 'auto'`；新增 `--brand-primary: hsl(var(--primary))` 后取它，实测 `rgb(0, 107, 230)`（亮/暗两态同）。② vben `authentication.vue:78` 用 `dark:bg-[#070709]`（rgb(7,7,9)）盖在 `bg-background-deep` 上，而 §2.3 L0 是 `#0B0F19`；同一文件的 `.login-background` 渐变两端也写死 `#07070915`。改法是删掉那条 dark 覆写（token 本身随主题切）+ 两端写 `hsl(var(--background-deep) / 8.2%)`（0x15/255）。实测暗态品牌列底色 `rgb(7,7,9)`→**`rgb(11, 15, 25)`**，渐变两端与底色逐通道相等；顺带消掉亮态一圈无意识的暗角（原先两端是叠在 `#F1F3F6` 上的 8.2% 近黑）。③ vben 浅色 `--foreground: 210 6% 21%`（`#323639`，shadcn 出厂 zinc）vs §2.2 定稿 `#1F2937`——暗色侧早已按 §2.3 落 `#F8FAFC`，浅色侧一直漏迁；改 `215 28% 17%` 后登录页实测标题 **12.19→14.68**、验证码文案（落 `#E4E4E7`）**9.61→11.57**。口径边界：其余 12 个内置配色预设（violet/pink/…）各带一套中性色，不在本条覆盖内；`bg-foreground` 5 处（3 文件）随之同色相。④ ele 浅色实测 paints EP 出厂值：文字-主 `#303133`、常规 `#606266`、画布 `--el-bg-color-page` `#F2F3F5`。前两项的目标值 `index.scss:48-50` 早就写过，但被后注入的 base.css 顶掉（本轮现证读回出厂值），生效档补进 `vendors/_element-plus.scss` 的 `:root` 带 `!important`，那三行原地标注为死声明。**画布这条同一个坑，但选择器必须收窄到 `html:not(.dark)`**：`_dark-mode.scss:71` 的暗色档不带 `!important`，把 `!important` 写进 `:root` 会压掉它、`LayoutMain`（`:132` 用 `var(--el-bg-color-page)`）整片刷成浅色——**这条回归是改完立刻复测暗态时抓到的**（读回 `#f1f3f6`），收窄后两态现证：暗 `--el-bg-color-page=#0B0F19`/`--el-bg-color=#111827`、文字 `#F8FAFC`/`#DCE3ED`/`#8B949E`；浅 `#f1f3f6`/`#fff`、文字 `#1f2937`/`#374151`/`#676e7c`，逐条对上 §2.2/§2.3，且换回暗态读数与改前一致（幂等）。

两处新落的规则面：**react 登录页无响应式断点**——`UserLayout.style.less` 的表单列 `min-width:500px` 与品牌列 `flex:1` 在 ≤768px 把内容顶出可视区（实测不可达），补 `@media (max-width:1023px)` 收品牌列、表单列转整宽；同文件另一处幽灵变量 `var(--login-page-bg, #0B0F19)`（全仓只有这一处出现该名字，fallback 才是实际生效值）改直写。**react 文字层次补齐**——`semantic-text.css` 新增 §2.2/§2.3 的主档与常规档（`--gowind-text-primary` `#1F2937`/`#F8FAFC`、`--gowind-text-regular` `#374151`/`#DCE3ED`）；过程中**自己造了一处回归又抓回**：品牌副标改取次要档 `#676E7C` 后，落在渐变最深那档 `#D5E0ED` 上实测只有 **3.83 < 4.5**（canvas 仲裁），退回常规档 = **7.71**，"渐变底按最深档算"这条判据已写进该文件注释。**ele 兜底页整条祖先链无底色**——`/offline`（同 `--fb-*` 家族还有 401/403/500/coming-soon）从 `html`/`body`/`#app` 到 `.fallback-container` 全为 `rgba(0, 0, 0, 0)`，暗态实际由 `color-scheme: dark` 的浏览器默认画布兜底，底色不归设计语言管；给容器加 `background: var(--el-bg-color-page)` 后实测两态落底 `rgb(11, 15, 25)`/`rgb(241, 243, 246)`，两态各 3 节点失败 0。**改前那轮读数是量具兜底值**：扫描器找不到底色时落成 `#000`，于是报出 20.07/6.83 的"达标"——不是真实落底，记为量具盲区一处。

覆盖面（现量、带分母）：vben 公开路由**首次**扫测 `/auth/login` 浅 7 / 暗 9 节点、`/auth/mfa-challenge` 暗 8、`/auth/sso/callback` 暗 7（含失败态：danger 文字档在 L0 上 **5.94**、返回按钮白字落实测 `#035EC7` **6.12**，两条都正逢 §2.1「暗色语义色另算」那条），失败均 0、SVG 裸 hex 0、未解析 `var()` 引用 0；ele `/login` 浅 7 / 暗 7、`/offline` 浅 3 / 暗 3，失败 0；react `/auth/login` 浅+暗失败 0（标题 10.98 浅 / 16.96 暗、副标 7.71 / 5.77）、`/auth/forgot-password` 10 节点失败 0。三端 slogan 插画具名化各 15/15 处按 pattern 逐条计数替换，实测 33 节点 / 25 枚涂色属性 / 15 处 `var()` 引用 / 裸 hex 0 / 未解析 0。量具本轮新增三档闸门模式并在每次读数里自报 `geometryGate`，注入文本一律按 `chars=13589 / djb2=1d970e3b` 校验，不符直接返回 `INSTRUMENT MISMATCH`。**两条披露**：① vben/ele 的品牌列在 724px 合成视口下 `display:none`——vben 那两行是用注入样式强制显示后测的（颜色不随断点变，比值可用，但**没有真实绘制**），ele 品牌两行只按 computed color + 祖先底色读、未强制显示；② vben 的 404 兜底页进不去，未登录时守卫直接跳 `#/auth/login?redirect=…`（实测 `location.hash`），登录后页面（含 ele 固定列那条待移植项）本轮一律未实测。

窄屏（响应式）补测，含同一条规则改前/改后的因果 A/B：**react `/auth/login` @375**——视口不是靠合成容器宽度伪造的，而是同源 iframe（media query 按 iframe 自身视口求值，`contentWindow.innerWidth` 现读为 375/768/1023/1024/1366 五档）。改后：品牌列 `display:none`、`.user-brand-icon` 不参与布局、`.user-layout-wrapper.scrollWidth=375`（与视口相等）、五档越界节点 0。把 CSSOM 里那条 `@media (max-width: 1023px)`（所在表 index 30、cssText 187 字符）整条 `deleteRule` 后，同一文档读数回退到品牌列 `flex`、icon rect **−180..180**、wrapper scrollWidth **500**（溢出 125px）；`insertRule` 原样插回后与"改后"逐字段相等（`restoreIdentical=true`、该条件规则计数回 1）。**判据口径**：删除臂里 80 个 rect 越界的节点全部有 `overflow != visible` 的祖先（外层 wrapper 本身 hidden），所以"越界计数"在这里没有判别力——真判据是 `scrollWidth > 视口` 加上 icon 的 −180，"溢出部分既看不见也滚不到"正是缺陷形态本身。浅态文字臂（用真实钩子 `html[data-theme=light]` + `.light-mode`，非改应用 JS）：标题 `rgb(31,41,55)` = `#1F2937`、副标 `rgb(55,65,81)` = `#374151`，副标落在品牌面板渐变最深档 `#D5E0ED` 上 **7.71**，同底若取次要档 `#676E7C` 复算 **3.83**（与上一条自造回归那次数值互证）；锚点白/黑 21、`#777777` 4.48 均在位。**ele `/login` @375/768/1024**：三档 `documentElement.scrollWidth` 都等于视口，品牌区 ≤768 收成 0×0、1024 可见（结果量到，触发它的钩子是 `min-width` 还是 JS 未追），375 下输入框净宽 217px 仍可用——**无缺陷，未改**。**vben `/auth/login` @375/768/1024/1366**：无溢出；768/1024 第一轮读到的"3 个按钮越界"经查是调色盘预设条 `w-0 overflow-hidden` 折叠态下子节点的固有布局（被自身裁剪，不是漏出），把它按 `group-hover:w-60` 的终值强制展开后，工具栏因 `absolute right-2` 右锚定而**向左生长**（768：预设 380..604、组 380..652、`rightMost − 视口 = −8` 即留 8px 边距；1024/1366 同形态），越界计数 0。**这里命中了老坑「隐藏标签页冻结 transition」**：第一次只写 `style.width='240px'` 时组矩形纹丝不动，读出来的是假档——量几何终值必须先 `transition:none`。工具栏在 <768 只收起调色盘与布局切换（`hidden md:flex` 实测 `display:none`），语言/主题两颗仍在位（375 下 291..355，越界 0）。**对照臂两条**：CSSOM 注入式（往 vben 375 帧注入 `.rounded-full{width:900px !important}` ⇒ 命中数 6→29，并把被顶到 −1457 的祖先报了出来；撤样式复原）能红；而更早一版"往 `#app` 追加一个绝对定位宽节点"的对照法**失败**（`canRed=false`：追加进了隐藏子树，rect 0×0 被零尺寸过滤直接跳过），凡对照臂先证明它能红再谈它的读数。

门禁：react `npm run typecheck` EXIT=0 / `error TS` 计数 0；ele `npx vue-tsc --noEmit` 基线 EXIT=0 / 0，往本轮改过的 `src/pages/core/error/fallback.vue` 注入一行类型错误 ⇒ 报 **恰好 1 条**且指向 `fallback.vue(52,7)` TS2322、RC=2，撤行后按 sha256 前 16 位 `2f368ead02b90449` 回证逐字节相等；vben `pnpm run typecheck` 基线 EXIT=0 / 0，往 `authentication.vue` 注入同样一行 ⇒ **2 条**（TS2322 + TS6133，同位 `(35,7)`）、RC=2，还原 sha `956225b4e0209747` 逐字节相等。**ele 的 SCSS 改动不在任何 TS 程序里**（门禁结构上看不见），其真值只由上面两态的 CSS 变量读数担保。

2026-10-14 第六笔（落上一笔挂着的三项，并更正其中一项的取证前提）：**先更正**——上一笔那句"三端错误页插画的模板靛蓝 `#6c63ff`（react/ele `error/icons/warning.svg`、vben `icon-500.vue` + `warning.svg`）"里，三枚 `warning.svg` 全仓代码 **0 引用**（`grep -rn "warning.svg"` 排除 node_modules 后唯一命中就是本文档本节），react `layouts/UserLayout/components/Background.tsx`（95 行）与它 `:6` 引入的 `background.less`（194 行，内含 `#f5f7fa`:23、紫 `rgba(147,51,234,.1)`:75/123）**0 importer** ⇒ vite 结构上不会把那张样式表注进产物，那些硬编码从来没被画出来过。所以"活的模板靛蓝"只有 vben `icon-500.vue` 一处，本轮改掉。死文件本轮点名删除：三枚 `warning.svg` 已删（`7e88c345`）——判据是"三枚同一 blob `951a905c`、8911 B × 3、全仓 0 文本引用、0 无扩展名引用、无 svg-sprite loader"，并逐一排除了唯一能推翻该结论的机制（三处 `import.meta.glob` 的根分别是 `vue-element/src/assets/icons` 56 项与 `vue-vben/framework/icons/svg/icons` 9 项，与被删路径不相交）；删后三端门禁真实退出码 0、ele `/offline` 现证插画照常渲染（4 枚 svg、64×64 与 40×40 有真实矩形）且 console 无 `Failed to resolve import`。react 那两枚无 importer 的文件仍在。

三项落地的实测：**① react 认证页浅色画布** `#f5f7fa`→§2.2 的 `#F1F3F6`：不靠手写 class，真点主题按钮（`.user-toolbar-btn-dark`）⇒ `html[data-theme=light]`、React 自己给 wrapper 加 `.light-mode`，实测 `backgroundColor` `rgb(11,15,25)`→**`rgb(241,243,246)`**；因果臂往同文档注入 `.user-layout-wrapper.light-mode{background:#f5f7fa !important}` ⇒ 读数变 `rgb(245,247,250)`，撤除后回 `rgb(241,243,246)`（对照能红 + 复原现证）。**② react 品牌圆 `.circle-2`** 的紫→`color-mix(in srgb, var(--brand-primary) 10%, transparent)`，与同文件 `.circle-1`（15%）同一写法：暗态实测计算值 `radial-gradient(circle, color(srgb 0 0.42 0.9 / 0.1) 0%, rgba(0,0,0,0) 70%)`，即 `#006BE6` 的 10%，且 rect 200×200、`opacity:1`、祖先 `.user-brand-section` 749×900——真绘制，不是只有计算值。**③ vben 暗色次要文字**：`design-tokens/dark.css` 默认块（`.dark, .dark[data-theme='custom'], .dark[data-theme='default']`）的 `--muted-foreground` 由 `240 5% 64.9%`（`#A1A1AA`）改 §2.3 定稿三元组，活文档实测解析 **`rgb(139,148,158)` = `#8B949E`**，与 react `SECONDARY_TEXT_DARK`、ele `--el-text-color-secondary` 暗态同值。口径边界：`240 5% 64.9%` 这个字面量在本文件出现 5 处，其余 4 处坐在 12 个内置配色预设块里（选择器权重 0-2-0，顶过本块），不属"默认暗色"这条口径、未动；同族的一处过时注释（`framework/styles/antd/index.css` 里"暗色侧不动：`#A1A1A8` 实测无违规"）已按现值改写。

接线对照与比值互证：往 iframe `html.dark` 注入 `--muted-foreground: 211.6 8.92% 22% !important` 后，同一批节点里"Copyright © 2026"计算色变 `#33383D`、比值 **1.62**（能红），撤除后回 6.23 ⇒ 现证这批文字真在消费该 token，而不是"定义了个值没人读"。独立实现（node，非浏览器内那段）复算 `#8B949E`：L0 `#0b0f19` **6.23**（与浏览器内实测逐位相等）、L1 `#111827` **5.77**、§2.3 钉的浮层 `#1c2128` **5.26**——后两档与文档数字完全对上，量具与规范互证；旧值 `#A1A1AA` 同底面 7.47 / 6.92 / 6.31，两档都达标，换过来是为了三端同值而非救不合格。另：本次在 vben 登录页实测到的浮层底是 `#1C2333`（不是 §2.3 写的 `#1C2128`），`#8B949E` 在其上 **5.10** 仍达标——**只记这个差异，不动权威取值**，底面归属待查。

一条仪器层面的教训：**顶层标签页视口 `innerWidth/innerHeight = 0×0`**（内嵌浏览器面板收起态，`document.body` 宽 0），于是 react 顶层页 `.user-brand-section` 是 `display:none`、`.circle-2` rect 0×0，而上一节在 vben 顶层页扫的对比只剩 5 个可见节点——分母是被退化视口削掉的，读数却一副正常。本轮 react/vben 两侧所有几何与"真绘制"读数全部重取于同源 iframe（`contentWindow.innerWidth=1440` 现读）；同一张 vben 暗色登录页在 1440×900 下扫出 **9 节点 / 5 组 fg-bg / 失败 0**，`#8B949E` 落在 4 个节点上（品牌列两行进入可见集，顶层页里它们是 0×0）。选判据时先问"这一臂测不测得到它"，别等读数看着正常才想起分母。

披露：`icon-500.vue` 属**框架分发面，本 app 端到端未实测**——全仓 `status="500"` 出现 0 处，`src/views/_core/fallback/` 只挂 403/404 两个视图，该图标由 `fallback.vue:94` 的 switch 分支经 `defineAsyncComponent` 动态引入，无路由可达。实测到的是机制而非产物：暗态活文档里注入 `<rect fill="var(--brand-grad-mid)">` 读回 `rgb(13,116,242)`（`:root` 的 `--brand-grad-mid: #0d74f2`，两态同值、dark.css 未覆写），文件内 10 处 `var()` / 残留 `#6c63ff` **0**。

门禁：三端各自目录全量跑，完整输出里 `error TS` 计数 **0** 且 `error|fail|cannot|ERR_` 零命中——react `npm run typecheck`（`tsc -p tsconfig.app.json --noEmit`）、ele `npx vue-tsc --noEmit`（本轮 ele 无文件改动）、vben `pnpm run typecheck`（`vue-tsc --noEmit --skipLibCheck`）。注意这三条这次是经 `| tail` 捕获的，管道末端退出码不属于 tsc，所以判据取"输出里 0 条诊断"而非 EXIT 码。

**未动、需点头**：ele 暗色 vxe 固定列与 35 处 `fixed: 'right'` 的同构漏字（react `10637946` 先例）——本机在跑的 :7788 二进制取不到 captcha、登录后页面进不去，按构造改的风险大于收益；vben `icon-500.vue` 其余物件色（`#2f2e41` 15 枚 / `#3f3d56` 8 枚 / `#f2f2f2` 4 枚 / `#a8a8a8` 1 枚，插画形状而非主题 chrome）保留。剩余死文件候选（已现量引用面，删不删待点名）：react `layouts/UserLayout/components/{Background.tsx,background.less}`（95 + 194 行、0 importer）、ele `src/assets/images/401.svg`（34510 B）与 `404.svg`（26633 B）——该目录只有 `logo.png`/`default-avatar.png` 被 import，两枚 `.svg` 全仓 0 引用（含 `assets/images` 路径式与模板串写法都查过）。react `public/icons.svg`（5031 B）**不属这一口径**：public 目录无条件拷进 `dist`，静态 grep 0 命中不足以判死。

2026-10-15 用户裁决回退一条（§2.1「危险实底按钮的沿革」）：亮色 danger 实底按钮的前景从暗墨 `#0B0F19` 改回各 UI 库原生白字，亮/暗两态一致——用户原话场景是「同步 API」按钮亮色下黑字坐在红底上刺眼。落点：react `semantic-text.css` 删 `html[data-theme='light'] .ant-btn-color-dangerous.ant-btn-variant-solid` 那条、vben `framework/styles/antd/index.css` 删 `html:not(.dark) .ant-btn-primary.ant-btn-dangerous` 那条（ele 本就无此规则，EP 原生白字）。`--gowind-solid-ink` 的适用面随之收窄为徽标数字/开关轨道这类非按钮小字；success/warning 实底按钮的暗墨不受影响。
同日第二批（徽标）：通知铃铛角标数字在 react/ele 两端同为暗墨坐红底，用户裁决一并回白字。落点：react `semantic-text.css` 删 `html[data-theme='light'] .ant-badge-count.ant-scroll-number` 那条（antd 原生白字，暗色本就未覆盖）、ele `_element-plus.scss` 删 `.el-badge__content--{danger,primary,success,warning}` 那条（EP 原生 `--el-color-white`，该规则不带主题作用域、暗色也一并恢复白字）；vben 无徽标规则未动。`--gowind-solid-ink` 适用面再收窄为仅开关轨道（ele 侧登录页验证码占位文案另有使用）。

2026-10-16（react 端样式修整一轮，用户口径「整体、亮暗两态都看」）：**暗色 placeholder 下调一档**，判据见 §2.2 末注的暗色补充——落点三处：`core/preferences/config/darkTheme.ts`（`colorTextPlaceholder` + `DARK_PALETTE.placeholder` 都改指 `#6E7681`，token 是 Select/Picker/AutoComplete 这些没进 CSS 覆写的控件的唯一来源）、`styles/pro-components-dark.css`（`!important` 那块跟着换 hex，并把它里面那条 **v5 死选择器** `.ant-select-selection-placeholder` 改成 v6 现名 `.ant-select-placeholder`：同页现数 旧名命中 0 / 新名命中 2，同族的 `.ant-select-selection-item`  同样命中 0，值色目前靠 token 兜着所以没露馅）、`components/common/Editor/src/tiptap-editor.css`（暗态空态提示原取 `--tte-text-secondary` `#8B949E`，比改后的表单 placeholder 还亮，一并降到 `--tte-text-muted`）。实测（`:opm/users`，同源页读 `::placeholder` 计算色）：暗态 input 与 select 全部 `rgb(110,118,129)` 落 `rgb(17,24,39)`，亮态仍为 antd 派生 `rgba(0,0,0,.25)` 未受影响。
同轮另四处同类瑕疵（都是"暗色优先写法在亮色下失效"，不动权威取值）：① dashboard `StatsCard`/`AiInsightsCard` 的图标块用 Tailwind `-400`（为深底校准的档）坐白卡，实测 1.61~2.98 连图形 3:1 都不够 → 补亮色档 `-600/-700`（cyan/emerald 试 `-600` 仍贴线 2.98/3.01，提到 `-700` 才 4.35/4.42），暗色侧保持 `-400`；卡片边框与内层小盒从 `border-white/N`、`bg-white/5` 改吃 `--ant-color-border-secondary`/`--ant-color-fill-quaternary`（同类还涉及 `RoleDrawer`、`ScriptDrawer`、`InboxDetailDrawer`）。② AI Agent 画布节点写死的 `#52c41a/#1677ff/#8c8c8c` 是**填充档**当图标文字用（暗色 2~3:1），且 `#1677ff` 是已退役旧主色 → `semantic-text.css` 新增 `--gowind-node-*` 两态档（暗取 -5、亮取 -7）。③ tiptap 编辑器自带一套 slate 暗色板（`#0f172a/#1e293b/#334155`）与全站蓝黑板并存，页面上是"两种暗色" → 12 个变量对齐 §2.3 的 L0/L1/L2/L3。④ `ai/query` 的 SQL 块 `bg-gray-900 text-gray-100` 不跟主题 → 改 `--ant-color-fill-tertiary`/`--ant-color-text`。
标签栏暗色选中态同轮调整（`.tabsbar-chrome/.tabsbar-card`）：选中底由 14% 白降到已有的 `--ant-color-fill-secondary`（9% 白，与亮色档的减重比例 1.52→1.28 对齐亮色实测 1.14），并补 `--ant-color-border-secondary` 描边、未选中文字降 `--ant-color-text-secondary`（L1 上 5.77）hover 回主档——**减重的前提是同时把"哪页是选中"这件事交给文字通道承担**，否则单纯压低底色会让选中/未选中糊成一片。
**未动、需点头**：暗色主色实底按钮仍走 antd darkAlgorithm 派生 `#035EC7`（白字 6.12 达标），是否统一成品牌蓝 `#006BE6` 未定；`tabsbar.css` 里标着"暗色模式适配"的 `.tabsbar-chrome`/`.tabsbar-card` 嵌套块是死代码（被 `.ant-tabs-nav .ant-tabs-tab` 的 4 类权重压过，两态都不生效，暗色 hover 底实际一直吃亮色档 `rgba(128,128,128,.08)`）。
**ele/vben 待同步**：`vue-element/src/styles/_dark-mode.scss:26,59` 与 `vendors/_element-plus.scss:526` 的 `--el-text-color-placeholder` 仍是 `#9CA3AF`（暗色 6.99，本轮同一判据下也该降；EP 侧另有 `--dark-text-placeholder` 这个名字被时间戳/辅助说明共用，同步时要先拆开再动，否则会把非占位的三级文字一起压暗）；vben 的暗色 `--muted-foreground` 承担同类角色，需按同判据复测。
门禁：react `npm run typecheck` 见本节末尾通用口径；本轮为纯 CSS/token 改动，无 TS 语义变化。

2026-10-16 同日第二批（抽屉一轮，用户报的是"底部区背景 + 分隔线"，实测根因在线与一批**静默失效的规则**）：先给判据结论——抽屉底部区背景本身合规（暗 `#111827` 与抽屉体同层、亮白，两态一致，靠发丝线分隔，是 2026 既定口径），正因为层次全交给那条线，**线一不画底部区就"糊"进内容**。踩到的三类量具陷阱：
① **antd v6 cssVar 实名换了，写错不报错**。`color: var(未定义)` 退化成 inherit，`border: 1px solid var(未定义)` 整条 shorthand IACVT 成 `border-style:none` ⇒ 线直接消失。现证（`:opm/users` 新建用户 + 顶栏偏好设置，两态各读）：偏好抽屉表头线两态都是 `0px none`、副标题两态都取**主档**（暗 `#F8FAFC`）。错名对照（react 端共 28 处 / 5 个 `.less`，另 2 处在 `iframe-layout.less` 与 `pro-components-dark.css`）：`--ant-text-color`→`--ant-color-text`、`--ant-text-color-secondary`→`--ant-color-text-secondary`、`--ant-text-color-tertiary`→`--ant-color-text-tertiary`、`--ant-border-color`→`--ant-color-border`、`--ant-border-color-split`→`--ant-color-split`、`--ant-color-mask`→`--ant-color-bg-mask`、`--ant-select-option-selected-bg`→`--ant-color-primary-bg`（后者是"选中项 hover 不冲灰"那条 `color-mix`，变量名不存在 ⇒ 规则整条没落地过）。改后现量：暗态偏好抽屉表头/底部都 `0.667px solid rgba(255,255,255,.08)`、副标题 `#8B949E`；亮态两条都 `rgba(5,5,5,.06)`、副标题 `rgba(0,0,0,.65)`，主/次层级回来。
② **浮层投影的载体类名也换了**：v6 抽屉投影挂在 `.ant-drawer-content-wrapper`，弹窗的底色与投影都在 `.ant-modal-container`（`antd/es/modal/style/index.js:122-130` 现读 `contentBg`/`boxShadow` 挂 `-container`），而 `global.css` 的黑投影覆写打在早已无对应节点的 `.ant-drawer-content, .ant-modal-content` 上（DOM 命中 0）⇒ 暗色浮层吃到的是 darkAlgorithm 派生的**白** α 投影 `rgba(255,255,255,.016) -6px 0 16px…`，正是 §2.3「投影必须用黑」要避免的。改指新名后暗态 wrapper 现读 `rgba(0,0,0,.55) 0 12px 48px, rgba(0,0,0,.4) 0 2px 8px`；亮态本就黑，未受影响。modal 那半条是**源码为证、未实开弹窗**。
③ 同抽屉两条线不同档：表头被 `pro-components-dark.css` 硬写 6%（落 L1 仅 1.170）、底部 8%（1.236），而 token `--ant-color-split` 暗色本就是 8% ⇒ 删掉硬写回 token，两线同档。表单标签硬写 `#C9D1D9`（11.49，不在 §2.3 色板）改吃 `--ant-color-text-heading` = 权威 `#DCE3ED`（13.73）；同文件那条"比画布亮一档（`#161B22`）"的注释与值不符，一并订正为"抽屉属 L1 `#111827`，与卡片同层"。
顺带一条同类残留：`pro-components-dark.css` 里 `.ant-select-selection-placeholder`、`.ant-select-selection-item` 都是 v5 类名（现数命中 0），前者上一轮已改对 v6 的 `.ant-select-placeholder`，后者的选中值色目前靠 token 兜着、两态实测无色差，暂留。
**未动、需点头**：① 亮色抽屉两条分隔线实测 1.140，比 §2.2 权威边框 `#E4E4E7`（1.269）还淡，而底部区又与体同色，亮色下"底部区"基本只由这条淡线撑着——提不提档待裁决；② `.ant-pro-top-nav-header`（`pro-layout-overrides.css:57`）与 `chat-page.css:18` 仍取 `--ant-box-shadow` / `--ant-box-shadow-tertiary`，这两个 token 在暗色实测就是白 α（`rgba(255,255,255,.016)…`），与 §2.3「投影必须用黑」相冲，但属顶栏/聊天页 chrome，不在本轮面内；③ react 无浅色 token（§2.2 开头"react 半 ✓"那条），本轮新亮的边框实测落 `#D9D9D9`、副标题落 `rgba(0,0,0,.65)`，与 §2.2 的 `#E4E4E7` / `#676E7C` 仍有差，属既有缺口非本轮引入。


2026-10-16 同日第三批（react「太过于生硬」一轮，用户口径是先评估归因、随后授权 ①–④ 全落）：**归因结论是感受成立，且主因不是骨架屏**。骨架覆盖面与另两端同级（react 42 页 `ListTable/TableSkeleton` + `LOADING_DELAY 250`，ele 41、vben 42），差的只是 shimmer 与骨架→内容的交接，撑不起整体评价；真正的量级在两处——**浅色没有层级系统** 与 **三个反馈通道被关掉/没接**。
① **新增 `core/preferences/config/lightTheme.ts`**（§2.2 的浅色 token，本文件此前不存在，`ThemeProvider` 的浅色分支一直是"保持 antd 默认"）。落 `colorBgLayout #F1F3F6` / `colorBgContainer #FFF` / `colorBorderSecondary #E4E4E7` / `borderRadiusLG 12`，另在 `styles/global.css` 补浅色卡片静止 `0 1px 2px rgba(0,0,0,.05)`、悬浮 `0 4px 12px rgba(0,0,0,.08)` 与 `transition: border-color, box-shadow .2s`。判据现量：改前白卡压 `#F5F5F5` 分离 **1.049**、卡片描边 `#F0F0F0` 对白底 **1.045**（≈隐形）且投影 `none`，即浅色全靠一条看不见的线撑层次；改后画布 `#F1F3F6`(1.062)、描边 `#E4E4E7`(1.269) + 1px 贴身投影。antd 不给"带边框卡片静止投影"任何可覆写 token（`es/card/style/index.js` 只把 `boxShadowTertiary` 给 `:not(-bordered)`、`boxShadowCard` 只服务 hoverable 的 hover，且 `boxShadowCard` 不在 `AliasToken` 上，写进 `ThemeConfig.token` 被 TS 挡回 TS2561），所以两档只能落 CSS——这一点也订正了 `global.css` 原注释里"亮色卡片自带边框层次无需加"的错误假设。
② **连带白捡一条既有缺口**：`es/theme/util/alias.js:51` 的 `colorSplit = getAlphaColor(colorBorderSecondary, colorBgContainer)`，于是抬 `colorBorderSecondary` 同时把浅色分隔线从 `rgba(5,5,5,.06)`（合成 `#F0F0F0`，1.140）抬到 `rgba(10,10,37,.11)`（合成 `#E4E4E7`，**1.269**）——上一轮抽屉那节"未动、需点头"的第 ① 条（浅色抽屉两条线只有 1.140、低于权威 1.269）就此按权威档解决，无需单独裁决。
③ **重新打开点击波纹**：`ThemeProvider` 原带 `wave={{ disabled: true }}`（注释写"提升 Admin 操作手感"），三端里只有 react 关它。删除后实测 `MutationObserver` 抓到 `DIV.ant-wave` 随点击入树。⚠️ 排障坑：改 `ConfigProvider` 的这层 props 后 **HMR 不会重建 provider**，量具会在 `waveNodes=0` 上连续假阴性——必须整页 reload 再测。
④ **接上路由入场动效**：`MainLayout` 此前是裸 `<Outlet key={refreshTrigger}/>`，实测内容区 `transition: all/0s`、切页 opacity 恒 1 ⇒ 硬切；而 `preferences.transition.{enable,name}`（四档 fade/fade-down/fade-slide/fade-up + 偏好面板里的选择器）在路由层**从未被消费**，只有错误页 `Fallback` 读它——UI 上是个显示"已生效"的死开关。现按 key 挪到包裹 div 上（保住 `flex:1 + min-height:0` 高度链，列表页表格依赖它）、按 `transition.name` 挂 `.transition-*`、`enable=false` 挂 `.transition-disabled`，并加 `@media (prefers-reduced-motion: reduce)`（§2.7 无障碍那条要求）。规格按 §2.7 取 240ms `ease-in-out`、位移 **8px**（预览卡片里的 `-20px` 是 36px 小方块的尺度，整屏用会像滚动跳帧）；**只做起不做落**——ProLayout 的 Outlet 没有可挂的退出相位。验证走 `document.getAnimations()`：切页后读到 `gowind-page-fade-slide` 且 `playState=finished, currentTime=240`，加 `.transition-disabled` 后 `animationName` 转 `none`。
⑤ **一条自我订正**：上一轮报给用户的"表格行 hover 硬切（`.ant-table-row` transition `all/0s`）"**不成立**——行 hover 底色画在 `td` 上（`pro-components-dark.css:22-24` 的 `> tr:hover > td`），而 `td` 自带 `transition: background-color, border-color / 0.2s`（两态实测一致），`tr` 上量错了元素。遗留的小不一致：固定列的高亮叠在 `background-image` 渐变上（为防 sticky 底下透字而设计），渐变不可插值 ⇒ 固定列仍瞬变，主体行 0.2s 渐显，本轮不动。
两态复测：浅色 `/dashboard/workspace` 卡片实读 `shadow rgba(0,0,0,.05) 0 1px 2px`、`border 0.667px #E4E4E7`、`radius 12px`，`--ant-color-bg-layout #F1F3F6`、根背景同值（原 `ThemeProvider` 里硬编码的 `#f5f5f5` 已改读 `LIGHT_PALETTE.bgLayout`）；暗色零回归（画布 `#0B0F19`、卡片 L1 `#111827`、边框/split 8% 白、投影 `none`、圆角 12、placeholder `#6E7681`）。门禁 `npm run typecheck` 0 错误。
**已知不一致，未动**：浅色列表页的搜索卡片是 `.ant-pro-card`，它自带 `0 4px 6px -1px rgba(0,0,0,.1)` + 边框 `#E5E7EB`（Tailwind `shadow-md`/`gray-200` 档），与 §2.2 的卡片档（`0 1px 2px`/`#E4E4E7`）并存——两套卡片系统两种投影，ProTable 系页面看着比 dashboard 更"高"，是否统一到 §2.2 待裁决。

2026-10-16 同日第四批 + 第五批（第四批用户报"搜索栏有立体阴影、下面列表 card 是平的"；第五批用户口径"能处理的，都处理了"）：**第四批**判定感受成立——搜索卡 `.ant-pro-table-search` 吃 Tailwind `shadow-md`、列表侧一条投影都没有，两块之间还空 24px，没有叠放关系却读出"搜索卡压在列表上面"；按方案 1 把两块统一到 §2.2 静止档（暗色那条 `shadow-xl` 压在 `#0B0F19` 上不显形，属"写了不生效"，删），描边两处越界值回 `--ant-color-border-secondary`。**本节末原来那句"是否统一到 §2.2 待裁决"到此关闭。**
**第五批**把上一版留下的错处与全部可客观判定的待办一次清掉：
① **盒子载体上移**（订正 §2.2 新注里"打在 `.ant-table` 上"那句）：`.ant-table` 坐在 `.ant-pro-card-body`（出厂 `padding: 0 24px`）里，描边画在它身上就比搜索卡**窄 48px**；宿主 `.ant-pro-card` 实测与搜索卡同 x 同宽、面也已是 L1/白，只是 `border-width: 0`——它是"没画出来的那块卡片"。现在 border/radius/box-shadow 全归宿主，表格只留 antd 自己的网格线，卡片体内缩 24→20（四边同一档 p-5），ProTable 写死在宿主 inline 上的 `padding-block-start: 0` 用 `!important` 收回、工具条补的 8px `padding-top` 同步归零。两态复测 `/opm/users`：亮 `x=240 w=1120` / 暗 `x=240 w=1120`，两块逐位相等，四边留白 20/20/20/20。
② **侧栏折叠收成一条时钟**（判据与数值见 §2.7 新注）；站名从"随折叠条件卸载"改为常驻淡出。
③ **死规则清完了，但顺带证伪了一条旧判断**：`.custom-scrollbar`、`.ant-pro-menu-item`、`.ant-pro-menu-item-selected`、`.ant-pro-sider`、`.ant-pro-top-nav-header` 五组选择器现数命中 0 且全仓无引用（`BasicLayout` → 自定义 `AuthenticatedLayout`，ProLayout 从不挂载），已删——这也把上一批"未动②"里 `.ant-pro-top-nav-header` 取白投影的那半条一并解决（`chat-page.css:18` 的 `--ant-box-shadow-tertiary` 仍在）。但**上一批记的"`.ant-select-selection-item` 同属 v5 死类名"不成立**：`antd/es/select/style/index.js:26` 与 `select-input-multiple.js:71` 至今仍在生成它，命中 0 只是那次量测的页面上没有选中值 ⇒ 该规则保留。教训：类名"命中 0"只有在**该控件处于会渲染它的那个状态**时才等于死亡。
④ **暗色文字/描边越界值回单一真相**：`.ant-menu-dark` 的 `#f0f6fc`、`#fff`、`rgba(230,237,243,.72)`（GitHub 侧三档，都不在 §2.3 色板上）→ `var(--ant-color-text)` 与 `rgba(248,250,252,.72)`；`DARK_PALETTE.borderCard` 从 5% 白（§2.3 点名"糊成一片"那档、且与 token 实际生效值不一致）订正为 8% 白并让 `colorBorderSecondary` 直接引用它，同时删掉无人引用、与 §2.3 又差一档的 `borderLine: 6%`；表头分隔线 `#E5E7EB` → token。另删一条从未生效的 `.ant-pro-table .ant-table-tbody > tr > td { border-bottom: 1px solid }`——亮/暗两态实测行分隔线一直是 antd colorSplit（亮 `#F0F0F0`、暗 6% 白），这条 shorthand 被 antd 同权重且更晚注入的规则整体压过。
两态实测零回归（暗：卡片 8% 白描边 + 投影 `none` + 表头 `#1F2937`；亮：静止投影 + `#E4E4E7` + 表头 `#FAFAFA`）。门禁 react `npm run typecheck` 0 错误。**仍留给用户点头**：暗色主色实底按钮与分页选中项用 antd 派生 `#035EC7` 还是品牌 `#006BE6`；`tabsbar.css` 标着"暗色模式适配"的死嵌套块是否清理；ele/vben 的暗色 placeholder、分页间距与本轮动效差异属跨端同步债。
