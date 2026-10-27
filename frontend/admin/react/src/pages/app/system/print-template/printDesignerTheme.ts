/**
 * go-wind-print 设计器暗色主题（宿主侧实现，v1.3.11）。
 *
 * 内核把亮色值定义在 shadow 内的 .designer-container 上（scoped 属性压过宿主继承的
 * CSS 变量），所以暗色 = 向 shadowRoot 注入一段变量覆盖块（!important 压过定义）。
 * 该实现与钉死的 1.3.11 类名耦合；fork 内核后续应加正式的 data-theme 块替代（记
 * docs/print_template.md）。
 *
 * 取值对齐 react 暗色六轮定稿（#0B0F19/#111827 家族）；画纸 --pd-paper 保持白底
 * ——设计工具的"画布恒白"惯例（Photoshop/Word 同款），暗色只作用于周边铬件。
 */
const DARK_VAR_BLOCK = `
.designer-container {
  --pd-bg: #0b0f19 !important;
  --pd-surface: #111827 !important;
  --pd-sidebar: #111827 !important;
  --pd-sidebar-hover: #1f2937 !important;
  --pd-field-bg: #0d1420 !important;
  --pd-input-bg: #0d1420 !important;
  --pd-hover-tint: rgba(255, 255, 255, 0.06) !important;
  --pd-canvas-bg: #0a0e17 !important;
  --pd-text: #e6e8ef !important;
  --pd-text-regular: #d1d5db !important;
  --pd-text-muted: #9aa3b2 !important;
  --pd-text-faint: #6b7280 !important;
  --pd-border: #2a3444 !important;
  --pd-border-soft: #232c3a !important;
  --pd-accent: #4a8dff !important;
  --pd-accent-soft: rgba(74, 141, 255, 0.16) !important;
  --pd-accent-2: #7c9dff !important;
  --pd-accent-secondary: #7c9dff !important;
  --pd-danger: #ff6b6b !important;
  --pd-ink-green: #4ade80 !important;
  --pd-ruler-bg: #0d1420 !important;
  --pd-ruler: #2a3444 !important;
  --pd-ruler-cursor: #f87171 !important;
  --pd-paper: #ffffff !important;
  --pd-shadow-paper: rgba(0, 0, 0, 0.55) !important;
  --pd-scrollbar: transparent !important;
  --pd-scrollbar-thumb: #2a3444 !important;
  --pd-scrollbar-thumb-hover: #3a4557 !important;
  --pd-menu-text: #e6e8ef !important;
  --pd-menu-muted: #9aa3b2 !important;
  --pd-menu-hover: #1f2937 !important;
  --pd-menu-divider: #232c3a !important;
  --pd-menu-disabled: #4b5563 !important;
  --pd-menu-border: #232c3a !important;
  --pd-help-variable-ink: #7dd3fc !important;
  --pd-help-variable-line: rgba(125, 211, 252, 0.4) !important;
  --pd-help-variable-bg: rgba(125, 211, 252, 0.08) !important;
  --pd-help-function-ink: #c4b5fd !important;
  --pd-help-function-line: rgba(196, 181, 253, 0.4) !important;
  --pd-help-function-bg: rgba(196, 181, 253, 0.08) !important;
  --pd-help-field-ink: #86efac !important;
  --pd-help-field-line: rgba(134, 239, 172, 0.4) !important;
  --pd-help-field-bg: rgba(134, 239, 172, 0.08) !important;
  --pd-shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.5) !important;
  --pd-shadow-md: 0 4px 12px rgba(0, 0, 0, 0.5) !important;
}
`;

const STYLE_ID = 'pd-host-dark-theme';

/** 在自定义元素的 shadowRoot 内应用/清除暗色变量块 */
export function applyPrintDesignerTheme(el: HTMLElement | null, isDark: boolean) {
  const root = (el as any)?.shadowRoot as ShadowRoot | null | undefined;
  if (!root) return;
  let style = root.getElementById?.(STYLE_ID) ?? null;
  if (!isDark) {
    style?.remove();
    return;
  }
  if (!style) {
    style = document.createElement('style');
    style.id = STYLE_ID;
    root.appendChild(style);
  }
  style.textContent = DARK_VAR_BLOCK;
}
