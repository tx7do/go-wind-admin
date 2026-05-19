<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <!-- 搜索 -->
    <PageSearch
      ref="searchRef"
      :search-config="searchConfig"
      @query-click="handleQueryClick"
      @reset-click="handleResetClick"
    />

    <!-- 列表 -->
    <PageContent
      ref="contentRef"
      :content-config="contentConfig"
      @add-click="handleAddClick"
      @operate-click="handleOperateClick"
      @toolbar-click="handleToolbarClick"
    >
      <!-- 是否启用 -->
      <template #enable="{ row }">
        <ElSwitch
          v-model="row.enable"
          :loading="row.pending"
          :active-text="$t('common.switch.active')"
          :inactive-text="$t('common.switch.inactive')"
          @change="(value: boolean) => handleEnableChanged(row, value)"
        />
      </template>

      <!-- 任务类型 -->
      <template #type="{ row }">
        <ElTag size="small" effect="dark" round :color="taskTypeToColor(row.type)">
          {{ taskTypeToName(row.type) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <TaskDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElMessage, ElMessageBox, ElSwitch, ElTag } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import TaskDrawer from "./task-drawer.vue";

import { enableList, taskTypeList, taskTypeToColor, taskTypeToName, useTaskStore } from "@/stores";
import { $t } from "@/i18n";

const taskStore = useTaskStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true, // 启用 Grid 布局
  formItems: [
    {
      type: "select",
      label: $t("pages.task.type"),
      prop: "type",
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
      prop: "typeName",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("common.table.status"),
      prop: "enable",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: enableList.value,
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:task", // 任务管理权限前缀
  toolbarRight: [
    {
      name: "startAll",
      text: $t("pages.task.button.startAll"),
      attrs: {
        type: "success",
      },
    },
    {
      name: "stopAll",
      text: $t("pages.task.button.stopAll"),
      attrs: {
        type: "danger",
      },
    },
    {
      name: "restartAll",
      text: $t("pages.task.button.restartAll"),
      attrs: {
        type: "primary",
      },
    },
    "add",
  ],
  defaultToolbar: ["refresh", "exports", "filter"],
  table: {
    border: true,
    stripe: false,
  },
  pagination: false, // 禁用分页
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await taskStore.listTask(
      {
        page: page || 1,
        pageSize: pageSize || 10,
      },
      queryParams
    );
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
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
      template: "date",
      dateFormat: "YYYY-MM-DD HH:mm:ss",
    },
    { prop: "remark", label: $t("common.table.remark"), minWidth: 150 },
    {
      prop: "action",
      label: $t("common.table.action"),
      fixed: "right",
      width: 240,
      template: "tool",
      action: [
        {
          name: "edit",
          text: $t("common.button.edit"),
        },
        {
          name: "start",
          text: $t("pages.task.text.do_you_want_start_task", {
            moduleName: $t("pages.task.moduleName"),
          }),
          attrs: {
            type: "success",
          },
        },
        {
          name: "stop",
          text: $t("pages.task.text.do_you_want_stop_task", {
            moduleName: $t("pages.task.moduleName"),
          }),
          attrs: {
            type: "danger",
          },
        },
        {
          name: "restart",
          text: $t("pages.task.text.do_you_want_restart_task", {
            moduleName: $t("pages.task.moduleName"),
          }),
          attrs: {
            type: "primary",
          },
        },
        {
          name: "delete",
          text: $t("common.button.delete"),
          attrs: {
            type: "danger",
          },
        },
      ],
    },
  ],
};

// 处理操作点击
const handleOperateClick = (data: IOperateData) => {
  const { name, row } = data;

  if (name === "edit") {
    // 编辑
    drawerRef.value?.open(row);
  } else if (name === "start") {
    // 启动任务
    ElMessageBox.confirm(
      $t("pages.task.text.do_you_want_start_task", { moduleName: $t("pages.task.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await taskStore.controlTask(row.typeName, "Start");
        ElMessage.success($t("common.notification.operation_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.operation_failed"));
      }
    });
  } else if (name === "stop") {
    // 停止任务
    ElMessageBox.confirm(
      $t("pages.task.text.do_you_want_stop_task", { moduleName: $t("pages.task.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await taskStore.controlTask(row.typeName, "Stop");
        ElMessage.success($t("common.notification.operation_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.operation_failed"));
      }
    });
  } else if (name === "restart") {
    // 重启任务
    ElMessageBox.confirm(
      $t("pages.task.text.do_you_want_restart_task", { moduleName: $t("pages.task.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await taskStore.controlTask(row.typeName, "Restart");
        ElMessage.success($t("common.notification.operation_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.operation_failed"));
      }
    });
  } else if (name === "delete") {
    // 删除
    ElMessageBox.confirm(
      $t("common.confirm.do_you_want_delete", { moduleName: $t("pages.task.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await taskStore.deleteTask(row.id);
        ElMessage.success($t("common.notification.delete_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.delete_failed"));
      }
    });
  }
};

// 处理新增点击
const handleAddClick = () => {
  drawerRef.value?.open();
};

// 处理成功回调
const handleSuccess = () => {
  contentRef.value?.fetchPageData({}, true);
};

// 处理工具栏点击
const handleToolbarClick = async (name: string) => {
  if (name === "startAll") {
    ElMessageBox.confirm(
      $t("pages.task.text.do_you_want_start_all_task", { moduleName: $t("pages.task.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await taskStore.startAllTask();
        ElMessage.success($t("common.notification.operation_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.operation_failed"));
      }
    });
  } else if (name === "stopAll") {
    ElMessageBox.confirm(
      $t("pages.task.text.do_you_want_stop_all_task", { moduleName: $t("pages.task.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await taskStore.stopAllTask();
        ElMessage.success($t("common.notification.operation_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.operation_failed"));
      }
    });
  } else if (name === "restartAll") {
    ElMessageBox.confirm(
      $t("pages.task.text.do_you_want_restart_all_task", {
        moduleName: $t("pages.task.moduleName"),
      }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await taskStore.restartAllTask();
        ElMessage.success($t("common.notification.operation_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.operation_failed"));
      }
    });
  }
};

// 修改状态
async function handleEnableChanged(row: any, checked: boolean) {
  row.pending = true;
  row.enable = checked;

  try {
    await taskStore.updateTask(row.id, { enable: row.enable });
    await taskStore.controlTask(row.typeName, row.enable ? "Start" : "Stop");
    ElMessage.success($t("common.notification.update_status_success"));
  } catch {
    ElMessage.error($t("common.notification.update_status_failed"));
  } finally {
    row.pending = false;
  }
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
