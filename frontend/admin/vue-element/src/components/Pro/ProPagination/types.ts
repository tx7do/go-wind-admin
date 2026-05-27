import type { PaginationProps } from "element-plus";

/** ProPagination 分页组件 Props */
export interface ProPaginationProps extends Partial<PaginationProps> {
  /** 分页状态（currentPage + pageSize） */
  modelValue?: {
    currentPage?: number;
    pageSize?: number;
  };
  /** 数据总条数 */
  total?: number;
  /** 是否显示总条数 */
  showTotal?: boolean;
  /** 是否显示每页条数选择器 */
  showSizes?: boolean;
  /** 是否显示跳转输入框 */
  showJump?: boolean;
  /** 分页布局模板 */
  layout?: string;
  /** 可选的每页条数列表 */
  pageSizes?: number[];
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否显示按钮背景色 */
  background?: boolean;
  /** 数据只有一页时是否隐藏 */
  hideOnSinglePage?: boolean;
}

/** ProPagination 组件事件 */
export interface PaginationEmits {
  /** 分页状态变更（currentPage + pageSize） */
  "update:modelValue": [value: { currentPage: number; pageSize: number }];
  /** 页码变更 */
  "current-change": [currentPage: number];
  /** 每页条数变更 */
  "size-change": [pageSize: number];
  /** 点击上一页 */
  "prev-click": [currentPage: number];
  /** 点击下一页 */
  "next-click": [currentPage: number];
}
