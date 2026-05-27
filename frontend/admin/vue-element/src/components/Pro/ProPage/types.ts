import type { DialogProps, DrawerProps } from "element-plus";
import type { ProFormField } from "../ProForm/types";
import type { ProTableColumn, TableEngine } from "../ProTable/types";

/** 左侧工具栏内置按钮类型 */
export type ToolbarLeft = "add" | "delete" | "import" | "export";
/** 右侧工具栏内置图标按钮类型 */
export type ToolbarRight = "refresh" | "filter" | "search" | "imports" | "exports" | "zoom";

/** ProPage 工具栏自定义按钮配置 */
export interface ToolsButton {
  /** 按钮唯一标识，用于 button-click 事件回调 */
  name: string;
  /** 按钮显示文本 */
  text?: string;
  /** 按钮文本国际化 key */
  textKey?: string;
  /** 权限标识（角色码或权限码），配合 AccessControl 组件控制按钮显隐 */
  auth?: string | string[];
  /** ElButton 属性透传 */
  attrs?: Record<string, any>;
  /** 动态显隐函数 */
  visible?: (row: Record<string, any>) => boolean;
}

/** 列表请求函数类型 */
export type ListAction<TItem = any, TQuery = any> = (
  queryParams: TQuery
) => Promise<{ items: TItem[]; total: number } | TItem[]>;

/**
 * ProPage 配置对象
 * 一个配置对象描述一个完整的 CRUD 页面（搜索 → 工具栏 → 表格 → 分页 → 弹窗）
 */
export interface ProPageConfig<T = any, Q = any> {
  /** 导出文件名（不含扩展名） */
  exportFilename?: string;
  /** 表格引擎，默认 "vxe" */
  engine?: TableEngine;
  /** 行数据主键字段名，默认 "id" */
  rowKey?: string;
  /** 表格唯一标识（默认从路由路径自动生成，用于 vxe-table 列配置持久化） */
  tableId?: string;

  // === 搜索配置 ===
  search?: {
    /** 搜索字段列表 */
    fields?: ProFormField[];
    /** 是否支持展开/收起，默认 true */
    isExpandable?: boolean;
    /** 默认显示字段数，默认 3 */
    showNumber?: number;
    /** 标签后是否显示冒号 */
    colon?: boolean;
    /** 自适应网格布局 */
    grid?: boolean | "left" | "right";
  };

  // === 表格配置 ===
  table: {
    /** 列配置列表（必填） */
    columns: ProTableColumn<T>[];
    /** 透传底层表格组件属性 */
    tableAttrs?: Record<string, any>;
    /** 是否分页，默认 true */
    pagination?: boolean;
    /** 左侧工具栏按钮（支持内置类型字符串或自定义按钮对象） */
    toolbar?: Array<ToolbarLeft | ToolsButton>;
    /** 右侧文本按钮 */
    toolbarRight?: Array<ToolbarLeft | ToolsButton>;
    /** 右侧图标按钮 + 自定义按钮 */
    defaultToolbar?: Array<ToolbarRight | ToolsButton>;
    /** 列表数据请求函数（必填） */
    listAction: ListAction<T, Q>;
    /** 分页参数名配置 */
    request?: { pageName: string; limitName: string };
    /** 行内属性修改回调（cellType: "switch" 触发） */
    modifyAction?: (data: { [key: string]: any; field: string; value: any }) => Promise<any>;
    /** 删除请求函数，接收逗号分隔的 ID 字符串 */
    deleteAction?: (ids: string) => Promise<any>;
    /** 客户端导出请求函数 */
    exportAction?: (queryParams: Q) => Promise<any>;
    /** 客户端导入请求函数 */
    importAction?: (file: File) => Promise<any>;
    /** 服务端全量导出请求函数，返回全部数据数组 */
    exportsAction?: (queryParams: Q) => Promise<any[]>;
    /** 服务端批量导入请求函数，接收解析后的数据数组 */
    importsAction?: (data: Record<string, any>[]) => Promise<any>;
    /** 导入模板 URL 或异步获取函数 */
    importTemplate?: string | (() => Promise<any>);
  };

  // === 弹窗配置 ===
  modal?: {
    /** 弹窗组件类型，默认 "dialog" */
    component?: "dialog" | "drawer";
    /** ElDialog 属性透传 */
    dialog?: Partial<Omit<DialogProps, "modelValue">>;
    /** ElDrawer 属性透传 */
    drawer?: Partial<Omit<DrawerProps, "modelValue">>;
    /** ElForm 属性透传 */
    form?: Record<string, any>;
    /** 标签后是否显示冒号 */
    colon?: boolean;
    /** 表单字段配置（必填） */
    fields: ProFormField<T>[];
    /** 提交前处理回调 */
    beforeSubmit?: (data: T) => void;
    /** 提交网络请求函数 */
    submitAction?: (data: T) => Promise<any>;
  };
}
