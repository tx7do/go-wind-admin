import type { ExtendedFormApi } from '@vben-core/form-ui';
import type { VxeGridInstance } from 'vxe-table';

import type { VxeGridProps } from './types';

import { ref, toRaw } from 'vue';

import { Store } from '@vben-core/shared/store';
import {
  bindMethods,
  isBoolean,
  isFunction,
  mergeWithArrayOverride,
  StateHandler,
} from '@vben-core/shared/utils';

function getDefaultState(): VxeGridProps {
  return {
    class: '',
    gridClass: '',
    gridOptions: {},
    gridEvents: {},
    formOptions: undefined,
    showSearchForm: true,
  };
}

export class VxeGridApi {
  private isMounted = false;

  private stateHandler: StateHandler;
  public formApi = {} as ExtendedFormApi;

  // private prevState: null | VxeGridProps = null;
  public grid = {} as VxeGridInstance;

  public state: null | VxeGridProps = null;

  public store: Store<VxeGridProps>;

  // proxy query 的生命周期由 extends.ts 包裹 query 时写入，组件只读它渲染加载/错误态。
  // 不借用 vxe 内部的 tableLoading：它在请求发出瞬间就点亮遮罩，本地快响应会闪一下，
  // 而"超过 250ms 才出现"的阈值只能由外层自己计时。
  public queryLoading = ref(false);
  /** 最近一次列表请求的失败原因（null = 无错），成功返回后清空 */
  public queryError = ref<string | null>(null);
  /** 是否成功取到过数据：首屏骨架与刷新加载态的分界 */
  public hasLoaded = ref(false);

  constructor(options: VxeGridProps = {}) {
    const storeState = { ...options };

    const defaultState = getDefaultState();
    this.store = new Store<VxeGridProps>(
      mergeWithArrayOverride(storeState, defaultState),
      {
        onUpdate: () => {
          // this.prevState = this.state;
          this.state = this.store.state;
        },
      },
    );

    this.state = this.store.state;
    this.stateHandler = new StateHandler();
    bindMethods(this);
  }

  mount(instance: null | VxeGridInstance, formApi: ExtendedFormApi) {
    if (!this.isMounted && instance) {
      this.grid = instance;
      this.formApi = formApi;
      this.stateHandler.setConditionTrue();
      this.isMounted = true;
    }
  }

  async query(params: Record<string, any> = {}) {
    try {
      await this.grid.commitProxy('query', toRaw(params));
    } catch (error) {
      console.error('Error occurred while querying:', error);
    }
  }

  async reload(params: Record<string, any> = {}) {
    try {
      await this.grid.commitProxy('reload', toRaw(params));
    } catch (error) {
      console.error('Error occurred while reloading:', error);
    }
  }

  setGridOptions(options: Partial<VxeGridProps['gridOptions']>) {
    this.setState({
      gridOptions: options,
    });
  }

  setLoading(isLoading: boolean) {
    this.setState({
      gridOptions: {
        loading: isLoading,
      },
    });
  }

  setState(
    stateOrFn:
      | ((prev: VxeGridProps) => Partial<VxeGridProps>)
      | Partial<VxeGridProps>,
  ) {
    if (isFunction(stateOrFn)) {
      this.store.setState((prev) => {
        return mergeWithArrayOverride(stateOrFn(prev), prev);
      });
    } else {
      this.store.setState((prev) => mergeWithArrayOverride(stateOrFn, prev));
    }
  }

  toggleSearchForm(show?: boolean) {
    this.setState({
      showSearchForm: isBoolean(show) ? show : !this.state?.showSearchForm,
    });
    // nextTick(() => {
    //   this.grid.recalculate();
    // });
    return this.state?.showSearchForm;
  }

  unmount() {
    this.isMounted = false;
    this.stateHandler.reset();
  }
}
