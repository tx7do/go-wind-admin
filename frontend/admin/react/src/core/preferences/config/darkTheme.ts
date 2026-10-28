import type { ThemeConfig } from 'antd';

/**
 * 暗黑模式扩展色板 —— 暗夜蓝黑大底 + gray 系一楼（2026-08-24 定稿）
 *
 *   L0 #0B0F19  地面：全局大背景（暗夜蓝黑，最暗）
 *   L1 #111827  一楼：搜索卡片 + 抽屉 + 表格容器（gray-900，比大底微亮）
 *   L2 #161F33  二楼：输入框/下拉框（比卡片微亮，"浮起"式，避免黑洞感）
 *   L3 #1F2937  表头独立次级深色；#1C2128 下拉/弹窗浮层
 *
 * 主色 hsl(212 100% 45%)（#006BE6，见 docs/design-language.md）；文字基色 #F8FAFC；placeholder #6E7681
 */
export const DARK_PALETTE = {
  bgLayout: '#0B0F19',
  bgNav: '#111827',
  bgSurface: '#111827',
  bgElevated: '#1C2128',
  bgInset: '#111827',
  borderInput: 'rgba(148, 163, 184, 0.22)',
  borderInputHover: 'rgba(148, 163, 184, 0.4)',
  // 卡片描边 = 分隔线 = §2.3 的 8% 白一档（下面 colorBorderSecondary 直接引用，不留第二份真相）。
  // 此前这里是 5% 白，正是 §2.3 点名"卡片与大底糊成一片"的档，且与 token 实际生效值不一致。
  borderCard: 'rgba(255, 255, 255, 0.08)',
  textPrimary: '#F8FAFC',
  textSecondary: '#8B949E',
  textTertiary: '#6E7681',
  // placeholder 取三级灰（2026-10-16 由 #9CA3AF 下调）：placeholder 的"显眼度"要按
  // 它与自己控件里已输入值的**分离度**判，不是按它对底色的对比度判。浅侧
  // #1F2937/#9CA3AF = 5.78 倍，暗侧同值只有 2.43 倍（16.96 vs 6.99），于是暗色下
  // 提示语和真内容一样跳。降到本档后分离度 4.39 倍，且复用已有文字档、不新增 hex。
  placeholder: '#6E7681',
  accentBlue: '#006BE6',
} as const;

/**
 * 暗黑模式扩展 token —— 叠加在 theme.darkAlgorithm 之上的精调
 * （用户 2026-08-24 定稿：暗夜蓝黑大底 + gray 一楼；主色 2026-09-08 起对齐设计语言规范）
 */
export const darkThemeTokens: ThemeConfig['token'] = {
  // 地面：最暗的暗夜蓝黑大底
  colorBgBase: '#0B0F19',
  colorBgLayout: '#0B0F19',
  // 一楼：卡片、抽屉、表格容器（比大底微亮）
  colorBgContainer: '#111827',
  // 浮层：下拉、弹窗
  colorBgElevated: '#1C2128',
  // 文字：明亮高级白为基，三阶灰拉开对比
  colorTextBase: '#F8FAFC',
  colorText: '#F8FAFC',
  colorTextSecondary: '#8B949E',
  colorTextTertiary: '#6E7681',
  colorTextPlaceholder: DARK_PALETTE.placeholder,
  // 边框：拒绝隐形。输入类控件边框用 slate 蓝灰（带蓝相，和整页蓝黑调一致，
  // 纯白 α 边框在蓝黑底上会显"脏灰"）；分隔线 8% 白保证卡片/表格
  // 与大底之间有可感知的层次（5% 时几乎隐形，整页糊成一片）
  colorBorder: 'rgba(148, 163, 184, 0.28)',
  colorBorderSecondary: DARK_PALETTE.borderCard,
  colorFill: 'rgba(255, 255, 255, 0.14)',
  colorFillSecondary: 'rgba(255, 255, 255, 0.09)',
  colorFillTertiary: 'rgba(255, 255, 255, 0.05)',
  colorFillQuaternary: 'rgba(255, 255, 255, 0.03)',
  // 浮层列表项 hover：默认派生自 colorFillTertiary（5% 白），在 #1C2128
  // 浮层上几乎不可察；提到 8% 让 Select/Dropdown/Menu 等悬停可感知，
  // 仍保持"暗底 + 细亮显形"的克制风格（不加发光）
  controlItemBgHover: 'rgba(255, 255, 255, 0.08)',
  // 浮层投影：暗色下必须用黑投影才有"浮起"感（antd v6 darkAlgorithm 派生的
  // 白色投影在暗底上不可见，弹层会像贴在页面上）
  boxShadowSecondary: '0 6px 16px rgba(0, 0, 0, 0.45), 0 3px 6px rgba(0, 0, 0, 0.3)',
  // 表单标签降一档亮度：让"值"成为视觉主角，标签退为辅助信息
  colorTextHeading: '#DCE3ED',
  // 卡片/弹窗/抽屉大圆角（对应规范"卡片 rounded-xl"）
  borderRadiusLG: 12,
};

