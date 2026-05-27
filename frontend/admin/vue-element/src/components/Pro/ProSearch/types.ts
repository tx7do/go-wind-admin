import type { FormProps, CardProps } from "element-plus";
import type { ProFormField } from "../ProForm/types";

/** ProSearch 搜索栏配置 */
export interface ProSearchConfig<T = any> {
  /** 标签后是否显示冒号 */
  colon?: boolean;

  /** 自适应网格布局，true 为均分，"left"/"right" 为对齐方向 */
  grid?: boolean | "left" | "right";
  /** 行内表单模式 */
  inline?: boolean;

  /** 是否支持展开/收起，默认 true */
  isExpandable?: boolean;
  /** 默认显示字段数，默认 3 */
  showNumber?: number;

  /** 卡片容器属性 */
  cardAttrs?: Partial<CardProps>;

  /** 表单属性透传 */
  form?: Partial<FormProps>;

  /** 搜索字段配置列表 */
  fields: ProFormField<T>[];

  /** 是否显示搜索按钮，默认 true */
  showSearchButton?: boolean;
  /** 是否显示重置按钮，默认 true */
  showResetButton?: boolean;
  /** 搜索按钮文本 */
  searchButtonText?: string;
  /** 重置按钮文本 */
  resetButtonText?: string;

  /** 搜索按钮权限标识 */
  searchAuth?: string;
  /** 重置按钮权限标识 */
  resetAuth?: string;
}

/** ProSearch 组件事件 */
export interface ProSearchEmits<T = any> {
  /** 搜索提交 */
  search: [queryParams: T];
  /** 重置搜索 */
  reset: [queryParams: T];
  /** 展开/收起切换 */
  expand: [expanded: boolean];
}
