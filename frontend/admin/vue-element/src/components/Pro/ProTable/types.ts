import type { TableProps, ButtonProps } from "element-plus";

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
  | "custom";

export type TableColumnType = "default" | "selection" | "index" | "expand";

export type TableEngine = "vxe" | "element";

export interface ProTableColumn<T = any> {
  // 列类型
  type?: TableColumnType;
  // 标签
  label?: string;
  // 字段名
  prop?: keyof T & string;
  // 列宽
  width?: string | number;
  // 最小列宽
  minWidth?: string | number;
  // 固定列
  fixed?: "left" | "right" | boolean;
  // 对齐
  align?: "left" | "center" | "right";
  // 排序
  sortable?: boolean | "custom";
  // 列宽是否可拖动调整
  resizable?: boolean;
  // 是否显示
  show?: boolean;
  // 树节点列
  treeNode?: boolean;

  // === 单元格渲染类型 ===
  cellType?: CellType;
  // image 模板
  imageWidth?: number;
  imageHeight?: number;
  // tag 模板
  labelMap?: Record<string, any>;
  tagType?: string;
  // switch 模板
  activeValue?: boolean | string | number;
  inactiveValue?: boolean | string | number;
  activeText?: string;
  inactiveText?: string;
  // input 模板
  inputType?: string;
  // price 模板
  pricePrefix?: string;
  // date 模板
  dateFormat?: string;
  // tool 模板（操作列按钮）
  buttons?: Array<{
    name: string;
    text?: string;
    textKey?: string;
    perm?: string | string[];
    attrs?: Partial<ButtonProps>;
    visible?: (row: Record<string, any>) => boolean;
  }>;
  // 自定义插槽名
  slotName?: string;

  // 透传 el-table-column 原生属性
  attrs?: Record<string, any>;
  // 初始化函数
  initFn?: (item: Record<string, any>) => void;
  // 是否保留选中（跨页）
  reserveSelection?: boolean;
  // filter 值拼接符
  filterJoin?: string;

  [key: string]: any;
}

export interface ProTableProps<T = any> {
  columns: ProTableColumn<T>[];
  data?: T[];
  loading?: boolean;
  // 表格引擎
  engine?: TableEngine;
  rowKey?: string;
  // 透传表格属性（根据 engine 不同）
  table?: Record<string, any>;

  // === 分页 ===
  pagination?: boolean;
  total?: number;
  currentPage?: number;
  pageSize?: number;
  pageSizes?: number[];
}
