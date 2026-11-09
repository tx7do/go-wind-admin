<template>
  <ElDropdown @command="handleCommand">
    <ElButton :loading="exporting" class="mr-1 whitespace-nowrap">
      {{ t("pages.curd.export.exportAll") }}
    </ElButton>
    <template #dropdown>
      <ElDropdownMenu>
        <ElDropdownItem command="csv">
          {{ t("pages.curd.export.csv") }}
        </ElDropdownItem>
        <ElDropdownItem command="xlsx">
          {{ t("pages.curd.export.xlsx") }}
        </ElDropdownItem>
        <ElDropdownItem v-if="serverExport" command="server-xlsx" divided>
          {{ t("pages.curd.export.serverXlsx") }}
        </ElDropdownItem>
      </ElDropdownMenu>
    </template>
  </ElDropdown>
</template>

<script setup lang="ts">
import { ref } from "vue";
import {
  ElButton,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElMessage,
} from "element-plus";
import ExcelJS from "exceljs";
import { useI18n } from "@/core/i18n";
import { PaginationQuery } from "@/core/transport/rest";

/**
 * 列表「导出全部」下拉（CSV / Excel / 服务端全量），与 react 端 TableExportButton、
 * vben 端审计页下拉三端同构：按钮纯文字「导出全部」，客户端两项按当前搜索条件
 * 分页聚合（默认上限 1 万行）生成文件；serverExport 提供时追加「服务端全量 (XLSX)」
 * （上限远大于客户端聚合），错误处理与提示由页面闭包自己负责。
 * 放置位置：ProPage 的 #toolbar-before-tools 插槽（图标工具区之前，与 vben 同位）。
 */
const props = withDefaults(
  defineProps<{
    fetcher: (query: PaginationQuery) => Promise<{ items?: any[]; total?: number | string }>;
    columns: { prop?: string; label?: string; type?: string; cellType?: string }[];
    /** 当前搜索条件（不含分页字段），与服务端导出透传同源 */
    params?: Record<string, any>;
    filename?: string;
    maxRows?: number;
    serverExport?: () => Promise<void>;
  }>(),
  { filename: "export", maxRows: 10_000, params: undefined, serverExport: undefined },
);

const { t } = useI18n();

const exporting = ref(false);

const PAGE_SIZE = 1000;

function exportColumns() {
  return (props.columns || [])
    .filter((c: any) => c.prop && c.label && !c.type && c.cellType !== "tool")
    .map((c: any) => ({ key: String(c.prop), title: String(c.label) }));
}

function escapeCsvCell(value: unknown): string {
  if (value === null || value === undefined) return "";
  let str: string;
  if (value instanceof Date) str = value.toISOString();
  else if (typeof value === "object") str = JSON.stringify(value);
  else str = String(value);
  // 值含逗号/引号/换行时包双引号，内部引号翻倍（RFC 4180）
  if (/[",\n\r]/.test(str)) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

function downloadBlob(content: BlobPart, filename: string, mime: string) {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

async function handleCommand(command: string | number | object) {
  const key = String(command);
  if (key === "server-xlsx") {
    await props.serverExport?.();
    return;
  }
  if (key !== "csv" && key !== "xlsx") return;

  const cols = exportColumns();
  if (cols.length === 0) {
    ElMessage.error(t("pages.curd.message.selectFields"));
    return;
  }

  exporting.value = true;
  try {
    // 分页聚合拉取（受 maxRows 上限约束），透传当前搜索条件——导出的即当前搜索看到的
    const filters = props.params ?? {};
    const rows: any[] = [];
    for (let page = 1; rows.length < props.maxRows; page++) {
      const resp = await props.fetcher(
        new PaginationQuery({
          paging: { page, pageSize: PAGE_SIZE },
          formValues:
            filters && Object.keys(filters).length > 0 ? filters : undefined,
        }),
      );
      const items = resp.items ?? [];
      rows.push(...items);
      if (items.length === 0) break; // 防御：空页避免死循环
    }
    const sliced = rows.slice(0, props.maxRows);

    if (key === "xlsx") {
      const workbook = new ExcelJS.Workbook();
      const sheet = workbook.addWorksheet("Sheet1");
      sheet.addRow(cols.map((c) => c.title));
      sliced.forEach((row) => {
        sheet.addRow(cols.map((c) => row?.[c.key] ?? ""));
      });
      const buffer = await workbook.xlsx.writeBuffer();
      downloadBlob(
        buffer as ArrayBuffer,
        `${props.filename}-${Date.now()}.xlsx`,
        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      );
    } else {
      // BOM：保证 Excel 打开中文不乱码
      const header = cols.map((c) => escapeCsvCell(c.title)).join(",");
      const lines = sliced.map((row) =>
        cols.map((c) => escapeCsvCell(row?.[c.key])).join(","),
      );
      downloadBlob(
        `\uFEFF${header}\n${lines.join("\n")}`,
        `${props.filename}-${Date.now()}.csv`,
        "text/csv;charset=utf-8",
      );
    }
    ElMessage.success(t("pages.curd.export.success", { count: sliced.length }));
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含原因
    console.error("[export] client-side export failed", error);
    ElMessage.error(error?.message || t("pages.curd.export.failed"));
  } finally {
    exporting.value = false;
  }
}
</script>
