import type { TableProps, ButtonProps } from "element-plus";

/** 单元格渲染类型 */
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

/** 列特殊类型（复选框、序号、展开行） */
export type TableColumnType = "default" | "selection" | "index" | "expand";

/** 表格渲染引擎 */
export type TableEngine = "vxe" | "element";

/** Pro 表格列配置，统一 vxe-table 和 el-table 的列定义 */
export interface ProTableColumn<T = any> {
  /** 列特殊类型（复选框、序号、展开行），设置后忽略 prop/label */
  type?: TableColumnType;
  /** 列标题 */
  label?: string;
  /** 数据字段名 */
  prop?: keyof T & string;
  /** 列宽度（像素值或字符串如 "auto"） */
  width?: string | number;
  /** 最小列宽 */
  minWidth?: string | number;
  /** 固定列位置 */
  fixed?: "left" | "right" | boolean;
  /** 对齐方式 */
  align?: "left" | "center" | "right";
  /** 排序，"custom" 表示服务端排序 */
  sortable?: boolean | "custom";
  /** 列宽是否可拖动调整（仅 vxe-table） */
  resizable?: boolean;
  /** 是否显示该列，默认 true */
  show?: boolean;
  /** 是否为树节点列（仅 vxe-table） */
  treeNode?: boolean;

  // === 单元格渲染类型 ===

  /** 单元格渲染类型，决定如何渲染该列内容 */
  cellType?: CellType;
  /** 图片宽度（像素），cellType: "image" 时生效，默认 40 */
  imageWidth?: number;
  /** 图片高度（像素），cellType: "image" 时生效，默认 40 */
  imageHeight?: number;
  /** 值到标签的映射表，cellType: "tag" 时生效 */
  labelMap?: Record<string, any>;
  /** 标签类型，cellType: "tag" 时生效 */
  tagType?: string;
  /** 开关激活值，cellType: "switch" 时生效，默认 1 */
  activeValue?: boolean | string | number;
  /** 开关非激活值，cellType: "switch" 时生效，默认 0 */
  inactiveValue?: boolean | string | number;
  /** 开关激活文本 */
  activeText?: string;
  /** 开关非激活文本 */
  inactiveText?: string;
  /** 输入框类型，cellType: "input" 时生效 */
  inputType?: string;
  /** 价格前缀，cellType: "price" 时生效，如 "¥" */
  pricePrefix?: string;
  /** 日期格式，cellType: "date" 时生效，默认 "YYYY-MM-DD HH:mm:ss" */
  dateFormat?: string;
  /** 操作列按钮配置，cellType: "tool" 时生效 */
  buttons?: Array<{
    /** 按钮标识，用于 operate 事件回调 */
    name: string;
    /** 按钮显示文本，不设则用 name */
    text?: string;
    /** 按钮文本国际化 key */
    textKey?: string;
    /** 权限标识（角色码或权限码），配合 AccessControl 组件控制按钮显隐 */
    auth?: string | string[];
    /** ElButton 属性 */
    attrs?: Partial<ButtonProps>;
    /** 动态显隐函数，返回 false 时隐藏 */
    visible?: (row: Record<string, any>) => boolean;
  }>;
  /** 自定义插槽名，cellType: "custom" 或省略时通过 prop/slotName 匹配插槽 */
  slotName?: string;

  /** 透传表格列原生属性（vxe-table 或 el-table 的 column 属性） */
  attrs?: Record<string, any>;
  /** 列配置初始化回调 */
  initFn?: (item: Record<string, any>) => void;
  /** 跨页保留选中状态（仅 selection 列有效） */
  reserveSelection?: boolean;
  /** filter 值拼接分隔符 */
  filterJoin?: string;

  /** 允许扩展任意属性 */
  [key: string]: any;
}

/** ProTable 组件 Props */
export interface ProTableProps<T = any> {
  /** 列配置 */
  columns: ProTableColumn<T>[];
  /** 静态数据（不传则由 ProPage 的 listAction 管理） */
  data?: T[];
  /** 加载状态 */
  loading?: boolean;
  /** 表格引擎，默认 "vxe" */
  engine?: TableEngine;
  /** 行数据主键字段名，默认 "id" */
  rowKey?: string;
  /** 表格唯一标识（用于 vxe-table 列配置持久化到 localStorage） */
  tableId?: string;
  /** 透传底层表格组件属性（vxe-table 或 el-table） */
  table?: Record<string, any>;

  // === 分页 ===
  /** 是否显示分页 */
  pagination?: boolean;
  /** 数据总条数 */
  total?: number;
  /** 当前页码 */
  currentPage?: number;
  /** 每页条数 */
  pageSize?: number;
  /** 可选的每页条数列表 */
  pageSizes?: number[];
}
