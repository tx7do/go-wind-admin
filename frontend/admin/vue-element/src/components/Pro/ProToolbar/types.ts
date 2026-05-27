import type { ButtonProps } from "element-plus";

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

export interface ToolbarButton {
  name: string;
  text?: string;
  type?: ToolbarButtonType;
  icon?: string;
  perm?: string | string[];
  attrs?: Partial<ButtonProps> & { style?: any };
  hidden?: boolean;
  disabled?: boolean;
  loading?: boolean;
  visible?: (data?: any) => boolean;
}

export interface ProToolbarProps {
  // 左侧按钮
  leftButtons?: ToolbarButton[];

  // 右侧按钮（在内置图标按钮之前）
  rightButtons?: ToolbarButton[];

  // 默认工具栏（右侧图标按钮 + 自定义按钮）
  defaultToolbar?: Array<ToolbarRightType | ToolbarCustomButton>;

  // 列配置（用于 filter 列显隐，仅 el-table 引擎使用）
  columns?: Array<{ prop?: string; label?: string; show?: boolean }>;

  // 权限前缀
  permPrefix?: string;

  // 是否显示
  visible?: boolean;

  // 自定义 class
  class?: string;
}

/** 内置工具栏按钮类型 */
export type ToolbarRightType = "refresh" | "filter" | "search" | "exports" | "imports" | "zoom";

/** 自定义工具栏按钮（插入到内置图标按钮区域） */
export interface ToolbarCustomButton {
  name: string;
  text?: string;
  icon?: string;
  perm?: string | string[];
  attrs?: Partial<ButtonProps> & { style?: any };
  hidden?: boolean;
  disabled?: boolean;
  loading?: boolean;
  visible?: (data?: any) => boolean;
}

export interface ProToolbarEmits {
  "button-click": [name: string, data?: any];
  refresh: [];
  filter: [];
  search: [];
  export: [];
  import: [];
  zoom: [isFullscreen: boolean];
}
