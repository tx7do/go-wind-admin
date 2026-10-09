import { apiClient } from '#/api/client';

// ==============================
// AI 内容生成（表单助手）与语义搜索
// ==============================

/** AI 按场景生成文本（DESCRIPTION / ANNOUNCEMENT / REPLY / GENERAL） */
export async function generateAiContent(req: {
  scene: 'DESCRIPTION' | 'ANNOUNCEMENT' | 'REPLY' | 'GENERAL';
  topic: string;
  context?: string;
  lang?: string;
  maxLength?: number;
}) {
  return apiClient.aiContentService.GenerateContent({
    scene: req.scene,
    topic: req.topic,
    context: req.context,
    lang: req.lang ?? 'zh-CN',
    maxLength: req.maxLength,
  });
}

/** 语义搜索（pgvector 菜单索引，返回可导航条目） */
export async function fetchSemanticSearch(query: string, limit = 8) {
  return apiClient.aiContentService.SemanticSearch({ limit, query });
}
