# 打印模板（单据打印设计器）设计文档

> **状态：方案定稿（2026-10-27），代码未实施。** 本文档是打印模板模块的唯一权威说明：
> 定位边界、选型决策记录、总体架构、数据模型、三端接入、输出链与分期计划。
> 动打印模板相关代码前先读本文；实施后各阶段状态以文内标记为准。

## 1. 定位与边界

打印设计器回答的问题只有一个：**把一条（或一组）业务数据，按用户自定义的模板排到纸上**。
定位是「单据 / 标签打印模板」，不是 BI 报表工具。

与相邻能力的分工：

| 需求形态 | 用什么 |
|---|---|
| 把列表数据拿走加工 | Excel 导入导出（已有，服务端导出 + 客户端导出） |
| 数据统计、交叉分析、大屏 | **不做**（BI 报表是独立品类：积木报表 / UReport） |
| 单据、标签、面单、工牌排版打印 | **本模块** |
| 打印前的数据加工 / 条件逻辑 | 模板内轻量表达式（内核自带）；图灵完备加工走脚本系统钩子（§7） |
| 「描述一句话生成模板」 | AI 生成（§7，接 AI 模块） |

**明确不做**：BI 报表设计器、套打毫米级精调承诺（标签/面单类以连续纸 + 微调偏移配置覆盖）、
租户自写服务端渲染插件、工作流表单打印（无工作流）。

竞品事实：JeecgBoot 的积木报表（含打印设计）是其核心卖点；简道云/宜搭等平台打印模板是标配；
若依没有。本模块是「底座给轨道业务造车」的轨道能力——采用者用 CRUD 生成器造出订单/库存/工单后，
单据出不了纸面，业务模块就没有闭环。

## 2. 选型决策（决策记录）

前提约束：本仓 **MIT**；产品定调「做完善」（深度绑定内核、长期演进）——因此许可证从
"风险项"升级为"一票否决项"，稳定性优先于省代码（维护者裁决：不介意多实现代码，只要稳）。

| 候选 | 协议 | 结论 |
|---|---|---|
| hiprint（vue-plugin-hiprint） | 内核 hiprint 2.5.4 公认 **LGPL**，npm 包装声明 MIT 属衍生争议区 | ❌ 淘汰。生态最大但许可证脏，与"做完善 + MIT"冲突 |
| **worm-vue3-print** | **MIT**（依赖均已审计，全 MIT/Apache/ISC 系） | ✅ **基座**。见下 |
| openprint | **GPL-3.0** | ❌ 代码零复用。仅采点子（AI 生成模板、云打印队列），且仅 7 commits 无架构参考价值 |
| 全自研 | MIT | ❌ 不选。拖拽画布是打磨型工程（吸附/标尺/撤销重做/组合），月级成本；接管 fork 后稳定性取决于"谁维护"而非"谁写的"，fork 即把风险转入自己手里 |

