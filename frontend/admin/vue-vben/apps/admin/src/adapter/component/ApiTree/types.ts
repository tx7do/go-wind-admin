import type { AnyPromiseFunction } from '@vben/types';

export type OptionsItem = {
  [name: string]: any;
  children?: OptionsItem[];
  disabled?: boolean;
  label?: string;
  value?: string;
};

export interface Props {
  title?: string;
  toolbar?: boolean;
  checkable?: boolean;

  search?: boolean;
  searchText?: string;

  /** 是否将value从数字转为string */
  numberToString?: boolean;
  /** 获取options数据的函数 */
  api?: (arg?: any) => Promise<OptionsItem[] | Record<string, any>>;
  /** 传递给api的参数 */
  params?: Record<string, any>;
  /** 从api返回的结果中提取options数组的字段名 */
  resultField?: string;
  /** label字段名 */
  labelField?: string;
  /** children字段名，需要层级数据的组件可用 */
  childrenField?: string;
  /** value字段名 */
  valueField?: string;
  /** 组件接收options数据的属性名 */
  optionsPropName?: string;
  /** 是否立即调用api */
  immediate?: boolean;
  /** 每次`visibleEvent`事件发生时都重新请求数据 */
  alwaysLoad?: boolean;
  /** 在api请求之前的回调函数 */
  beforeFetch?: AnyPromiseFunction<any, any>;
  /** 在api请求之后的回调函数 */
  afterFetch?: AnyPromiseFunction<any, any>;
  /** 直接传入选项数据，也作为api返回空数据时的后备数据 */
  options?: OptionsItem[];
  /** 组件的插槽名称，用来显示一个"加载中"的图标 */
  loadingSlot?: string;
  /** 触发api请求的事件名 */
  visibleEvent?: string;
  /** 组件的v-model属性名，默认为modelValue。部分组件可能为value */
  modelPropName?: string;
  /** 树形组件是否默认展开所有节点 */
  treeDefaultExpandAll?: boolean;
}

export type TreeEmits = {
  optionsChange: [OptionsItem[]];
  search: [string];
};

export enum ToolbarEnum {
  SELECT_ALL,
  UN_SELECT_ALL,
  EXPAND_ALL,
  UN_EXPAND_ALL,
  CHECK_STRICTLY,
  CHECK_UN_STRICTLY,
}

export interface MenuInfo {
  key: ToolbarEnum;
}
