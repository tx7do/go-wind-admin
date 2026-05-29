import type { ProTableConfig } from "./types";
import { DEFAULT_PAGE_SIZE, DEFAULT_PAGE_SIZES } from "@/components/Pro";

export const proTableGlobalConfig: ProTableConfig = {
  engine: "vxe",
  rowKey: "id",
  // 行高由全局 _vxe-table.scss 通过 padding 控制（表项 ~30px、表头 ~30px）
  // 此处保留字段用于 el-table 引擎兼容
  rowHeight: 30,
  headerHeight: 30,
  border: "none",
  stripe: true,
  borderRadius: 6,
  fontSize: 13,
  emptyText: "暂无数据",

  pagination: {
    show: true,
    pageSizes: DEFAULT_PAGE_SIZES,
    pageSize: DEFAULT_PAGE_SIZE,
    layout: "total, sizes -> prev, pager, next",
    background: true,
  },

  column: {
    align: "center",
    resizable: true,
    showOverflowTooltip: true,
    sortable: false,
  },

  toolButton: {
    size: "small",
    link: true,
    iconSize: 16,
  },
};

export function mergeTableConfig(
  global: ProTableConfig,
  component?: Partial<ProTableConfig>
): ProTableConfig {
  return {
    ...global,
    ...component,
    pagination: { ...global.pagination, ...component?.pagination },
    column: { ...global.column, ...component?.column },
    toolButton: { ...global.toolButton, ...component?.toolButton },
  };
}
