import type { ColProps, FormItemRule } from "element-plus";

/** 表单组件类型枚举 */
export type FormValueType =
  | "input"
  | "textarea"
  | "select"
  | "radio"
  | "checkbox"
  | "switch"
  | "date-picker"
  | "time-picker"
  | "time-select"
  | "input-number"
  | "cascader"
  | "tree-select"
  | "api-tree-select"
  | "input-tag"
  | "custom-tag"
  | "icon-select"
  | "number"
  | "date"
  | "custom";

/** Pro 表单字段配置，被 ProForm、ProSearch、ProModal 共用 */
export interface ProFormField<T = any> {
  /** 组件类型，默认 "input" */
  type?: FormValueType;
  /** 标签文本 */
  label: string;
  /** 字段名，对应数据对象的 key */
  field: keyof T & string;
  /** 标签提示文字或提示配置，显示在标签旁的 Tooltip 中 */
  tips?: string | Record<string, any>;
  /** 组件属性，透传给底层组件（如 placeholder、clearable 等） */
  attrs?: Record<string, any>;
  /** 选项列表，用于 select / radio / checkbox 等选择型组件 */
  options?: { label: string; value: any; [key: string]: any }[];
  /** 表单校验规则 */
  rules?: FormItemRule[];
  /** 初始值 */
  initialValue?: any;
  /** 自定义插槽名，type 为 "custom" 时生效 */
  slotName?: string;
  /** 是否隐藏该字段 */
  hidden?: boolean;
  /** 栅格占位列数，默认 24（整行） */
  span?: number;
  /** Element Plus Col 属性，优先级高于 span */
  col?: Partial<ColProps>;
  /** 组件事件监听器 */
  events?: Record<string, (...args: any) => void>;
  /** 字段初始化回调 */
  initFn?: (item: Record<string, any>) => void;
  /** 异步数据源，用于 api-tree-select 类型 */
  api?: () => Promise<any[]>;
}
