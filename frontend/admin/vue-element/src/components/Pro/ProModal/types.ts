import type { DialogProps, DrawerProps, FormProps } from "element-plus";
import type { ProFormField } from "../ProForm/types";

export type ModalType = "dialog" | "drawer";
export type ModalMode = "add" | "edit" | "view";

export interface ProModalConfig<T = any> {
  // 权限前缀
  permPrefix?: string;
  // 组件类型
  component?: ModalType;
  // dialog 属性
  dialog?: Partial<Omit<DialogProps, "modelValue">>;
  // drawer 属性
  drawer?: Partial<Omit<DrawerProps, "modelValue">>;
  // form 属性
  form?: Partial<Omit<FormProps, "model" | "rules">>;
  // 标签冒号
  colon?: boolean;
  // 主键名
  rowKey?: string;
  // 表单字段
  fields: ProFormField<T>[];
  // 提交网络请求
  submitAction?: (data: T) => Promise<any>;
  // 提交前处理
  beforeSubmit?: (data: T) => void;
}
