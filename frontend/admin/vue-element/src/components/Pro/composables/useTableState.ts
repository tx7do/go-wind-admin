import { shallowRef, ref, reactive } from "vue";
import type { PagingResult } from "@/core/transport/rest";
import { DEFAULT_CURRENT_PAGE, DEFAULT_PAGE_SIZE, DEFAULT_PAGE_SIZES } from "../constants";

export interface UseTableConfig {
  indexAction: (queryParams: any) => Promise<PagingResult<any> | any[]>;
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

  async function fetch(queryParams: any = {}, resetPage = false) {
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
      if (showPagination && !Array.isArray(res)) {
        data.value = (res as PagingResult<T>).items ?? [];
        pagination.total = (res as PagingResult<T>).total ?? 0;
      } else {
        data.value = Array.isArray(res) ? res : ((res as PagingResult<T>).items ?? []);
      }
    } finally {
      loading.value = false;
    }
  }

  function handleSelectionChange(rows: T[]) {
    selection.value = rows;
  }

  function getSelectionIds() {
    return selection.value.map((r) => (r as any)[rowKey]);
  }

  return {
    data,
    loading,
    pagination,
    selection,
    showPagination,
    fetch,
    handleSelectionChange,
    getSelectionIds,
  };
}
