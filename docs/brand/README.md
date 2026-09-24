# GoWind 风行 · 品牌视觉标识

品牌主名 **GoWind**，中文名 **风行**。图形主题为「风」：速度、流动、汇聚。
Admin 是产品线之一（另有 CMS / UBA / IM / Quant 量化等），因此采用**主品牌标识 + 产品名文字后缀**的架构：图形标与字标属于 GoWind 风行，产品不单独造标。

> **✅ 2026-09-10 已选定方案 B「旋涡 Vortex」并落地三端**（见文末落地记录）。可视化对比仍见 [preview.html](./preview.html)。

## 色板（与 docs/design-language.md 同源）

| 用途 | 值 |
|---|---|
| 品牌主色（权威值） | `hsl(212 100% 45%)` ≈ `#006BE6` |
| 渐变亮端 | `#70B3FF` |
| 渐变中段 | `#0D74F2` |
| 渐变暗端 | `#005AC2` |
| 应用图标底渐变 | `#1E7BFF` → `#0053BD`（135°） |
| 中文名「风行」 | `#0D74F2`（实色） |

## 文件清单

| 方案 | 图形标 | 应用图标 | favicon | 横版组合 |
|---|---|---|---|---|
| A 阵风 Gust | `gust.svg` | `gust-tile.svg` | `gust-favicon.svg` | `gust-lockup.svg` |
| B 旋涡 Vortex ✅ **已选定 v3** | `vortex.svg` | `vortex-tile.svg` | `vortex-favicon.svg` | `vortex-lockup.svg` |
| C W 字标 Monogram | `wmark.svg` | `wmark-tile.svg` | `wmark-favicon.svg` | `wmark-lockup.svg` |

> v3 几何（2026-09-10 二次迭代）：双粗臂涡旋（外臂 r176/宽 84 扫 295°，内臂 r58/宽 54 扫 235°，缺口错位 ~125° 形成 S 形负空间气道），替代原三细弧版——16px favicon 与 32px 侧栏下辨识度显著更高。

公共资产：`wordmark.svg`（GOWIND + 风行 两行式几何字标）、`preview.html`、`vortex-login.svg`（登录页品牌插画 v2：旋涡主标 **24s/圈 缓旋**（SMIL animateTransform，以字形局部坐标为轴）+ 双层对旋虚线环流 + 轨道光点 + 四向漂移风痕 + GOWIND 几何字标签名；CSS 动效尊重 prefers-reduced-motion；已入三端代码）、`brand-render.html`（PNG 导出画布，改资产后可经 Playwright 截图重新生成）。

## 落地记录（2026-09-10，方案 B）

| 资产 | 目标 |
|---|---|
| `logo.png`（tile 200×200 透明底） | `react/public/`、`vue-element/public/`（补齐原 404）、`vue-element/src/assets/images/`（登录页头部 24px + 侧栏 LayoutLogo 32px）、`vue-vben/apps/admin/public/`（侧栏 + 认证页经 `preferences.logo.source`） |
| `favicon.ico`（16/32/48 多尺寸） | 三端 `public/favicon.ico` |
| `pwa-icon-192/512.png` | `vue-vben/apps/admin/public/` |
| `vortex-login.svg` | 登录页品牌插画，三端同款：`vue-element/src/pages/core/login/icons/slogan.vue`、`react/src/components/bussiness/AuthLayout/icons/SloganIcon.tsx`（AuthLayout + UserLayout 共用）、`vue-vben/packages/effects/layouts/src/authentication/icons/slogan.vue`（替换原库存占位插画 87667-SVG8，float 动效保留） |

vue-element / react / vben 三端 typecheck 门禁全部通过（2026-09-10）。PNG 导出管线：`brand-render.html` + Playwright 截图（omitBackground 保透明）+ Pillow 打包 ICO。

## 使用规则

- 图形标为渐变描边，浅底/深底通用，无需出双版本；禁拉伸、禁改色相、禁加投影描边。
- 字标（GOWIND + 风行）已全部转路径，不依赖字体，任何环境渲染一致。其中「风行」二字取自 **Noto Sans SC Bold**（SIL OFL 授权，可免费商用与嵌入）轮廓，经 fontTools 提取转路径。
- `*-tile.svg`：512×512、圆角 116（≈22.6%），用于桌面/移动应用图标与产品列表头像；缩放即可。
- `*-favicon.svg`：48 视窗简化形，16px 仍可辨识；生产 `favicon.ico` 由它导出 32/16 位 PNG 后打包。
- 最小留白：图形标四周预留 ≥ 1/4 高度的净空。
- 产品线命名：「GoWind + 产品名」（GoWind Admin / GoWind CMS / GoWind UBA / GoWind IM / GoWind Quant），图形标不变，产品名用系统字体作后缀。

## 落点核对（已落地，改资产时按这几处回归）

- react：`frontend/admin/react/public/logo.png`（200×200，由 tile SVG 导出 PNG）+ `src/core/preferences/config/default.ts:52-54` 的 `logo.source`（实测值为 `/logo.png`）
- favicon：三端 `public/favicon.ico`（react / vue-element / vue-vben）
- 登录页品牌区：按 `docs/design-language.md` 认证页规范（画布深底 + 表面卡 + float 动效）
