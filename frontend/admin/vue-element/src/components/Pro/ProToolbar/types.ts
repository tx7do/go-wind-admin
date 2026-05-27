import type { ButtonProps } from "element-plus";

/** 工具栏按钮内置类型 */
export type ToolbarButtonType =
  | "add"
  | "delete"
  | "edit"
  | "view"
  | "export"
  | "import"
  | "refresh"
  | "filter"
  | "search"
  | "custom";

/** 工具栏按钮配置（左侧/右侧按钮通用） */
export interface ToolbarButton {
  /** 按钮唯一标识，用于 button-click 事件回调 */
  name: string;
  /** 按钮显示文本 */
  text?: string;
  /** 内置按钮类型 */
  type?: ToolbarButtonType;
  /** 图标组件名 */
  icon?: string;
  /** 权限标识（角色码或权限码），配合 AccessControl 组件控制按钮显隐 */
  auth?: string | string[];
  /** ElButton 属性透传 */
  attrs?: Partial<ButtonProps> & { style?: any };
  /** 是否始终隐藏 */
  hidden?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否加载中 */
  loading?: boolean;
  /** 动态显隐函数，返回 false 时隐藏 */
  visible?: (data?: any) => boolean;
}

/** ProToolbar 组件 Props */
export interface ProToolbarProps {
  /** 左侧操作按钮列表 */
  leftButtons?: ToolbarButton[];
  /** 右侧文本按钮列表（排列在内置图标按钮之前） */
  rightButtons?: ToolbarButton[];
  /** 右侧图标按钮 + 自定义按钮配置 */
  defaultToolbar?: Array<ToolbarRightType | ToolbarCustomButton>;
  /** 列配置（用于 filter 列显隐控制，仅 el-table 引擎使用） */
  columns?: Array<{ prop?: string; label?: string; show?: boolean }>;
  /** 是否显示工具栏 */
  visible?: boolean;
  /** 自定义 CSS 类名 */
  class?: string;
}

/** 内置工具栏按钮类型 */
export type ToolbarRightType = "refresh" | "filter" | "search" | "exports" | "imports" | "zoom";

/** 自定义工具栏按钮（插入到内置图标按钮区域） */
export interface ToolbarCustomButton {
  /** 按钮唯一标识 */
  name: string;
  /** 按钮显示文本 */
  text?: string;
  /** 图标组件名 */
  icon?: string;
  /** 权限标识（角色码或权限码），配合 AccessControl 组件控制按钮显隐 */
  auth?: string | string[];
  /** ElButton 属性透传 */
  attrs?: Partial<ButtonProps> & { style?: any };
  /** 是否始终隐藏 */
  hidden?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否加载中 */
  loading?: boolean;
  /** 动态显隐函数 */
  visible?: (data?: any) => boolean;
}

/** ProToolbar 组件事件 */
export interface ProToolbarEmits {
  /** 按钮点击事件，name 为按钮标识，data 为按钮对象 */
  "button-click": [name: string, data?: any];
  /** 刷新 */
  refresh: [];
  /** 筛选弹窗切换 */
  filter: [];
  /** 筛选列确认 */
  "filter-change": [columns: any[]];
  /** 搜索切换 */
  search: [];
  /** 导出 */
  export: [];
  /** 导入 */
  import: [];
  /** 全屏切换 */
  zoom: [isFullscreen: boolean];
}
