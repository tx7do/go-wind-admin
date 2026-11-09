import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { type PaginationQuery, queryClient } from '@/core';
import { makeUpdateMask } from '@/core/transport/rest/utils';
import type {
  aiservicev1_AiDoc,
  aiservicev1_AiKnowledgeBase,
  aiservicev1_CreateAiKnowledgeBaseRequest,
  aiservicev1_DeleteAiKnowledgeBaseRequest,
  aiservicev1_ListAiKnowledgeBaseResponse,
} from '@/api/generated/admin/service/v1';

// ── 知识库 ──────────────────────────────────────────────────────────

// useListAiKnowledgeBases
export function useListAiKnowledgeBases(
  query: PaginationQuery,
  options?: Omit<UseQueryOptions<aiservicev1_ListAiKnowledgeBaseResponse, Error>, 'queryKey' | 'queryFn'>,
) {
  return useQuery({
    queryKey: ['listAiKnowledgeBases', query],
    queryFn: () => apiClient.aiKnowledgeBaseService.List(query.toRawParams()),
    ...options,
  });
}

// fetchListAiKnowledgeBases — 非组件上下文（聊天页选择器下拉）
export async function fetchListAiKnowledgeBases(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listAiKnowledgeBases', params],
    queryFn: () => apiClient.aiKnowledgeBaseService.List(params.toRawParams()),
    retry: 0,
  });
}

// useCreateAiKnowledgeBase
export function useCreateAiKnowledgeBase(
  options?: UseMutationOptions<{}, Error, aiservicev1_CreateAiKnowledgeBaseRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.aiKnowledgeBaseService.Create(data),
    ...options,
  });
}

// useUpdateAiKnowledgeBase — 固定签名 { id, values }
export function useUpdateAiKnowledgeBase(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.aiKnowledgeBaseService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

// useDeleteAiKnowledgeBase — 后端级联删除文档与切片
export function useDeleteAiKnowledgeBase(
  options?: UseMutationOptions<{}, Error, aiservicev1_DeleteAiKnowledgeBaseRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.aiKnowledgeBaseService.Delete(req),
    ...options,
  });
}

// ── 文档 ────────────────────────────────────────────────────────────

// fetchListAiDocs — 文档管理抽屉
export async function fetchListAiDocs(baseId: string) {
  return queryClient.fetchQuery({
    queryKey: ['listAiDocs', baseId],
    queryFn: () =>
      apiClient.aiKnowledgeBaseService.ListDocs({ baseId }),
    retry: 0,
  });
}

// useUploadAiDoc — 上传纯文本：切片 → 向量化 → 落库
export function useUploadAiDoc(
  options?: UseMutationOptions<
    { doc?: aiservicev1_AiDoc; chunkCount?: number },
    Error,
    { baseId: string; name: string; content: string }
  >,
) {
  return useMutation({
    mutationFn: ({ baseId, name, content }) =>
      apiClient.aiKnowledgeBaseService.UploadDoc({ baseId, name, content }),
    ...options,
  });
}

// useUploadDocFile — 文件上传（txt/md/docx/pdf：后端抽取文本后走同一入库链）
// contentBase64 在 protojson 下就是 base64 字符串（bytes 类型）
export function useUploadDocFile(
  options?: UseMutationOptions<
    { doc?: aiservicev1_AiDoc; chunkCount?: number },
    Error,
    { baseId: string; fileName: string; contentBase64: string }
  >,
) {
  return useMutation({
    mutationFn: ({ baseId, fileName, contentBase64 }) =>
      apiClient.aiKnowledgeBaseService.UploadDocFile({ baseId, fileName, contentBase64 }),
    ...options,
  });
}

// useDeleteAiDoc — 级联删除切片
export function useDeleteAiDoc(
  options?: UseMutationOptions<{}, Error, { baseId: string; id: string }>,
) {
  return useMutation({
    mutationFn: ({ baseId, id }) =>
      apiClient.aiKnowledgeBaseService.DeleteDoc({ baseId, id }),
    ...options,
  });
}

export type AiKnowledgeBase = aiservicev1_AiKnowledgeBase;
export type AiDoc = aiservicev1_AiDoc;
