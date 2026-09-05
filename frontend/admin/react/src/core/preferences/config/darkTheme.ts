import type { ThemeConfig } from 'antd';

/**
 * 暗黑模式扩展色板 —— 暗夜蓝黑大底 + gray 系一楼（2026-08-24 定稿）
 *
 *   L0 #0B0F19  地面：全局大背景（暗夜蓝黑，最暗）
 *   L1 #111827  一楼：搜索卡片 + 抽屉 + 表格容器（gray-900，比大底微亮）
 *   L2 #090D16  二楼：输入框/下拉框（更暗，与卡片对立 + 亮边框显形）
 *   L3 #1F2937  表头独立次级深色；#1C2128 下拉/弹窗浮层
 *
 * 主色 #3B82F6（科技蓝）；文字基色 #F8FAFC；placeholder #64748B
 */
export const DARK_PALETTE = {
  bgLayout: '#0B0F19',
  bgNav: '#111827',
  bgSurface: '#111827',
  bgElevated: '#1C2128',
  bgInset: '#0A0F1C',
  borderInput: 'rgba(148, 163, 184, 0.28)',
  borderInputHover: 'rgba(148, 163, 184, 0.45)',
  borderCard: 'rgba(255, 255, 255, 0.05)',
  borderLine: 'rgba(255, 255, 255, 0.06)',
  textPrimary: '#F8FAFC',
  textSecondary: '#8B949E',
  textTertiary: '#6E7681',
  placeholder: '#64748B',
  accentBlue: '#3B82F6',
} as const;

/**
 * 暗黑模式扩展 token —— 叠加在 theme.darkAlgorithm 之上的精调
 * （用户 2026-08-24 定稿：暗夜蓝黑大底 + gray 一楼 + 科技蓝主色）
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
  colorTextPlaceholder: '#64748B',
  // 边框：拒绝隐形。输入类控件边框用 slate 蓝灰（带蓝相，和整页蓝黑调一致，
  // 纯白 α 边框在蓝黑底上会显"脏灰"）；分隔线 8% 白保证卡片/表格
  // 与大底之间有可感知的层次（5% 时几乎隐形，整页糊成一片）
  colorBorder: 'rgba(148, 163, 184, 0.28)',
  colorBorderSecondary: 'rgba(255, 255, 255, 0.08)',
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
  colorBgContainer: '#0A0F1C',
  borderRadius: 6,
  hoverBorderColor: 'rgba(148, 163, 184, 0.45)',
  activeBorderColor: '#3B82F6',
  activeShadow: '0 0 0 3px rgba(59, 130, 246, 0.12)',
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
  Table: {
    colorHeaderBg: '#1F2937',
    // 行悬停/选中用主色微底，行状态一眼可辨（默认派生的 5% 白在暗色下不可察）
    rowHoverBg: 'rgba(59, 130, 246, 0.08)',
    rowSelectedBg: 'rgba(59, 130, 246, 0.15)',
    rowSelectedHoverBg: 'rgba(59, 130, 246, 0.2)',
    headerSplitColor: 'transparent',
  },
};
