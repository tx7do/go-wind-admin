<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @toolbar="handleToolbar"
      @operate="handleOperate"
    >
      <!-- 语言 -->
      <template #language="scope: any">
        <ElTag size="small" :type="scriptLanguageToTagType(scope.row.language) as any">
          {{ scriptLanguageToName(scope.row.language) }}
        </ElTag>
      </template>

      <!-- 挂载钩子点 -->
      <template #hookPoint="scope: any">
        <ElTag v-if="scope.row.hookPoint" size="small" type="primary" effect="plain">
          {{ scope.row.hookPoint }}
        </ElTag>
        <span v-else class="text-[var(--el-text-color-placeholder)]">
          {{ $t("pages.script.unmounted") }}
        </span>
      </template>

      <!-- 是否启用 -->
      <template #isEnabled="scope: any">
        <ElSwitch
          :model-value="scope.row.isEnabled"
          :loading="!!scope.row.pending"
          @update:model-value="
            (value: string | number | boolean) => handleEnableChanged(scope.row, !!value)
          "
        />
      </template>

      <!-- 关键脚本 -->
      <template #critical="scope: any">
        <ElTag v-if="scope.row.critical" size="small" type="danger">
          {{ $t("pages.script.critical") }}
        </ElTag>
        <span v-else>-</span>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <ScriptDrawer ref="drawerRef" @success="handleSuccess" />

    <!-- 试运行对话框 -->
    <TestRunDialog ref="testRunRef" @success="handleSuccess" />

    <!-- 执行日志对话框 -->
    <ScriptLogDialog ref="logDialogRef" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElMessage, ElSwitch, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import ScriptDrawer from "./script-drawer.vue";
import ScriptLogDialog from "./script-log-dialog.vue";
import TestRunDialog from "./test-run-dialog.vue";

import {
  fetchListScripts,
  scriptLanguageList,
  scriptLanguageToName,
  scriptLanguageToTagType,
  useDeleteScript,
  useUpdateScript,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const { mutateAsync: deleteScript } = useDeleteScript();
const { mutateAsync: updateScript } = useUpdateScript();

const pageRef = ref();
const drawerRef = ref();
const testRunRef = ref();
const logDialogRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.script.name"),
        field: "name",
        attrs: { placeholder: $t("pages.script.namePlaceholder"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.script.language"),
        field: "language",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: scriptLanguageList.value,
      },
      {
        type: "input",
        label: $t("pages.script.hookPoint"),
        field: "hookPoint",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListScripts(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: Number(result.total || 0) };
    },
    deleteAction: async (ids: string) => {
      await deleteScript({ ids: ids.split(",").map((id) => Number(id)) });
    },
    toolbar: [
      {
        name: "logs",
        label: $t("pages.script.logTitle"),
        attrs: { type: "default" },
      } as ToolsButton,
    ],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.script.name"), minWidth: 180 },
      {
        prop: "language",
        label: $t("pages.script.language"),
        width: 110,
        slotName: "language",
      },
      {
        prop: "hookPoint",
        label: $t("pages.script.hookPoint"),
        minWidth: 160,
        slotName: "hookPoint",
      },
      { prop: "priority", label: $t("pages.script.priority"), width: 80 },
      {
        prop: "isEnabled",
        label: $t("pages.script.isEnabled"),
        width: 90,
        slotName: "isEnabled",
      },
      {
        prop: "critical",
        label: $t("pages.script.critical"),
        width: 90,
        slotName: "critical",
      },
      { prop: "version", label: $t("pages.script.version"), width: 80 },
      { prop: "description", label: $t("pages.script.description"), minWidth: 150 },
      {
        prop: "updatedAt",
        label: $t("pages.script.updatedAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        // 试运行+编辑+删除 实测需 171px，170 贴边只差 1px
        width: 180,
        cellType: "tool",
        buttons: [
          {
            name: "testRun",
            label: $t("pages.script.testRun"),
            icon: "lucide:play",
            attrs: { type: "success" },
          },
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          {
            name: "delete",
            label: $t("common.button.delete"),
            icon: "lucide:trash-2",
            attrs: { type: "danger" },
          },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open();
}

function handleEdit(row: any) {
  drawerRef.value?.open(row);
}

function handleToolbar(name: string) {
  if (name === "logs") {
    logDialogRef.value?.open();
  }
}

function handleOperate(data: { name: string; row: any }) {
  const { name, row } = data;
  if (name === "testRun") {
    testRunRef.value?.open(row);
  }
}

async function handleEnableChanged(row: any, checked: boolean) {
  row.pending = true;
  // 与任务管理同款策略：Switch 单向绑定服务器值，结果由 refresh() 重载决定
  try {
    await updateScript({ id: row.id, values: { isEnabled: checked } });
    ElMessage.success($t("common.notification.update_status_success"));
  } catch {
    ElMessage.error($t("common.notification.update_status_failed"));
  } finally {
    row.pending = false;
    pageRef.value?.refresh();
  }
}

function handleSuccess() {
  pageRef.value?.refresh();
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
