import { apiClient } from '#/api/client';

import { makeUpdateMask } from '#/transport/rest';

// ==============================
// AI 知识库（RAG：文档切片 → embedding → 向量检索）
// ==============================

/** 分页查询知识库 */
export async function fetchListAiKnowledgeBases(query: any) {
  return apiClient.aiKnowledgeBaseService.List(query.toRawParams());
}

/** 创建知识库 */
export async function createAiKnowledgeBase(values: Record<string, any>) {
  return apiClient.aiKnowledgeBaseService.Create({ data: values as any });
}

/** 更新知识库 */
export async function updateAiKnowledgeBase(id: number, values: Record<string, any>) {
  return apiClient.aiKnowledgeBaseService.Update({
    id,
    data: { ...values } as any,
    updateMask: makeUpdateMask(Object.keys(values ?? {})),
  });
}

/** 删除知识库（级联删除文档与切片） */
export async function deleteAiKnowledgeBase(id: number) {
  return apiClient.aiKnowledgeBaseService.Delete({ id });
}

/** 查询知识库下的文档 */
export async function fetchListAiDocs(baseId: number) {
  return apiClient.aiKnowledgeBaseService.ListDocs({ baseId });
}

/** 上传文档（纯文本：切片 → 向量化 → 落库） */
export async function uploadAiDoc(baseId: number, name: string, content: string) {
  return apiClient.aiKnowledgeBaseService.UploadDoc({ baseId, name, content });
}

/** 删除文档（级联删除切片） */
export async function deleteAiDoc(baseId: number, id: number) {
  return apiClient.aiKnowledgeBaseService.DeleteDoc({ baseId, id });
}