**决策：fork 接管 [worm-vue3-print](https://github.com/worm-longliu/worm-vue3-print)（v1.3.11，MIT）的
`core` + `canvas` + `common` 三个包，以独立仓维护（建议名 `go-wind-print`，npm 包 `@go-wind/print-*`，
具体名 M0 定稿），本仓按精确版本依赖自己的 fork——不挂靠上游 npm。**

决策理由：

1. **稳 = 自己维护，而非挂靠他人版本。** 上游 18 stars、单人维护、bus factor = 1，作为 npm 依赖
   长期挂靠是最大风险源；MIT 允许随时 fork，fork 之后该风险即告消除。上游继续活跃则择期同步
   （fork 保留 upstream remote 与同步窗口），停更则无感。
2. **架构与本仓同构**：`core` 纯 TypeScript 零框架依赖（表达式引擎 + 同构渲染管线
   `bindData → paginate → generateHtml`），三端天然共用；`canvas` 是 Vue 3 设计器画布，
   vue-element / vue-vben 直接使用；`common` 是 Custom Element + Shadow DOM 通用宿主形态，
   是 react 端的预期通道。
3. **文档与工程化成熟**：中文文档全套（快速开始 / API / 表达式引擎 / 渲染管线 / 三端一致性 /
   静默打印）、32 项 UI 验收脚本、架构守卫脚本、诚实标注的已知限制。
4. 上游私有件是"可重建的私有"而非黑盒：Electron 静默打印客户端不发源码但 **WebSocket 协议与
   SDK 开放**（`core/client`，端口探测自 127.0.0.1:17521，`connect/listPrinters/print/pair`，
   错误码齐全），客户端二进制挂其 GitHub Releases；服务端 PDF 微服务私有但 `generateHtml`
   开放，Go 侧 headless Chromium 渲染该 HTML 即为自有实现（§6）。

## 3. 总体架构

```
┌─ go-wind-print（fork，独立仓，MIT）────────────────────────┐
│  @go-wind/print-core     纯 TS：表达式引擎（safelist）、     │
│                          bindData/paginate/generateHtml、   │
│                          静默打印 SDK（/client 子路径）      │
│  @go-wind/print-canvas   Vue3 设计器画布（拖拽/标尺/图层/     │
│                          撤销重做/多选组合；无 UI 库依赖）    │
│  @go-wind/print-common   Custom Element 形态（react 宿主）   │
└──────────────────────────────────────────────────────────┘
            │ npm 精确版本
┌─ go-wind-admin（本仓）─────────────────────────────────────┐
│  后端模板域：sys_print_templates CRUD + 种子模板 + 权限/租户  │
│  三端接入：模板管理页 + 数据绑定面板 + 列表行「打印」         │
│  输出链：浏览器打印 / 服务端 PDF（chromedp）/ 静默打印对接    │
│  差异化：AI 生成模板（AI 模块）/ 脚本渲染钩子 / 套餐配额      │
└──────────────────────────────────────────────────────────┘
```

数据流（模板 JSON 是唯一事实源，存库）：

```
sys_print_templates.payload (TemplateData JSON, 毫米布局, 含 schemaVersion)
  → bindData(payload, 行数据)     // {field.path} 绑定 + 表达式求值
  → paginate(bound, …)           // 智能分页/跨页表头/小计
  → generateHtml(pages)          // 同构 HTML
  → 浏览器打印 | chromedp → PDF | 静默打印客户端
```

模板内表达式用内核自带的 safelist 求值器（轻量、无 IO、安全），**不接** Lua/JS 脚本引擎
（重量不匹配）；需要图灵完备的数据加工时走脚本系统新增的渲染前钩子（§7）。
模板元素含任意 HTML 时由内核依赖 dompurify 消毒（fork 继承该依赖并纳入审计清单）。

## 4. 数据模型与后端

按 CRUD 模块定式（`docs/crud_module_guide/`）落地，命名细节 M1 定稿，方向如下：

- 实体 `sys_print_templates`（遵守"ent 实体禁叫 Config"类命名纪律）：

| 字段 | 说明 |
|---|---|
| `name` / `code` | 显示名 / 唯一编码（code 租户内唯一，供业务按编码调用） |
| `resource_key` | 绑定资源标识（对应菜单/资源的 generated service 键） |
| `payload` | TemplateData JSON（JSONB），带 `schemaVersion` 字段供演进 |
| `builtin` | 系统内置标记（内置模板禁删改，可复制派生） |
| `tenant_id` | 租户外键，走既有租户隔离 mixin |
| `paper` / `orientation` / `margins` | 常用版式冗余列（列表页直接展示，免解析 JSONB） |

- 命令链：proto → `gow api` → `make ts` → 管理页「接口同步」登记 Api 表 → 「菜单同步」MERGE；
  与 `docs/field_change_guide.md`、`docs/crud_module_guide/` 完全同轨，无特殊命令。
- 权限点照常登记（模板管理页 + 行打印按钮各一组）；租户闸门 fail-closed 语义不变。
- **种子模板**走 `pkg/constants/default_data.go` 的 count==0 播种定式：选不依赖业务域的样例
  （用户工牌、资产标签、通用单据各一），绑演示资源；worm 自带的采购收货单等业务模板作参考不直接搬。
- **无新增数据通道 = 无新增越权面**：模板渲染数据就是前端在既有权限下已可见的行数据；
  服务端 PDF 批量渲染时后端按当前用户上下文调用既有 List/Get 服务，不旁路权限。

## 5. 三端接入

dev 策略例外说明：本模块**内核三端同一份**（设计器交互一致性由同一内核保证，谈不上端间先后）；
管理页 / 绑定面板 / 打印按钮的 UX 仍以 react 为行为基准先行，双 Vue 跟进移植。

| 端 | 设计器画布 | 说明 |
|---|---|---|
| vue-element / vue-vben | `@go-wind/print-canvas` 组件直用 | 成熟路径：`<PrintDesigner :fields :initial-template @save @preview>` + `PrintHtmlPreview`；注意 canvas 的 style.css 全局引入一次 |
| react | 优先 `@go-wind/print-common`（Custom Element） | **M0 实测 react 19 + Custom Element**（property/事件桥接、Shadow DOM 与 antd6 样式互扰、卸载清理）；fork 内负责把它做到稳定（上游已有 PARITY.md 追踪 canvas/common 视差）；实测有硬伤则兜底自研 react 画布（引擎全部复用 core，只重写编辑器外壳） |

统一约定：

- **列表行「打印」**走 TableActions 定式（图标 + 文本；打印不入「更多」下拉，属于高频动作），
  按钮挂权限点；行数据直接作为 printData 喂模板。
- **打印预览抽屉**遵守 `docs/form_drawer_close_guard.md`（预览抽屉无表单脏数据，守卫按其"只读
  抽屉"档处理）；抽屉标题遵守三端统一拼句词条。
- **数据绑定面板**：资源选择器 → 读 `make ts` generated service 的字段 schema 生成字段树
  （含字段路径，如 `order.items[].sku`）；面板为各端原生 UI（design-language 管辖），设计器画布
  在 Shadow DOM 内不受三端样式体系管辖（隔离是特性：三端看到同一个设计器）。
- 管理页其余部分（列表/搜索/抽屉壳）完全按三端各自规范与 design-language 落地。
- i18n 三份 locales 按 `menu-i18n` 惯例对齐；菜单图标注意 vben 端 lucide 离线渲染约束。

## 6. 输出链

| 输出 | 阶段 | 实现 |
|---|---|---|
| 浏览器打印 / 另存 PDF | M2 | `PrintHtmlPreview`（ref.print()）浏览器原生；零新增服务端依赖 |
| 服务端 PDF + 批量合并 | M3 | Go 侧 chromedp（headless Chromium）渲染 `generateHtml` 产物；多行 → 多页合并单 PDF；渲染池 + 超时 + panic 兜底；产物走既有文件下发链路并记审计 |
| 静默打印 | M3 文档化 / M4 可选自研 | 先对接 worm 官方客户端二进制（协议开放，SDK 在 core/client）；采用者按部署文档自行装配；后续若自研 Electron 客户端照同一协议实现，对前端零改动 |

三端一致性约束（继承上游结论，写入部署文档）：像素级一致要求**字体与 Chromium 内核同源**——
服务端渲染容器需安装与客户端约定一致的字体集；不做此承诺时降级为"版式一致、像素不承诺"。

## 7. 差异化能力（M4，按序另立任务）

1. **AI 一句话生成模板**（采 openprint 点子、接本仓 AI 模块）：描述单据要素 → LLM 产出
   TemplateData → **schema 校验 + 字段白名单校验后**方可保存；走 AI_TOKENS 配额与白名单机制。
   这是相对竞品（含 worm 上游）的独有卖点，且实现条件本仓优于任何参照仓。
2. **脚本渲染前钩子**：脚本系统新增 hook 点类型（渲染前数据加工，如 `print_template.render`，
   可改写 printData）；与实体生命周期钩子并列登记，安全模型不变。
3. **套餐配额**：模板数纳入套餐配额四类体系（若运营上值得）。
4. 云打印队列 / 多门店远程打印（openprint 点子，仅存目）。

## 8. 分期计划与验收

| 阶段 | 内容 | 验收标准 |
|---|---|---|
| **M0 尖刺**（先行，约 1 周） | ① fork 建仓 + 构建链/CI/测试跑通；② 双 Vue 端 canvas 最小接入 demo；③ **react 19 + common 实测**（挂载/property/事件/Shadow DOM 隔离/卸载）；④ 浏览器打印实测（中文/条码二维码/分页）；⑤ 接管成本评估 → 定稿 fork 落点与包名 | 三端各一张样例单据从设计到打印闭环（mock 数据）；react 通道定案（common 或自研画布兜底） |
| **M1 模板域全栈** | §4 全部：实体 CRUD、种子模板、权限点、接口同步、菜单 | crud guide 造演示模块，后端链路全绿；三端 typecheck 门禁保持 0 错误 |
| **M2 三端接入 + 浏览器打印** | §5 全部：管理页、绑定面板、行打印、预览抽屉 | 三端各自对同一模板完成"设计→绑定→行打印"闭环；视觉过 design-language；暗/亮两模式实测 |
| **M3 输出链补全** | 服务端 PDF + 批量合并；静默打印对接文档化；部署文档补字体同源约束 | 批量 100 行合并 PDF 成功且权限按当前用户收敛；SSE 网关部署形态不受影响 |
| **M4 差异化** | §7 按序另立任务 | 各自验收另附 |

文档随行：本模块启用时补 `docs/crud_module_guide/` 「给你的模块接打印」一章；本文档同步实施状态。

## 9. 风险与对冲

| 风险 | 对冲 |
|---|---|
| 上游停更 / 上游带病版本 | **fork 接管即根除**；依赖精确版本钉死（与工具链钉死铁律同轨）；保留 upstream 同步窗口 |
| react 19 + Custom Element 不达标 | M0 即实测；兜底自研 react 画布（引擎复用 core，只重写编辑器外壳，成本可控且已获维护者授权"多实现代码"） |
| 套打精度诉求（发票/税控类毫米级） | 不承诺；标签/面单以连续纸 + 偏移微调覆盖；确有强诉求时再立专项 |
| 三端打印结果不一致 | 字体与 Chromium 同源写进部署文档；不承诺像素级时明确降级口径 |
| 任意 HTML 元素注入 | dompurify 消毒（fork 依赖审计清单跟踪）；AI 生成结果双重校验后才可保存 |
| 维护面 ×3 | 设计器单一内核；各端只维护管理页外壳（与全仓三端策略同构，无额外负担） |

## 10. 交叉引用

- 定位参照：`docs/adopt-one-frontend.md`（三端并列、采用者取其一——fork 包按端独立可装，core 共享）
- 接入定式：`docs/crud_module_guide/`、`docs/field_change_guide.md`、`docs/list_query_rule.md`
- 视觉与交互：`docs/design-language.md`（管理页外壳）、`docs/form_drawer_close_guard.md`（预览抽屉）
- 安全与租户：`docs/tenant_isolation.md`、`docs/frontend_authority.md`、`docs/plan_billing.md`（M4 配额）
- 扩展点：`docs/script_system.md`（M4 渲染钩子）、`docs/ai_module.md`（M4 AI 生成）
- 基座上游：https://github.com/worm-longliu/worm-vue3-print （MIT，fork 母本）
