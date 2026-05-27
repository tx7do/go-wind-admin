import type { DialogProps, DrawerProps, FormProps } from "element-plus";
import type { ProFormField } from "../ProForm/types";

/** 弹窗组件类型 */
export type ModalType = "dialog" | "drawer";
/** 弹窗模式：新增、编辑、查看 */
export type ModalMode = "add" | "edit" | "view";

/** ProModal 弹窗配置 */
export interface ProModalConfig<T = any> {
  /** 弹窗组件类型，默认 "dialog" */
  component?: ModalType;
  /** ElDialog 属性透传 */
  dialog?: Partial<Omit<DialogProps, "modelValue">>;
  /** ElDrawer 属性透传 */
  drawer?: Partial<Omit<DrawerProps, "modelValue">>;
  /** ElForm 属性透传 */
  form?: Partial<Omit<FormProps, "model" | "rules">>;
  /** 标签后是否显示冒号 */
  colon?: boolean;
  /** 行数据主键字段名 */
  rowKey?: string;
  /** 表单字段配置 */
  fields: ProFormField<T>[];
  /** 提交网络请求函数 */
  submitAction?: (data: T) => Promise<any>;
  /** 提交前处理回调 */
  beforeSubmit?: (data: T) => void;
}
