import type { Ref } from "vue";
import { computed, reactive, toRaw } from "vue";
import { bindMethods, isFunction, mergeWithArrayOverride, StateHandler } from "@/utils";
import type { ProPageConfig } from "../ProPage/types";

/**
 * ProPage 组件暴露的实例方法
 */
export interface ProPageExpose {
  refresh: () => void;
  tableRef: any;
  tableState: {
    data: Ref<any[]>;
    loading: Ref<boolean>;
    pagination: {
      currentPage: number;
      pageSize: number;
      total: number;
      pageSizes: number[];
      background: boolean;
    };
    selection: Ref<any[]>;
    fetch: (params?: any, resetPage?: boolean) => Promise<void>;
    handleSelectionChange: (rows: any[]) => void;
    getSelectionIds: () => any[];
  };
  modalState: {
    visible: Ref<boolean>;
    mode: Ref<"add" | "edit" | "view">;
    formData: Record<string, any>;
    open: (mode: "add" | "edit" | "view", row?: any) => void;
  };
  searchParams: Record<string, any>;
}

/**
 * ProPage 外部可操作的状态
 */
export interface ProPageState<T = any, Q = any> {
  /** 页面配置 */
  config?: ProPageConfig<T, Q>;
  /** 搜索参数 */
  searchParams?: Record<string, any>;
  /** 表格 loading */
  loading?: boolean;
  /** 弹窗可见 */
  modalVisible?: boolean;
  /** 弹窗模式 */
  modalMode?: "add" | "edit" | "view";
}

function getDefaultState(): ProPageState {
  return {
    config: undefined,
    searchParams: {},
    loading: false,
    modalVisible: false,
    modalMode: "add",
  };
}

/**
 * ProPage Api —— 命令式控制 ProPage 实例
 *
 * 提供 setState 方法，支持 mount 前预设配置，mount 后响应式更新。
 * 通过 reactive + expose 双层架构实现状态管理。
 */
export class ProPageApi {
  private isMounted = false;
  private pageExpose: ProPageExpose | null = null;
  private stateHandler: StateHandler;

  /** 响应式状态 */
  public store: ProPageState;

  constructor(config?: ProPageConfig) {
    const defaultState = getDefaultState();
    // defu 与 reactive 的类型都是递归推导的（Defu<...> / UnwrapNestedRefs<...>），
    // ProPageState 这类深层配置会连续触发 TS2589（excessively deep）。
    // 运行时行为不变：合并仍是深合并，响应式仍是同一个 reactive 代理，
    // 这里只把类型推导的中间态宽化，最后收回声明类型。
    const mergeState = mergeWithArrayOverride as unknown as (
      source: Partial<ProPageState>,
      defaults: ProPageState
    ) => ProPageState;
    this.store = reactive(
      mergeState({ config }, defaultState) as Record<string, unknown>
    ) as ProPageState;

    this.stateHandler = new StateHandler();
    bindMethods(this);
  }

  /**
   * 响应式更新状态（mount 前后均可调用）
   *
   * @example
   * ```ts
   * // mount 前预设配置
   * pageApi.setState({ config: myConfig });
   *
   * // mount 后动态更新 loading
   * pageApi.setState({ loading: true });
   *
   * // 函数式更新
   * pageApi.setState(prev => ({ searchParams: { ...prev.searchParams, extra: 1 } }));
   * ```
   */
  setState(stateOrFn: ((prev: ProPageState) => Partial<ProPageState>) | Partial<ProPageState>) {
    const prev = toRaw(this.store) as ProPageState;
    const update = isFunction(stateOrFn) ? stateOrFn(prev) : stateOrFn;
    const merged = mergeWithArrayOverride(update, prev) as ProPageState;
    Object.assign(this.store, merged);
  }

  /** 批量更新状态 */
  batchStore(cb: () => void) {
    cb();
  }

  /** 获取当前状态快照 */
  getState(): ProPageState {
    return toRaw(this.store) as ProPageState;
  }

  /** 获取状态的响应式引用 */
  useStore<T = ProPageState>(selector?: (state: NoInfer<ProPageState>) => T): Readonly<Ref<T>> {
    const sel = selector ?? ((s: any) => s);
    return computed(() => sel(this.store)) as Readonly<Ref<T>>;
  }

  /**
   * 由 ProPage 组件 mount 时调用
   */
  mount(expose: ProPageExpose) {
    if (!this.isMounted && expose) {
      this.pageExpose = expose;
      this.stateHandler.setConditionTrue();
      this.isMounted = true;
    }
  }

  /**
   * 由 ProPage 组件 unmount 时调用
   */
  unmount() {
    this.isMounted = false;
    this.pageExpose = null;
    this.stateHandler.reset();
  }

  // ==================== 数据操作 ====================

  /** 刷新当前页数据 */
  refresh() {
    this.pageExpose?.refresh();
  }

  /** 重新查询（重置到第 1 页） */
  reload(params?: Record<string, any>) {
    if (!this.pageExpose) return;
    if (params) {
      Object.assign(this.pageExpose.searchParams, params);
    }
    this.pageExpose.tableState.fetch(this.pageExpose.searchParams, true);
  }

  /** 追加查询参数并刷新 */
  query(params: Record<string, any>) {
    if (!this.pageExpose) return;
    Object.assign(this.pageExpose.searchParams, params);
    this.pageExpose.tableState.fetch(this.pageExpose.searchParams);
  }

  // ==================== 表格操作 ====================

  /** 获取表格数据 */
  getData(): any[] {
    return this.pageExpose?.tableState.data.value ?? [];
  }

  /** 获取选中行 */
  getSelection(): any[] {
    return this.pageExpose?.tableState.selection.value ?? [];
  }

  /** 获取选中行 ID */
  getSelectionIds(): any[] {
    return this.pageExpose?.tableState.getSelectionIds() ?? [];
  }

  /** 清空选中 */
  clearSelection() {
    this.pageExpose?.tableState.handleSelectionChange([]);
    this.pageExpose?.tableRef?.clearSelection?.();
  }

  /** 获取 loading 状态 */
  isLoading(): boolean {
    return this.pageExpose?.tableState.loading.value ?? false;
  }

  /** 获取分页信息 */
  getPagination(): {
    currentPage: number;
    pageSize: number;
    total: number;
  } {
    const p = this.pageExpose?.tableState.pagination;
    return {
      currentPage: p?.currentPage ?? 1,
      pageSize: p?.pageSize ?? 20,
      total: p?.total ?? 0,
    };
  }

  // ==================== 弹窗操作 ====================

  /** 打开新增弹窗 */
  openAdd() {
    this.pageExpose?.modalState.open("add");
  }

  /** 打开编辑弹窗 */
  openEdit(row: any) {
    this.pageExpose?.modalState.open("edit", row);
  }

  /** 打开查看弹窗 */
  openView(row: any) {
    this.pageExpose?.modalState.open("view", row);
  }

  /** 关闭弹窗 */
  closeModal() {
    if (this.pageExpose) {
      this.pageExpose.modalState.visible.value = false;
    }
  }

  // ==================== 搜索操作 ====================

  /** 获取当前搜索参数 */
  getSearchParams(): Record<string, any> {
    return { ...(this.pageExpose?.searchParams ?? {}) };
  }

  /** 更新配置（响应式更新，委托给 setState） */
  setConfig(config: ProPageConfig) {
    this.setState({ config });
  }

  /** 等待组件挂载完成 */
  async waitForMounted() {
    if (this.isMounted) return;
    await this.stateHandler.waitForCondition();
  }
}
