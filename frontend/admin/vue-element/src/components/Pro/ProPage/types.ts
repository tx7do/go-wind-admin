import type { ButtonProps } from "element-plus";
import type { DialogProps, DrawerProps } from "element-plus";
import type { ProFormField } from "../ProForm/types";
import type { ProTableColumn, TableEngine } from "../ProTable/types";

export type ToolbarLeft = "add" | "delete" | "import" | "export";
export type ToolbarRight = "refresh" | "filter" | "search" | "exports" | "imports" | "zoom";

export interface ToolsButton {
  name: string;
  label?: string;
  icon?: string;
  auth?: string | string[];
  attrs?: Partial<ButtonProps> & { style?: any };
  hidden?: boolean;
  disabled?: boolean;
  loading?: boolean;
  visible?: (...args: any[]) => boolean;
}

export type ListAction<TItem = any, TQuery = any> = (
  queryParams: TQuery & { page?: number; limit?: number }
) => Promise<{ items: TItem[]; total: number | string } | TItem[]>;

// 全局 ProPage 配置
export interface ProPageGlobalConfig {
  engine?: TableEngine;
  rowKey?: string;
  pagination?: {
    enabled?: boolean;
    pageSizes?: number[];
    pageSize?: number;
  };
  toolbar?: {
    defaultLeft?: Array<ToolbarLeft | ToolsButton>;
    defaultRight?: Array<ToolbarRight | ToolsButton>;
  };
  message?: {
    fetchSuccess?: boolean;
    fetchError?: boolean;
    deleteSuccess?: string;
    deleteConfirm?: string;
    submitSuccess?: string;
  };
}

// 骨架屏配置
export interface SkeletonConfig {
  /** 是否显示搜索区骨架 */
  showSearch?: boolean;
  /** 搜索字段数（默认 3） */
  searchFieldCount?: number;
  /** 是否显示选择列（默认 true） */
  showSelection?: boolean;
  /** 是否显示序号列（默认 true） */
  showIndex?: boolean;
  /** 骨架行数（默认 8） */
  rowCount?: number;
  /** 是否显示分页（默认 true） */
  showPagination?: boolean;
}

// 页面配置
export interface ProPageConfig<T = any, Q = any> {
  exportFilename?: string;
  engine?: TableEngine;
  rowKey?: string;
  tableId?: string;
  pageKey?: string;
  /** 首次加载时显示骨架屏（默认 false） */
  skeleton?: boolean | SkeletonConfig;

  search?: {
    fields?: ProFormField[];
    isExpandable?: boolean;
    showNumber?: number;
    colon?: boolean;
    grid?: boolean | "left" | "right";
  };

  table: {
    columns: ProTableColumn<T>[];
    tableAttrs?: Record<string, any>;
    /**
     * 空态引导按钮文案 i18n key。配置后 ProTable 空态将渲染 el-empty + 引导按钮，
     * 按钮点击触发 add 流程（打开新增抽屉 + emit add）。不配置则空态无按钮。
     */
    emptyActionText?: string;
    pagination?: boolean;
    toolbar?: Array<ToolbarLeft | ToolsButton>;
    toolbarRight?: Array<ToolbarLeft | ToolsButton>;
    defaultToolbar?: Array<ToolbarRight | ToolsButton>;
    listAction: ListAction<T, Q>;
    request?: { pageName: string; limitName: string };
    modifyAction?: (data: { [key: string]: any; field: string; value: any }) => Promise<any>;
    deleteAction?: (ids: string) => Promise<any>;
    /**
     * 服务端全量导出（XLSX，上限远大于客户端聚合）：提供后「导出全部」下拉
     * 追加「服务端全量」项；错误处理与提示由页面闭包自己负责（不吞错）。
     */
    exportServer?: () => Promise<void>;
    importsAction?: (data: any[]) => Promise<any>;
    importAction?: (file: File) => Promise<any>;
    importTemplate?: string | (() => Promise<any>);
  };

  modal?: {
    component?: "dialog" | "drawer";
    dialog?: Partial<Omit<DialogProps, "modelValue">>;
    drawer?: Partial<Omit<DrawerProps, "modelValue">>;
    form?: Record<string, any>;
    colon?: boolean;
    fields: ProFormField<T>[];
    beforeSubmit?: (data: T) => Promise<T> | T;
    submitAction?: (data: T) => Promise<any>;
    afterSubmit?: () => void;
  };
}
