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

  // 右侧按钮
  rightButtons?: ToolbarButton[];

  // 默认工具栏（右侧图标按钮）
  defaultToolbar?: ("refresh" | "filter" | "search" | "exports" | "imports")[];

  // 权限前缀
  permPrefix?: string;

  // 是否显示
  visible?: boolean;

  // 自定义 class
  class?: string;
}

export interface ProToolbarEmits {
  "button-click": [name: string, data?: any];
  refresh: [];
  filter: [];
  search: [];
  export: [];
  import: [];
}