/**
 * 暗黑模式组件级 token：
 * 输入类控件用更暗的内嵌底（#0A0F1C 蓝黑，与卡片对立）靠亮边框显形；
 * hover 边框提亮同色系 slate、focus 用主色边框 + 3px 柔光，状态反馈清晰；
 * Drawer 加深遮罩突出悬浮；Table 表头独立次级深色 + 主色行悬停/选中。
 * 注意：antd v6 已移除 .ant-select-selector/.ant-drawer-content，
 * 边框等需 CSS 打根/新层（见 pro-components-dark.css）。
 */
const inputLike = {
  // 输入类控件与表面同层（2026-09-08 对齐 vben 观感修订：vben 的输入框
  // 与卡片同底色、仅靠边框区分，实测观感更通透；此前的"浮起 lighter"
  // 与"内嵌更暗"两个方案先后被否）
  colorBgContainer: '#111827',
  colorBorder: 'rgba(255, 255, 255, 0.1)',
  hoverBorderColor: 'rgba(255, 255, 255, 0.2)',
  activeBorderColor: '#006BE6',
  activeShadow: '0 0 0 3px rgba(0, 107, 230, 0.12)',
};

export const darkThemeComponents: ThemeConfig['components'] = {
  Input: { ...inputLike },
  Select: { ...inputLike },
  InputNumber: { ...inputLike },
  DatePicker: { ...inputLike },
  TreeSelect: { ...inputLike },
  Drawer: {
    colorBgMask: 'rgba(0, 0, 0, 0.6)',
    paddingLG: 24,
  },
  Modal: {
    // 遮罩与 Drawer 同档（0.6）：弹窗是模态重交互面，背后的页面要压得更暗
    // 才衬得出悬浮层；antd 默认 0.45 在 L0 深色大底上几乎不可感
    colorBgMask: 'rgba(0, 0, 0, 0.6)',
  },
  Table: {
    // colorHeaderBg 非合法 antd Token（Table 表头色由 colorBgContainer/colorFillAlter 派生），原值 '#1F2937' 移除
    // 行悬停/选中用主色微底，行状态一眼可辨（默认派生的 5% 白在暗色下不可察）
    rowHoverBg: 'rgba(0, 107, 230, 0.08)',
    rowSelectedBg: 'rgba(0, 107, 230, 0.15)',
    rowSelectedHoverBg: 'rgba(0, 107, 230, 0.2)',
    headerSplitColor: 'transparent',
  },
  Menu: {
    // 展开的子菜单面板不铺底色（design-language.md §4「侧边栏菜单交互态」：
    // 「父级展开链路只做文字/图标提亮，不铺底色」）。
    // 必须在 token 层关掉、不能只删 CSS：出厂值 `darkSubMenuItemBg: '#000c17'`
    // （es/menu/style/index.js:449）比原先那条 rgba(0,0,0,.22) 的凹陷更黑，规则一撤就露出来。
    // 两个键都要写，因为它们喂的是**不同的 CSS 变量**（实测 cssVar 模式）：
    //   .ant-menu-dark（侧栏在暗色下走这套，SiderMenu/index.tsx 的 theme=isDark?'dark':'light'）
    //     → --ant-menu-dark-sub-menu-item-bg ← darkSubMenuItemBg
    //   .ant-menu-light（暗色应用里未显式指定 theme 的 Menu）
    //     → --ant-menu-sub-menu-item-bg      ← subMenuItemBg
    // 只改后者时暗色侧栏纹丝不动（#000c17 依旧），这就是本条注释存在的理由。
    darkSubMenuItemBg: 'transparent',
    subMenuItemBg: 'transparent',
    // 折叠后弹出的子菜单浮层是「浮层」，取 §2.3 的 #1C2128，不是 antd 出厂藏青 #001529。
    // 根 UL 有 SiderMenu/index.tsx:229 的 inline background:'transparent' 兜着，portal 里这一层没有。
    // 只补这一个键：实测弹层容器底读 --ant-menu-dark-popup-bg ← darkPopupBg；
    // darkItemBg → itemBg 只喂根 UL（已被 inline 盖住），动它属无据扩面。
    darkPopupBg: '#1C2128',
  },
};
