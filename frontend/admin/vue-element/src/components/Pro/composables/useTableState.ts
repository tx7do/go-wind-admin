import { shallowRef, ref, reactive, computed } from "vue";
import { DEFAULT_CURRENT_PAGE, DEFAULT_PAGE_SIZE, DEFAULT_PAGE_SIZES } from "../constants";
import { PaginationResult } from "@go-wind/admin-core";
import { createI18nGetErrorMsg } from "@go-wind/admin-element";

const getErrorMsg = createI18nGetErrorMsg();

export interface UseTableConfig {
  indexAction: (queryParams: any) => Promise<PaginationResult<any> | any[]>;
  rowKey?: string;
  pagination?: boolean;
  request?: { pageName: string; limitName: string };
}

export function useTableState<T = any, Q = any>(config: UseTableConfig) {
  const data = shallowRef<T[]>([]);
  const loading = ref(false);
  const selection = shallowRef<T[]>([]);
  const rowKey = config.rowKey ?? "id";
  const showPagination = config.pagination !== false;
  const request = config.request ?? { pageName: "page", limitName: "pageSize" };

  const pagination = reactive({
    currentPage: DEFAULT_CURRENT_PAGE,
    pageSize: DEFAULT_PAGE_SIZE,
    total: 0,
    pageSizes: [...DEFAULT_PAGE_SIZES] as number[],
    background: true,
  });

  // 最近一次列表请求的失败原因（null = 无错），成功返回后清空：中央 onError 只记日志不弹窗，
  // 失败必须在页面上留痕，否则用户看到的是「spinner 停了、数据还是旧的」。
  const error = ref<string | null>(null);
  const hasError = computed(() => error.value !== null);

  // 请求序号：防止快速翻页/搜索时「旧请求慢返回」覆盖「新请求」的结果。
  // 每次发起请求前自增，await 后比对；若期间又发起了新请求，则丢弃本次结果。
  let reqId = 0;

  async function fetch(queryParams: any = {}, resetPage = false) {
    const currentReqId = ++reqId;
    loading.value = true;
    if (resetPage) pagination.currentPage = 1;

    const params = showPagination
      ? {
          [request.pageName]: pagination.currentPage,
          [request.limitName]: pagination.pageSize,
          ...queryParams,
        }
      : { ...queryParams };

    try {
      const res = await config.indexAction(params as Q);
      // 期间若已发起新请求（翻页/搜索/重置），丢弃本次过期的结果，
      // 避免把旧数据写回 data/pagination 造成内容与分页器错位。
      if (currentReqId !== reqId) return;
      if (showPagination && !Array.isArray(res)) {
        data.value = (res as PaginationResult<T>).items ?? [];
        pagination.total = Number((res as PaginationResult<T>).total) || 0;
      } else {
        data.value = Array.isArray(res) ? res : ((res as PaginationResult<T>).items ?? []);
      }
      error.value = null;
    } catch (err) {
      // 同理丢弃过期请求的失败：用户此刻看的是最后一次成功请求的内容。
      if (currentReqId !== reqId) return;
      console.error("[useTableState] 列表数据请求失败", err);
      // 传输层已把 AxiosError.message 换成本地化文案（实测 2026-09-29：
      // 「网络连接错误,请检查网络设置后重试」），直接展示；没有 message 的
      // （响应拦截器抛的业务对象）才交给 getErrorMsg 按 reason 映射。
      error.value = (err as { message?: string } | null)?.message || getErrorMsg(err);
    } finally {
      // 仅当这是最后一次请求时才关闭 loading，否则交给后续请求的 finally 处理。
      if (currentReqId === reqId) loading.value = false;
    }
  }

  function handleSelectionChange(rows: T[]) {
    selection.value = rows;
  }

  function getSelectionIds() {
    return selection.value.map((r) => (r as any)[rowKey]);
  }

  // 清空已选中行。删除（批量/单条）成功后必须调用，
  // 否则 selection 仍含已删记录的 id，下次批量删除会带上不存在的 id。
  function clearSelection() {
    selection.value = [];
  }

  return {
    data,
    loading,
    error,
    hasError,
    pagination,
    selection,
    showPagination,
    fetch,
    handleSelectionChange,
    getSelectionIds,
    clearSelection,
  };
}
