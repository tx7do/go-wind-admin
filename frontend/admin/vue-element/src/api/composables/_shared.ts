/**
 * 通用枚举与工具函数
 * 从 stores/modules/api 迁移而来
 */
import { computed } from "vue";
import { $t } from "@/i18n";

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

export function enableBoolToColor(
  enable: "false" | "FALSE" | "False" | "true" | "TRUE" | "True" | boolean
) {
  switch (enable) {
    case false:
    case "false":
    case "FALSE":
    case "False": {
      return "#8C8C8C";
    }
    case true:
    case "true":
    case "TRUE":
    case "True": {
      return "#52C41A";
    }
    default: {
      return "#C9CDD4";
    }
  }
}

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

export const statusList = computed(() => [
  { value: "ON", label: $t("enum.status.ON") },
  { value: "OFF", label: $t("enum.status.OFF") },
]);

export function statusToName(status: "OFF" | "ON" | undefined) {
  const values = statusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

export function statusToColor(status: "OFF" | "ON" | undefined) {
  switch (status) {
    case "OFF": {
      return "#8C8C8C";
    }
    case "ON": {
      return "#52C41A";
    }
    default: {
      return "#C9CDD4";
    }
  }
}

// ==============================
// 成功/失败状态
// ==============================

export function successToColor(success: boolean) {
  return success ? "limegreen" : "crimson";
}

export function successToName(success: boolean) {
  return success ? $t("enum.successStatus.success") : $t("enum.successStatus.failed");
}

export function successToNameWithStatusCode(success: boolean, statusCode: number) {
  return success
    ? $t("enum.successStatus.success")
    : ` ${$t("enum.successStatus.failed")} (${statusCode})`;
}
