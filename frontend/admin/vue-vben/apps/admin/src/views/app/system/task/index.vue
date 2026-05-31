<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import {
  LucideCirclePlay,
  LucideCircleStop,
  LucideFilePenLine,
  LucideRotateCcw,
  LucideTrash2,
} from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  type taskservicev1_ControlTaskRequest_ControlType as ControlTaskRequest_ControlType,
  type taskservicev1_Task as Task,
} from '#/api';
import {
  enableList,
  fetchListTasks,
  fetchListTaskTypeNames,
  PaginationQuery,
  taskTypeList,
  taskTypeToColor,
  taskTypeToName,
  useControlTask,
  useDeleteTask,
  useRestartAllTasks,
  useStartAllTasks,
  useStopAllTasks,
  useUpdateTask,
} from '#/api';
import { $t } from '#/locales';

import TaskDrawer from './task-drawer.vue';

const { mutateAsync: controlTaskMut } = useControlTask();
const { mutateAsync: deleteTask } = useDeleteTask();
const { mutateAsync: restartAllTask } = useRestartAllTasks();
const { mutateAsync: startAllTask } = useStartAllTasks();
const { mutateAsync: stopAllTask } = useStopAllTasks();
const { mutateAsync: updateTask } = useUpdateTask();

const formOptions: VbenFormProps = {
  // 默认展开
  collapsed: false,
  // 控制表单是否显示折叠按钮
  showCollapseButton: false,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'Select',
      fieldName: 'type',
      label: $t('page.task.type'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: taskTypeList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },

    {
      component: 'ApiSelect',
      fieldName: 'typeName',
      label: $t('page.task.typeName'),
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await fetchListTaskTypeNames();
          return result.typeNames;
        },
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: any) => ({
            label: item,
            value: item,
          }));
        },
      },
    },
    {
      component: 'Select',
      fieldName: 'enable',
      label: $t('ui.table.status'),
      componentProps: {
        options: enableList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Task> = {
  toolbarConfig: {
    custom: true,
    export: true,
    // import: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  exportConfig: {},
  pagerConfig: {
    enabled: false,
  },
  rowConfig: {
    isHover: true,
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        console.log('query:', formValues);

        return await fetchListTasks(
          new PaginationQuery({
            paging: {
              page: page.currentPage,
              pageSize: page.pageSize,
            },
            formValues,
          }),
        );
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.task.type'), field: 'type', slots: { default: 'type' } },
    { title: $t('page.task.typeName'), field: 'typeName' },
    { title: $t('page.task.taskPayload'), field: 'taskPayload' },
    { title: $t('page.task.cronSpec'), field: 'cronSpec' },
    {
      title: $t('page.task.enable'),
      field: 'enable',
      slots: { default: 'enable' },
      width: 95,
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    { title: $t('ui.table.remark'), field: 'remark' },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 190,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: TaskDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      // 关闭时，重载表格数据
      gridApi.reload();
    }
  },
});

/* 打开模态窗口 */
function openModal(create: boolean, row?: any) {
  drawerApi.setData({
    create,
    row,
  });

  drawerApi.open();
}

/* 创建 */
function handleCreate() {
  console.log('创建');

  openModal(true);
}

async function handleRestartAllTask() {
  console.log('重启所有任务');

  try {
    await restartAllTask();

    notification.success({
      message: $t('ui.notification.operation_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}

async function handleStartAllTask() {
  console.log('启动所有任务');

  try {
    await startAllTask();

    notification.success({
      message: $t('ui.notification.operation_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}

async function handleStopAllTask() {
  console.log('停止所有任务');

  try {
    await stopAllTask();

    notification.success({
      message: $t('ui.notification.operation_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}

/**
 * 控制任务
 * @param typeName 任务类型名称
 * @param controlType 控制类型
 */
async function controlTask(
  typeName: string,
  controlType: ControlTaskRequest_ControlType,
) {
  try {
    await controlTaskMut({ typeName, controlType });

    notification.success({
      message: $t('ui.notification.operation_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}

async function handleStartTask(row: any) {
  await controlTask(row.typeName, 'Start');
}

async function handleStopTask(row: any) {
  await controlTask(row.typeName, 'Stop');
}

async function handleRestartTask(row: any) {
  await controlTask(row.typeName, 'Restart');
}

/* 编辑 */
function handleEdit(row: any) {
  console.log('编辑', row);
  openModal(false, row);
}

/* 删除 */
async function handleDelete(row: any) {
  console.log('删除', row);

  try {
    await deleteTask({ id: row.id });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}

/* 修改状态 */
async function handleEnableChanged(row: any, checked: boolean) {
  console.log('handleStatusChanged', row.enable, checked);

  row.pending = true;
  row.enable = checked;

  try {
    await updateTask({ id: row.id, values: { enable: row.enable } });

    await controlTask(row.typeName, row.enable ? 'Start' : 'Stop');

    notification.success({
      message: $t('ui.notification.update_status_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.update_status_failed'),
    });
  } finally {
    row.pending = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.system.task')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.task.button.create') }}
        </a-button>

        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('page.task.text.do_you_want_start_all_task', {
              moduleName: $t('page.task.moduleName'),
            })
          "
          @confirm="handleStartAllTask()"
        >
          <a-button class="btn-start-all mr-2" type="primary">
            {{ $t('page.task.button.startAll') }}
          </a-button>
        </a-popconfirm>

        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('page.task.text.do_you_want_stop_all_task', {
              moduleName: $t('page.task.moduleName'),
            })
          "
          @confirm="handleStopAllTask()"
        >
          <a-button danger class="mr-2" type="primary">
            {{ $t('page.task.button.stopAll') }}
          </a-button>
        </a-popconfirm>

        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('page.task.text.do_you_want_restart_all_task', {
              moduleName: $t('page.task.moduleName'),
            })
          "
          @confirm="handleRestartAllTask()"
        >
          <a-button class="mr-2" type="primary">
            {{ $t('page.task.button.restartAll') }}
          </a-button>
        </a-popconfirm>
      </template>

      <template #enable="{ row }">
        <a-switch
          :checked="row.enable === true"
          :loading="row.pending"
          :checked-children="$t('ui.switch.active')"
          :un-checked-children="$t('ui.switch.inactive')"
          @change="
            (checked: any) => handleEnableChanged(row, checked as boolean)
          "
        />
      </template>
      <template #type="{ row }">
        <a-tag :color="taskTypeToColor(row.type)">
          {{ taskTypeToName(row.type) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideFilePenLine)"
          @click.stop="handleEdit(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('page.task.text.do_you_want_start_task', {
              moduleName: $t('page.task.moduleName'),
            })
          "
          @confirm="handleStartTask(row)"
        >
          <a-button
            type="link"
            class="green-link-btn"
            :icon="h(LucideCirclePlay)"
          />
        </a-popconfirm>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('page.task.text.do_you_want_stop_task', {
              moduleName: $t('page.task.moduleName'),
            })
          "
          @confirm="handleStopTask(row)"
        >
          <a-button danger type="link" :icon="h(LucideCircleStop)" />
        </a-popconfirm>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('page.task.text.do_you_want_restart_task', {
              moduleName: $t('page.task.moduleName'),
            })
          "
          @confirm="handleRestartTask(row)"
        >
          <a-button type="link" :icon="h(LucideRotateCcw)" />
        </a-popconfirm>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.task.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>

<style scoped>
.btn-start-all {
  background-color: #52c41a !important;
  border-color: #52c41a !important;
  color: #fff !important;
}

.btn-start-all:hover,
.btn-start-all:focus {
  background-color: #4cae4c !important;
  border-color: #4cae4c !important;
}

.btn-start-all[disabled] {
  background-color: #c2e7b0 !important;
  border-color: #c2e7b0 !important;
  color: #86b379 !important;
  cursor: not-allowed !important;
}

:deep(.green-link-btn) {
  color: #52c41a !important;
}

:deep(.green-link-btn:hover) {
  color: #4cae4c !important;
}
</style>
