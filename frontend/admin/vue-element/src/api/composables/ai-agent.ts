import { apiClient } from "@/api/client";
import { PaginationQuery } from "@/core/transport/rest";

// ==============================
// AI Agent 编排（可视化画布应用）
// ==============================

/** 分页查询 Agent 应用列表 */
export async function fetchListAiAgentApps(query: PaginationQuery) {
  return apiClient.aiAgentService.List(query.toRawParams());
}

/** 查询应用详情（Get 全量，graph 字段完整）——画布/试运行打开时用它，
 * 不把列表行当图的事实源（行字段可能被裁剪或滞后）。 */
export async function fetchAiAgentApp(id: string) {
  return apiClient.aiAgentService.Get({ id });
}

/** 创建 Agent 应用（graph 允许空串=先建壳后画图） */
export async function createAiAgentApp(data: Record<string, any>) {
  return apiClient.aiAgentService.Create({ data: data as any });
}

/** 更新 Agent 应用；updateMask 显式列举业务字段 */
export async function updateAiAgentApp(
  id: string,
  values: Record<string, any>,
  updateMask?: string,
) {
  return apiClient.aiAgentService.Update({
    id,
    data: values as any,
    updateMask,
  });
}

/** 删除 Agent 应用 */
export async function deleteAiAgentApp(id: string) {
  return apiClient.aiAgentService.Delete({ id });
}

/** 试运行：同步执行图并返回含 trace 的运行记录 */
export async function runAiAgentApp(id: string, input: Record<string, string>) {
  return apiClient.aiAgentService.Run({ id, input });
}

/** 某应用的运行台账（appId eq 过滤） */
export async function fetchListAiAgentRuns(query: PaginationQuery) {
  return apiClient.aiAgentService.ListRuns(query.toRawParams());
}

/** 版本历史（每次更新留旧版 graph 快照，新快照在前） */
export async function listAiAgentRevisions(appId: string, page?: { page: number; pageSize: number }) {
  return apiClient.aiAgentService.ListRevisions({
    appId,
    page: new PaginationQuery({
      paging: page ? { page: page.page, pageSize: page.pageSize } : { page: 1, pageSize: 50 },
    }).toRawParams(),
  });
}

/** 恢复版本（快照 graph 写回主表；恢复前先快照当前版本，可逆） */
export async function restoreAiAgentRevision(id: string) {
  return apiClient.aiAgentService.RestoreRevision({ id });
}
