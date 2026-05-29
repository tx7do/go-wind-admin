import type { ButtonProps } from "element-plus";

// 全局表格配置
export interface ProTableConfig {
  engine?: "vxe" | "element";
  rowKey?: string;
  rowHeight?: number;
  headerHeight?: number;
  border?: boolean | "outer" | "inner" | "full" | "default" | "none";
  stripe?: boolean;
  borderRadius?: number;
  fontSize?: number;
  emptyText?: string;

  pagination?: {
    show?: boolean;
    pageSizes?: number[];
    pageSize?: number;
    layout?: string;
    background?: boolean;
  };

  toolButton?: {
    size?: "default" | "small" | "large";
    link?: boolean;
    iconSize?: number;
  };

  column?: {
    align?: "left" | "center" | "right";
    resizable?: boolean;
    showOverflowTooltip?: boolean;
    sortable?: boolean;
  };
}

export type CellType =
  | "text"
  | "image"
  | "tag"
  | "switch"
  | "input"
  | "date"
  | "link"
  | "price"
  | "percent"
  | "icon"
  | "tool"
  | "custom"
  | (string & {});

export type TableColumnType = "default" | "selection" | "index" | "expand";
export type TableEngine = "vxe" | "element";

export interface ProTableColumn<T = any> {
  type?: TableColumnType;
  label?: string;
  prop?: keyof T & string;
  width?: string | number;
  minWidth?: string | number;
  fixed?: "left" | "right" | boolean;
  align?: "left" | "center" | "right";
  sortable?: boolean | "custom";
  resizable?: boolean;
  show?: boolean;
  treeNode?: boolean;

  cellStyle?: Record<string, any>;
  headerStyle?: Record<string, any>;
  className?: string;
  headerClassName?: string;

  cellType?: CellType;
  imageWidth?: number;
  imageHeight?: number;
  labelMap?: Record<string, any>;
  tagType?: string;
  activeValue?: any;
  inactiveValue?: any;
  activeText?: string;
  inactiveText?: string;
  inputType?: string;
  pricePrefix?: string;
  dateFormat?: string;

  formatter?: (row: T, col: ProTableColumn<T>) => any;

  buttons?: Array<{
    name: string;
    label?: string;
    icon?: string;
    auth?: string | string[];
    attrs?: Partial<ButtonProps>;
    visible?: (row: T) => boolean;
  }>;

  slotName?: string;
  attrs?: Record<string, any>;
  initFn?: (item: any) => void;
  reserveSelection?: boolean;
  filterJoin?: string;

  [key: string]: any;
}

export interface ProTableProps<T = any> {
  columns: ProTableColumn<T>[];
  data?: T[];
  loading?: boolean;
  engine?: TableEngine;
  rowKey?: string;
  tableId?: string;
  table?: Record<string, any>;
  config?: Partial<ProTableConfig>;

  pagination?: boolean;
  total?: number;
  currentPage?: number;
  pageSize?: number;
  pageSizes?: number[];
}
