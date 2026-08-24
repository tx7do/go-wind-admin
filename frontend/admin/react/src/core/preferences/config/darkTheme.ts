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
  bgInset: '#090D16',
  borderInput: 'rgba(255, 255, 255, 0.15)',
  borderInputHover: 'rgba(255, 255, 255, 0.3)',
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
  // 边框：拒绝隐形，输入框精致浅色边框
  colorBorder: 'rgba(255, 255, 255, 0.15)',
  colorBorderSecondary: 'rgba(255, 255, 255, 0.05)',
  colorFill: 'rgba(255, 255, 255, 0.14)',
  colorFillSecondary: 'rgba(255, 255, 255, 0.09)',
  colorFillTertiary: 'rgba(255, 255, 255, 0.05)',
  colorFillQuaternary: 'rgba(255, 255, 255, 0.03)',
  // 浮层列表项 hover：默认派生自 colorFillTertiary（5% 白），在 #1C2128
  // 浮层上几乎不可察；提到 8% 让 Select/Dropdown/Menu 等悬停可感知，
  // 仍保持"暗底 + 细亮显形"的克制风格（不加发光）
  controlItemBgHover: 'rgba(255, 255, 255, 0.08)',
};

/**
 * 暗黑模式组件级 token：
 * 输入类控件用更暗的内嵌底（#090D16）与卡片对立，靠亮边框显形；
 * Drawer 加深遮罩突出悬浮；Table 表头独立次级深色。
 * 注意：antd v6 已移除 .ant-select-selector/.ant-drawer-content，
 * 边框等需 CSS 打根/新层（见 pro-components-dark.css）。
 */
export const darkThemeComponents: ThemeConfig['components'] = {
  Input: { colorBgContainer: '#090D16', borderRadius: 6 },
  Select: { colorBgContainer: '#090D16', borderRadius: 6 },
  InputNumber: { colorBgContainer: '#090D16', borderRadius: 6 },
  DatePicker: { colorBgContainer: '#090D16', borderRadius: 6 },
  TreeSelect: { colorBgContainer: '#090D16', borderRadius: 6 },
  Drawer: {
    colorBgMask: 'rgba(0, 0, 0, 0.6)',
    paddingLG: 24,
  },
  Table: {
    colorHeaderBg: '#1F2937',
  },
};
