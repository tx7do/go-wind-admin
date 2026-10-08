/**
 * 全局搜索的语义搜索提供者注册表。
 *
 * 框架层（layouts）不依赖任何应用的 API 客户端：应用侧在启动时把
 * 语义搜索实现（如 pgvector 菜单索引 SemanticSearch RPC）注册进来，
 * SearchPanel 检测到提供者存在才启用语义搜索区块。
 */

export interface SemanticSearchResult {
  /** 展示标题 */
  title: string;
  /** 目标路由（应用内 path） */
  route: string;
  /** 相似度得分（可选，仅展示用） */
  score?: number;
}

export type SemanticSearchProvider = (
  query: string,
) => Promise<SemanticSearchResult[]>;

let provider: SemanticSearchProvider | null = null;

/** 注册语义搜索提供者；传 null 可注销 */
export function registerSemanticSearchProvider(p: SemanticSearchProvider | null) {
  provider = p;
}

export function getSemanticSearchProvider(): SemanticSearchProvider | null {
  return provider;
}
