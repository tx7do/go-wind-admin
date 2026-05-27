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
      <!-- 是否启用 -->
      <template #enable="scope: any">
        <ElSwitch
          v-model="scope.row.enable"
          :loading="scope.row.pending"
          :active-text="$t('common.switch.active')"
          :inactive-text="$t('common.switch.inactive')"
          @change="(value: string | number | boolean) => handleEnableChanged(scope.row, !!value)"
        />
      </template>

      <!-- 任务类型 -->
      <template #type="scope: any">
        <ElTag size="small" effect="dark" round :color="taskTypeToColor(scope.row.type)">
          {{ taskTypeToName(scope.row.type) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <TaskDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElMessage, ElMessageBox, ElSwitch, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import TaskDrawer from "./task-drawer.vue";

import {
  enableList,
  taskTypeList,
  taskTypeToColor,
  taskTypeToName,
  fetchListTasks,
  useDeleteTask,
  useUpdateTask,
  useControlTask,
  useStartAllTasks,
  useStopAllTasks,
  useRestartAllTasks,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deleteTask } = useDeleteTask();
const { mutateAsync: updateTask } = useUpdateTask();
const { mutateAsync: controlTask } = useControlTask();
const { mutateAsync: startAllTask } = useStartAllTasks();
const { mutateAsync: stopAllTask } = useStopAllTasks();
const { mutateAsync: restartAllTask } = useRestartAllTasks();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "select",
        label: $t("pages.task.type"),
        field: "type",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: taskTypeList.value,
      },
      {
        type: "input",
        label: $t("pages.task.typeName"),
        field: "typeName",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("common.table.status"),
        field: "enable",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: enableList.value,
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListTasks(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteTask({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: [
      {
        name: "startAll",
        label: $t("pages.task.button.startAll"),
        attrs: { type: "success" },
      } as ToolsButton,
      {
        name: "stopAll",
        label: $t("pages.task.button.stopAll"),
        attrs: { type: "danger" },
      } as ToolsButton,
      {
        name: "restartAll",
        label: $t("pages.task.button.restartAll"),
        attrs: { type: "primary" },
      } as ToolsButton,
      "add",
    ],
    defaultToolbar: ["refresh", "exports", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      {
        prop: "type",
        label: $t("pages.task.type"),
        width: 120,
        slotName: "type",
      },
      { prop: "typeName", label: $t("pages.task.typeName"), minWidth: 150 },
      { prop: "taskPayload", label: $t("pages.task.taskPayload"), minWidth: 150 },
      { prop: "cronSpec", label: $t("pages.task.cronSpec"), minWidth: 150 },
      {
        prop: "enable",
        label: $t("pages.task.enable"),
        width: 100,
        slotName: "enable",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      { prop: "remark", label: $t("common.table.remark"), minWidth: 150 },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 240,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit") },
          { name: "start", label: $t("pages.task.button.start"), attrs: { type: "success" } },
          { name: "stop", label: $t("pages.task.button.stop"), attrs: { type: "danger" } },
          { name: "restart", label: $t("pages.task.button.restart"), attrs: { type: "primary" } },
          { name: "delete", label: $t("common.button.delete"), attrs: { type: "danger" } },
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

async function handleToolbar(name: string) {
  const actionMap: Record<string, () => Promise<any>> = {
    startAll: () => startAllTask(),
    stopAll: () => stopAllTask(),
    restartAll: () => restartAllTask(),
  };

  const i18nMap: Record<string, string> = {
    startAll: "pages.task.text.do_you_want_start_all_task",
    stopAll: "pages.task.text.do_you_want_stop_all_task",
    restartAll: "pages.task.text.do_you_want_restart_all_task",
  };

  const action = actionMap[name];
  if (!action) return;

  try {
    await ElMessageBox.confirm(
      $t(i18nMap[name], { moduleName: $t("pages.task.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    );
    await action();
    ElMessage.success($t("common.notification.operation_success"));
    pageRef.value?.refresh();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error($t("common.notification.operation_failed"));
    }
  }
}

async function handleOperate(data: { name: string; row: any }) {
  const { name, row } = data;
  const controlTypeMap: Record<string, string> = {
    start: "Start",
    stop: "Stop",
    restart: "Restart",
  };

  const controlType = controlTypeMap[name];
  if (controlType) {
    const i18nMap: Record<string, string> = {
      start: "pages.task.text.do_you_want_start_task",
      stop: "pages.task.text.do_you_want_stop_task",
      restart: "pages.task.text.do_you_want_restart_task",
    };

    try {
      await ElMessageBox.confirm(
        $t(i18nMap[name], { moduleName: $t("pages.task.moduleName") }),
        $t("common.title.confirm"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );
      await controlTask({ typeName: row.typeName, controlType });
      ElMessage.success($t("common.notification.operation_success"));
      pageRef.value?.refresh();
    } catch (error) {
      if (error !== "cancel") {
        ElMessage.error($t("common.notification.operation_failed"));
      }
    }
  }
}

async function handleEnableChanged(row: any, checked: boolean) {
  row.pending = true;
  row.enable = checked;
  try {
    await updateTask({ id: row.id, values: { enable: row.enable } });
    await controlTask({ typeName: row.typeName, controlType: row.enable ? "Start" : "Stop" });
    ElMessage.success($t("common.notification.update_status_success"));
  } catch {
    ElMessage.error($t("common.notification.update_status_failed"));
  } finally {
    row.pending = false;
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
