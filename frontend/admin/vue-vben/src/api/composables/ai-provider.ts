import type { aiservicev1_AiProvider } from '#/api/generated/admin/service/v1';

import { apiClient } from '#/api/client';

import { makeUpdateMask } from '#/transport/rest';

// ==============================
// AI 模型提供商（平台级配置）
// ==============================

/** 分页查询 AI 提供商 */
export async function fetchListAiProviders(query: any) {
  return apiClient.aiProviderService.List(query.toRawParams());
}

/**
 * 创建 AI 提供商。
 * apiKey 为请求级敏感字段：明文仅在本次请求内存在，后端加密落库、读取视图永不回显。
 */
export async function createAiProvider(values: Record<string, any>) {
  return apiClient.aiProviderService.Create({ data: values as any });
}

/**
 * 更新 AI 提供商。apiKey 空值/非空的语义由服务端 Update 分支统一裁决
 * （空值→摘 mask 清 DTO、保留已存 key；非空→加密落库并刷新 hint），
 * 前端不做预处理，按传入键统一构建 updateMask。
 */
export async function updateAiProvider(id: number, values: Record<string, any>) {
  return apiClient.aiProviderService.Update({
    id,
    data: { ...values } as any,
    updateMask: makeUpdateMask(Object.keys(values ?? {})),
  });
}

/** 删除 AI 提供商 */
export async function deleteAiProvider(id: number) {
  return apiClient.aiProviderService.Delete({ id });
}

export type { aiservicev1_AiProvider };
