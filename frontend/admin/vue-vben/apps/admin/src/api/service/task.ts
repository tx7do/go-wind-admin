import {
  createTaskServiceClient,
  type taskservicev1_ControlTaskRequest,
  type taskservicev1_CreateTaskRequest,
  type taskservicev1_DeleteTaskRequest,
  type taskservicev1_GetTaskRequest,
  type taskservicev1_UpdateTaskRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createTaskServiceClient> = null;

export function getTaskService() {
  if (!_instance) {
    _instance = createTaskServiceClient(requestApi);
  }
  return _instance;
}

export async function listTasks(query: PaginationQuery) {
  const params = query.toRawParams();
  return getTaskService().List(params);
}

export async function getTask(request: taskservicev1_GetTaskRequest) {
  return getTaskService().Get(request);
}

export async function createTask(request: taskservicev1_CreateTaskRequest) {
  return getTaskService().Create(request);
}

export async function updateTask(request: taskservicev1_UpdateTaskRequest) {
  return getTaskService().Update(request);
}

export async function deleteTask(request: taskservicev1_DeleteTaskRequest) {
  return getTaskService().Delete(request);
}

/** 获取任务类型名称列表 */
export async function listTaskTypeNames() {
  return getTaskService().ListTaskTypeName({});
}

/** 控制单个任务（Start/Stop/Restart） */
export async function controlTask(typeName: string, controlType: string) {
  return getTaskService().ControlTask({
    typeName,
    controlType: controlType as taskservicev1_ControlTaskRequest['controlType'],
  });
}

/** 启动所有任务 */
export async function startAllTasks() {
  return getTaskService().StartAllTask({});
}

/** 停止所有任务 */
export async function stopAllTasks() {
  return getTaskService().StopAllTask({});
}

/** 重启所有任务 */
export async function restartAllTasks() {
  return getTaskService().RestartAllTask({});
}
