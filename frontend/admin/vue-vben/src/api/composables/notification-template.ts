import type {
  notificationservicev1_CreateNotificationTemplateRequest,
  notificationservicev1_DeleteNotificationTemplateRequest,
  notificationservicev1_ListNotificationTemplateResponse,
  notificationservicev1_NotificationTemplate,
  notificationservicev1_RenderNotificationTemplateRequest,
  notificationservicev1_RenderNotificationTemplateResponse,
  notificationservicev1_UpdateNotificationTemplateRequest,
} from '#/api/generated/admin/service/v1';

import { useMutation } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { PaginationQuery } from '#/api';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// 通知模板（可复用的标题/正文占位模板，发送方以 template_code 引用）
// ==============================
//
// 建后 code 不可改：它是发送方的引用锚，改码等于让所有引用悬空
// （更新掩码固定列举业务字段，code 不进掩码，与 react/ele 基准一致）。

const LIST_KEY = 'listNotificationTemplates';

/** 更新掩码：code 刻意不进掩码（见文件头注释）。 */
export const NOTIFICATION_TEMPLATE_UPDATE_MASK =
  'name,titleTemplate,contentTemplate,isEnabled,remark';

/** 分页查询模板（非 Hook，列表页 query / 预览等手动调用） */
export async function fetchListNotificationTemplates(
  params: PaginationQuery,
): Promise<notificationservicev1_ListNotificationTemplateResponse> {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, params],
    queryFn: () =>
      apiClient.notificationTemplateService.ListNotificationTemplate(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

/** 创建模板：CRUD 请求体必须包 { data: {...} } */
export function useCreateNotificationTemplate(
  options?: Parameters<
    typeof useMutation<
      notificationservicev1_NotificationTemplate,
      Error,
      notificationservicev1_CreateNotificationTemplateRequest
    >
  >[0],
) {
  return useMutation({
    mutationFn: (
      req: notificationservicev1_CreateNotificationTemplateRequest,
    ) => apiClient.notificationTemplateService.CreateNotificationTemplate(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/** 更新模板 */
export function useUpdateNotificationTemplate(
  options?: Parameters<
    typeof useMutation<any, Error, notificationservicev1_UpdateNotificationTemplateRequest>
  >[0],
) {
  return useMutation({
    mutationFn: (req: notificationservicev1_UpdateNotificationTemplateRequest) =>
      apiClient.notificationTemplateService.UpdateNotificationTemplate(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/** 删除模板 */
export function useDeleteNotificationTemplate(
  options?: Parameters<
    typeof useMutation<any, Error, notificationservicev1_DeleteNotificationTemplateRequest>
  >[0],
) {
  return useMutation({
    mutationFn: (req: notificationservicev1_DeleteNotificationTemplateRequest) =>
      apiClient.notificationTemplateService.DeleteNotificationTemplate(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/** 试渲染：按变量集渲染，返回标题与正文（无投递副作用） */
export function useRenderNotificationTemplate(
  options?: Parameters<
    typeof useMutation<
      notificationservicev1_RenderNotificationTemplateResponse,
      Error,
      notificationservicev1_RenderNotificationTemplateRequest
    >
  >[0],
) {
  return useMutation({
    mutationFn: (req: notificationservicev1_RenderNotificationTemplateRequest) =>
      apiClient.notificationTemplateService.RenderNotificationTemplate(req),
    ...options,
  });
}
