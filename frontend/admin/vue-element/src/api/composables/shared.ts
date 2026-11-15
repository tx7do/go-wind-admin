/**
 * 通用枚举与工具函数
 * 从 stores/modules/api 迁移而来
 */
import { computed } from "vue";
import ExcelJS from "exceljs";
import { PaginationQuery } from "@go-wind/admin-core";
import { $t } from "@go-wind/admin-element";

export const enableList = computed(() => [
  { value: "true", label: $t("enum.enable.true") },
  { value: "false", label: $t("enum.enable.false") },
]);

export const enableBoolList = computed(() => [
  { value: true, label: $t("enum.enable.true") },
  { value: false, label: $t("enum.enable.false") },
]);

export const successStatusList = computed(() => [
  { value: true, label: $t("enum.successStatus.success") },
  { value: false, label: $t("enum.successStatus.failed") },
]);

/** Element Plus tag 的 type 取值域，全仓「枚举 → tag 语义色」映射统一用它 */
export type TagType = "primary" | "success" | "warning" | "danger" | "info";

export function enableBoolToName(
  enable: "false" | "FALSE" | "False" | "true" | "TRUE" | "True" | boolean
) {
  switch (enable) {
    case true:
    case "true":
    case "TRUE":
    case "True": {
      return $t("enum.enable.true");
    }
    default: {
      return $t("enum.enable.false");
    }
  }
}

export const methodList = [
  { value: "GET", label: "GET" },
  { value: "POST", label: "POST" },
  { value: "PUT", label: "PUT" },
  { value: "PATCH", label: "PATCH" },
  { value: "DELETE", label: "DELETE" },
];

// HTTP 方法对应的 Tag 语义色
export const httpMethodTagTypeMap: Record<string, TagType> = {
  GET: "success",
  POST: "primary",
  PUT: "warning",
  DELETE: "danger",
  PATCH: "info",
  HEAD: "info",
  OPTIONS: "info",
};

export const statusList = computed(() => [
  { value: "ON", label: $t("enum.status.ON") },
  { value: "OFF", label: $t("enum.status.OFF") },
]);

export function statusToName(status: "OFF" | "ON" | undefined) {
  const values = statusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

// 启用/禁用 → Element Plus tag type
// 启用（ON）映射 success（经暗色柔化），禁用（OFF/缺省）统一映射 info（灰色柔化），
// 避免与启用态混淆。配合 effect="plain" 走 _dark-mode.scss 的半透明柔化路径。
export function statusToType(status: "OFF" | "ON" | undefined): "success" | "info" {
  return status === "ON" ? "success" : "info";
}

// ==============================
// 成功/失败状态
// ==============================

// 成功/失败 → tag 语义 type（列表/抽屉的「成功/失败」标签统一走它）
export function successToType(success: boolean): TagType {
  return success ? "success" : "danger";
}

export function successToName(success: boolean) {
  return success ? $t("enum.successStatus.success") : $t("enum.successStatus.failed");
}

export function successToNameWithStatusCode(success: boolean, statusCode: number) {
  return success
    ? $t("enum.successStatus.success")
    : ` ${$t("enum.successStatus.failed")} (${statusCode})`;
}


/**
 * 生成导入模板（exceljs）：表头行 = 页面声明的可导入字段标签。
 * 供 ProPage 导入弹窗「下载模板」按钮的 importTemplate（函数形态）使用；
 * 返回 { data } 以兼容弹窗现有的响应对象消费形态。
 */
export async function generateImportTemplate(fields: { label: string }[]) {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet("Sheet1");
  sheet.addRow(fields.map((f) => f.label));
  const buffer = await workbook.xlsx.writeBuffer();
  return { data: buffer as ArrayBuffer };
}

/**
 * 创建导入动作：把导入弹窗解析出的行（以模板表头即字段标签为键）经页面
 * 声明的可导入字段映射回 prop，逐行调用页面既有 create 变体——
 * { data } 包裹、服务端 proto 校验、租户隔离与审计全走既有链路。
 * 任一行失败即抛错（带行号），由导入弹窗提示；不含任何可识别字段的行跳过。
 * 外键类字段（如 orgUnitId）不进导入字段清单——名称解析属后续演进。
 */
export function createImportsAction(
  fields: { label: string; prop: string }[],
  createRow: (values: Record<string, any>) => Promise<any>
) {
  const labelToProp = new Map(fields.map((f) => [f.label, f.prop]));
  return async (rows: Record<string, any>[]) => {
    for (let i = 0; i < rows.length; i++) {
      const mapped: Record<string, any> = {};
      for (const key of Object.keys(rows[i])) {
        const prop = labelToProp.get(key);
        if (prop) mapped[prop] = rows[i][key];
      }
      if (Object.keys(mapped).length === 0) continue;
      try {
        await createRow(mapped);
      } catch (err: any) {
        throw new Error(`row ${i + 1}: ${err?.message ?? String(err)}`);
      }
    }
  };
}
