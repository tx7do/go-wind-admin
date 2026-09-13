// 全局通用分页查询类
export class PaginationQuery {
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

  /**
   * 移除对象中的 null 和 undefined 值
   * @param obj - 需要清理的对象
   * @returns 清理后的对象
   */
  private static removeNullUndefined<T extends Record<string, unknown>>(obj: T): Partial<T> {
    return Object.fromEntries(
      Object.entries(obj).filter(([_, v]) => v !== null && v !== undefined && v !== ''),
    ) as Partial<T>;
  }

  /**
   * 创建列表查询 JSON 过滤字符串
   *
   * 后端 go-crud 的裸 `{"field": value}` 走 EQ 精确匹配；搜索框输入部分关键词
   * 必须使用 `field__contains` 才是模糊匹配。这里约定：字符串值统一转
   * `__contains`（contains 是完整值精确匹配的超集，对下拉枚举等完整值查询
   * 结果一致），非字符串（数字/布尔）保持 EQ 精确语义。
   * @param formValues - 查询表单值
   * @param needCleanTenant - 是否需要清理租户字段
   * @returns JSON 字符串或 undefined
   */
  /**
   * key 末段是否已是 go-crud 支持的查询操作符（created_at__gte、type__not 等）。
   * 这类 key 不能再叠加 __contains：go-crud 会把 `a__gte__contains`
   * 解析成 `a CONTAINS value`，对时间/布尔列直接 SQL 报错（500），
   * "排除"等语义也可能被静默反转。
   *
   * 下表与 go-crud `filter/operator_converter.go` 的 operatorMap 别名集
   * 逐字对齐（含 i_* 前缀变体、数值比较长拼写/连字符拼写、is_not_null
   * 拼写族、json_contains/array_contains/exists/search/exact/iexact）。
   * 后端映射表变更时须同步此表；匹配前做小写归一（后端同）。此表缺项
   * 会让对应后缀键被追加 __contains 而语义反转或 500（2026-09-13 修复的缺口）。
   */
  private static hasOperatorSuffix(key: string): boolean {
    const idx = key.lastIndexOf('__');
    if (idx === -1) return false;
    return [
      'eq', 'equal', 'equals',
      'ne', 'neq', 'not', 'not_equal', 'not_equals', 'not-equal',
      'gt', 'greater_than', 'greater-than',
      'gte', 'greater_than_or_equal', 'greater_equals', 'greater_or_equal', 'greater-or-equal',
      'lt', 'less_than', 'less-than',
      'lte', 'less_than_or_equal', 'less_equals', 'less_or_equal', 'less-or-equal',
      'like', 'ilike', 'i_like', 'not_like', 'notlike',
      'in', 'nin', 'not_in', 'notin',
      'is_null', 'isnull', 'is_not_null', 'isnot_null', 'isnotnull', 'not_isnull',
      'between', 'range',
      'regexp', 'regex', 'iregexp', 'i_regexp', 'iregex',
      'contains', 'icontains', 'i_contains',
      'starts_with', 'startswith', 'istarts_with', 'i_starts_with', 'istartswith',
      'ends_with', 'endswith', 'iends_with', 'i_ends_with', 'iendswith',
      'json_contains', 'array_contains', 'exists', 'search', 'exact', 'iexact', 'i_exact',
    ].includes(key.slice(idx + 2).toLowerCase());
  }

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

    // 字符串值转模糊匹配。ID 类字段（*_id/idXxx）即使值是字符串也保持 EQ：
    // 它们指向数字列且多为页面隐式固定参数，contains 会导致 SQL 报错或误匹配。
    // 已带操作符后缀的 key（field__gte / field__not 等）保持原样。
    const fuzzy = Object.fromEntries(
      Object.entries(cleaned).map(([key, value]) => [
        typeof value === 'string' &&
        !/(_id$|Id$|ID$|^id$)/.test(key) &&
        !this.hasOperatorSuffix(key)
          ? `${key}__contains`
          : key,
        value,
      ]),
    );

    if (needCleanTenant) {
      // 删除租户相关字段 tenant_id 和 tenantId
      const { tenant_id, tenantId, ...rest } = fuzzy as Record<string, unknown>;

      // 过滤掉空对象
      if (Object.keys(rest).length === 0) {
        return undefined;
      }

      return JSON.stringify(rest);
    }

    // 默认返回整个 fuzzy 对象的 JSON 字符串
    return JSON.stringify(fuzzy);
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

/**
 * 分页查询结果类型
 */
export type PaginationResult<T> = {
  items: T[]; // 数据项
  total: number; // 总数
};
