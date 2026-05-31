// 全局通用分页查询类
export class PaginationQuery {
  fieldMask?: null | string | string[];
  formValues?: null | Record<string, unknown>;
  isTenantUser: boolean = false;
  orderBy?: null | string[];

  paging?: { page?: number; pageSize?: number };

  constructor(data?: Partial<PaginationQuery>) {
    if (data) {
      this.paging = data.paging;
      this.formValues = data.formValues;
      this.fieldMask = data.fieldMask;
      this.orderBy = data.orderBy;
      this.isTenantUser = data?.isTenantUser ?? false;
    }
  }

  /**
   * 创建排序字符串
   * @param orderBy - 排序字段数组
   * @returns JSON 字符串或 undefined
   */
  private static makeOrderBy(orderBy?: null | string[]): string | undefined {
    if (orderBy === undefined) {
      orderBy = ['-created_at'];
    }
    if (orderBy === null) {
      orderBy = ['-created_at'];
    }
    return JSON.stringify(orderBy) ?? undefined;
  }

  /**
   * 创建列表查询 JSON 过滤字符串
   * @param formValues - 查询表单值
   * @param needCleanTenant - 是否需要清理租户字段
   * @returns JSON 字符串或 undefined
   */
  private static makeQueryString(
    formValues?: null | Record<string, unknown>,
    needCleanTenant: boolean = false,
  ): string | undefined {
    if (formValues === null || formValues === undefined) {
      return undefined;
    }

    // 去除掉空值
    const cleaned = this.removeNullUndefined(formValues);

    if (cleaned === undefined) return undefined;

    // 若是数组，直接按数组处理
    if (Array.isArray(cleaned)) {
      return cleaned.length === 0 ? undefined : JSON.stringify(cleaned);
    }

    // 过滤掉空对象
    if (Object.keys(cleaned).length === 0) {
      return undefined;
    }

    if (needCleanTenant) {
      // 删除租户相关字段 tenant_id 和 tenantId
      const { tenant_id, tenantId, ...rest } = cleaned as Record<
        string,
        unknown
      >;

      // 过滤掉空对象
      if (Object.keys(rest).length === 0) {
        return undefined;
      }

      return JSON.stringify(rest);
    }

    // 默认返回整个 cleaned 对象的 JSON 字符串
    return JSON.stringify(cleaned);
  }

  /**
   * 移除对象中的 null 和 undefined 值
   * @param obj - 需要清理的对象
   * @returns 清理后的对象
   */
  private static removeNullUndefined<T extends Record<string, unknown>>(
    obj: T,
  ): Partial<T> {
    return Object.fromEntries(
      Object.entries(obj).filter(
        ([_, v]) => v !== null && v !== undefined && v !== '',
      ),
    ) as Partial<T>;
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

  // 是否不分页
  get noPaging(): boolean {
    return !this.paging?.page && !this.paging?.pageSize;
  }

  // 生成 orderBy 字符串
  get orderByString(): string | undefined {
    return PaginationQuery.makeOrderBy(this.orderBy);
  }

  // 生成 query 字符串
  get queryString(): string | undefined {
    return PaginationQuery.makeQueryString(this.formValues, this.isTenantUser);
  }
}

/**
 * 分页查询结果类型
 */
export type PaginationResult<T> = {
  items: T[]; // 数据项
  total: number; // 总数
};
