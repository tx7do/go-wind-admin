import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createTaskServiceClient,
  type taskservicev1_Task_Type as Task_Type,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useTaskStore = defineStore('task', () => {
  const service = createTaskServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

  /**
   * 查询任务列表
   */
  async function listTask(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await service.List({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: makeOrderBy(orderBy),
      query: makeQueryString(formValues, userStore.isTenantUser()),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取任务
   */
  async function getTask(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建任务
   */
  async function createTask(values: Record<string, any> = {}) {
    return await service.Create({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新任务
   */
  async function updateTask(id: number, values: Record<string, any> = {}) {
    return await service.Update({
      id,
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除任务
   */
  async function deleteTask(id: number) {
    return await service.Delete({ id });
  }

  /**
   * 获取任务类型名称列表
   */
  async function listTaskTypeName() {
    return await service.ListTaskTypeName({});
  }

  /**
   * 重启所有任务
   */
  async function restartAllTask() {
    return await service.RestartAllTask({});
  }

  /**
   * 停止所有任务
   */
  async function stopAllTask() {
    return await service.StopAllTask({});
  }

  /**
   * 启动所有任务
   */
  async function startAllTask() {
    return await service.StartAllTask({});
  }

  /**
   * 控制任务运行
   */
  async function controlTask(controlType: any, typeName: any) {
    return await service.ControlTask({ controlType, typeName });
  }

  function $reset() {}

  return {
    $reset,
    listTask,
    getTask,
    createTask,
    updateTask,
    deleteTask,
    listTaskTypeName,
    restartAllTask,
    startAllTask,
    stopAllTask,
    controlTask,
  };
});

export const taskTypeList = computed(() => [
  {
    value: 'PERIODIC',
    label: $t('enum.task.type.Periodic'),
  },
  {
    value: 'DELAY',
    label: $t('enum.task.type.Delay'),
  },
  {
    value: 'WAIT_RESULT',
    label: $t('enum.task.type.WaitResult'),
  },
]);

export function taskTypeToName(taskType: Task_Type) {
  const values = taskTypeList.value;
  const matchedItem = values.find((item) => item.value === taskType);
  return matchedItem ? matchedItem.label : '';
}

export function taskTypeToColor(taskType: Task_Type) {
  switch (taskType) {
    case 'DELAY': {
      return 'blue'; // 延迟任务：蓝色（表示计划中、待执行的状态）
    }
    case 'PERIODIC': {
      return 'orange'; // 周期性任务：橙色（表示循环执行、持续运行的特性）
    }
    case 'WAIT_RESULT': {
      return 'purple'; // 等待结果任务：紫色（表示过渡状态、等待响应）
    }
    default: {
      return 'gray'; // 未知任务类型：灰色（默认中性色，避免返回undefined）
    }
  }
}
