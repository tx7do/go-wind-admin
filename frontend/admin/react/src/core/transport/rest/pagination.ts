import { makeOrderBy, makeQueryString } from '@/core/transport/rest';

// 全局通用分页查询类
class PaginationQuery {
  paging?: { page?: number; pageSize?: number };
  formValues?: Record<string, unknown> | null;
  fieldMask?: string | string[] | null;
  orderBy?: string[] | null;

  isTenantUser: boolean = false;

  constructor(data?: Partial<PaginationQuery>) {
    if (data) {
      this.paging = data.paging;
      this.formValues = data.formValues;
      this.fieldMask = data.fieldMask;
      this.orderBy = data.orderBy;
      this.isTenantUser = data?.isTenantUser ?? false;
    }
  }

  // 是否不分页
  get noPaging(): boolean {
    return !this.paging?.page && !this.paging?.pageSize;
  }

  // 生成 orderBy 字符串
  get orderByString(): string | undefined {
    return makeOrderBy(this.orderBy);
  }

  // 生成 query 字符串
  get queryString(): string | undefined {
    return makeQueryString(this.formValues, this.isTenantUser);
  }

  // 自动格式化 fieldMask
  private get formattedFieldMask(): string | undefined {
    if (!this.fieldMask) return undefined;

    // 数组 → 逗号分隔
    if (Array.isArray(this.fieldMask)) {
      return this.fieldMask.filter(Boolean).join(',');
    }

    // 字符串直接返回
    return this.fieldMask.trim() || undefined;
  }

  // 直接生成后端需要的 pagination_PagingRequest
  toRawParams() {
    return {
      page: this.paging?.page,
      pageSize: this.paging?.pageSize,
      noPaging: this.noPaging,
      fieldMask: this.formattedFieldMask,
      orderBy: this.orderByString,
      query: this.queryString,
      sorting: undefined,
      offset: undefined,
      limit: undefined,
      token: undefined,
      filter: undefined,
      filterExpr: undefined,
    };
  }
}

type PaginationResult<T> = {
  items: T[];
  total: number;
};

export type { PaginationResult, PaginationQuery };
