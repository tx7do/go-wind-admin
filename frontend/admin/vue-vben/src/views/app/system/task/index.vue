<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideCirclePlay, LucideCircleStop, LucideFilePenLine, LucidePause, LucidePlay, LucideRotateCcw, LucideRotateCw, LucideServerCog, LucideTrash2 } from '@vben/icons';

import { Modal, notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import SystemTasksModal from './system-tasks-modal.vue';
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
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';

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
      width: 220,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListTasks(new PaginationQuery({ paging: { page, pageSize } }));

const sysTasksOpen = ref(false);

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

  openModal(true);
}

async function handleRestartAllTask() {

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

/* 「更多」下拉里的重启/删除走 Modal 确认（下拉里放不下 Popconfirm） */
function confirmRestartTask(row: any) {
  Modal.confirm({
    title: $t('page.task.text.do_you_want_restart_task', {
      moduleName: $t('page.task.moduleName'),
    }),
    okText: $t('ui.button.ok'),
    cancelText: $t('ui.button.cancel'),
    onOk: () => handleRestartTask(row),
  });
}

function confirmDeleteTask(row: any) {
  Modal.confirm({
    title: $t('ui.text.do_you_want_delete', {
      moduleName: $t('page.task.moduleName'),
    }),
    okText: $t('ui.button.ok'),
    cancelText: $t('ui.button.cancel'),
    okType: 'danger',
    onOk: () => handleDelete(row),
  });
}

/* 编辑 */
function handleEdit(row: any) {
  openModal(false, row);
}

/* 操作列描述符：行内只留前 3 个（编辑/启动/停止），重启/删除溢出进「更多」，
   下拉里放不下 Popconfirm 由组件自动降级 Modal.confirm（confirmRestart/confirmDelete）。
   行参数在点击时由组件回传（onClick(row)），这里只需静态描述。 */
function buildRowActions(): TableAction[] {
  return [
    {
      key: 'edit',
      label: $t('common.edit'),
      icon: h(LucideFilePenLine),
      onClick: handleEdit,
    },
    {
      key: 'start',
      label: $t('page.task.button.start'),
      icon: h(LucideCirclePlay),
      success: true,
      confirm: {
        title: $t('page.task.text.do_you_want_start_task', {
          moduleName: $t('page.task.moduleName'),
        }),
      },
      onClick: handleStartTask,
    },
    {
      key: 'stop',
      label: $t('page.task.button.stop'),
      icon: h(LucideCircleStop),
      danger: true,
      confirm: {
        title: $t('page.task.text.do_you_want_stop_task', {
          moduleName: $t('page.task.moduleName'),
        }),
      },
      onClick: handleStopTask,
    },
    {
      key: 'restart',
      label: $t('page.task.button.restart'),
      icon: h(LucideRotateCcw),
      confirm: {
        title: $t('page.task.text.do_you_want_restart_task', {
          moduleName: $t('page.task.moduleName'),
        }),
      },
      onClick: confirmRestartTask,
    },
    {
      key: 'delete',
      label: $t('common.delete'),
      icon: h(LucideTrash2),
      danger: true,
      confirm: {
        title: $t('ui.text.do_you_want_delete', {
          moduleName: $t('page.task.moduleName'),
        }),
      },
      onClick: confirmDeleteTask,
    },
  ];
}

/* 删除 */
async function handleDelete(row: any) {

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
        <a-button class="mr-2" :icon="h(LucideServerCog)" @click="sysTasksOpen = true">
          {{ $t('page.task.sysTasksButton') }}
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
          <a-button class="gowind-btn-success-outline mr-2" :icon="h(LucidePlay)">
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
          <a-button class="gowind-btn-danger-outline mr-2" :icon="h(LucidePause)">
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
          <a-button class="gowind-btn-warning-outline mr-2" :icon="h(LucideRotateCw)">
            {{ $t('page.task.button.restartAll') }}
          </a-button>
        </a-popconfirm>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="tasks" />
        <TableAddButton @click="handleCreate" />
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
        <TableActions :row="row" :actions="buildRowActions()" />
      </template>
    </Grid>
    <SystemTasksModal v-model:open="sysTasksOpen" />
    <Drawer />
  </Page>
</template>
