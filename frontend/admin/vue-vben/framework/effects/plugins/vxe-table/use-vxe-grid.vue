<script lang="ts" setup>
import type { VbenFormProps } from '@vben-core/form-ui';
import type {
  VxeGridDefines,
  VxeGridInstance,
  VxeGridListeners,
  VxeGridPropTypes,
  VxeGridProps as VxeTableGridProps,
} from 'vxe-table';

import type { ExtendedVxeGridApi, VxeGridProps } from './types';

import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  onUnmounted,
  ref,
  toRaw,
  useSlots,
  useTemplateRef,
  watch,
} from 'vue';

import { usePriorityValues } from '@vben/hooks';
import { EmptyIcon } from '@vben/icons';
import { $t } from '@vben/locales';
import { preferences, usePreferences } from '@vben/preferences';
import { cloneDeep, cn, mergeWithArrayOverride } from '@vben/utils';
import {
  VbenButton,
  VbenHelpTooltip,
  VbenLoading,
} from '@vben-core/shadcn-ui';

import { VxeGrid, VxeUI } from 'vxe-table';

import GridTableSkeleton from './table-skeleton.vue';
import { extendProxyOptions } from './extends';
import { useTableForm } from './init';

import 'vxe-table/styles/cssvar.scss';
import 'vxe-pc-ui/styles/cssvar.scss';
import './style.css';

interface Props extends VxeGridProps {
  api: ExtendedVxeGridApi;
}

const props = withDefaults(defineProps<Props>(), {});

const FORM_SLOT_PREFIX = 'form-';

const TOOLBAR_ACTIONS = 'toolbar-actions';
const TOOLBAR_TOOLS = 'toolbar-tools';

const gridRef = useTemplateRef<VxeGridInstance>('gridRef');

const state = props.api?.useStore?.();

const {
  gridOptions,
  class: className,
  gridClass,
  gridEvents,
  formOptions,
  tableTitle,
  tableTitleHelp,
  showSearchForm,
} = usePriorityValues(props, state);

const { isMobile } = usePreferences();

const slots = useSlots();

// ---------- 列表三态（首屏骨架 / 延迟加载态 / 失败态），与 react 端 ListTable 行为对齐 ----------
// 这三个 ref 由 extends.ts 包裹 proxyConfig.ajax.query 时写入：vxe 内部的 tableLoading 在请求
// 发出的瞬间就点亮遮罩，"超过阈值才出现"只能由外层自己计时，所以不复用它。
const queryLoading = props.api.queryLoading;
const queryError = props.api.queryError;
const hasLoaded = props.api.hasLoaded;

/** 加载态出场阈值（ms）：本地接口常在 100ms 内返回，早于阈值出现的骨架/遮罩比什么都不显示更闪 */
const LOADING_DELAY = 250;
/** 骨架最多铺这么多行：真实表体由 vxe 定高后自己滚动，铺满可见区即可，多铺的是白画的 DOM */
const MAX_SKELETON_ROWS = 8;

const delayedLoading = ref(false);
let loadingDelayTimer: null | ReturnType<typeof setTimeout> = null;

function clearLoadingDelayTimer() {
  if (loadingDelayTimer !== null) {
    clearTimeout(loadingDelayTimer);
    loadingDelayTimer = null;
  }
}

watch(queryLoading, (isLoading) => {
  clearLoadingDelayTimer();
  if (isLoading) {
    if (!delayedLoading.value) {
      loadingDelayTimer = setTimeout(() => {
        delayedLoading.value = true;
        loadingDelayTimer = null;
      }, LOADING_DELAY);
    }
  } else {
    delayedLoading.value = false;
  }
});

onBeforeUnmount(clearLoadingDelayTimer);

/** 首屏（还没成功取到过数据）用表格形状骨架，之后的刷新用转圈；偏好设置里 transition.loading 关掉则一律转圈 */
const showSkeleton = computed(
  () => !hasLoaded.value && preferences.transition.loading,
);

/**
 * 失败时表格里是否还留有可读的旧数据：有则只在表格上方出一条横幅（旧数据不该被顶掉），
 * 一行都没有时交给 #empty 显示原因与重试——那里读起来才是"这次查询失败了"而不是"没有数据"。
 * 只在 queryError 变化时求值，读到的就是本次失败后 vxe 保留下来的那份数据。
 */
const hasRowsOnError = computed(() => {
  if (!queryError.value) {
    return false;
  }
  const grid = props.api.grid as unknown as { getData?: () => any[] };
  return (grid?.getData?.() ?? []).length > 0;
});

function retryQuery() {
  props.api.query();
}

// 横幅走 vxe 的 top 插槽，它计入 getExcludeHeight，但失败路径不会触发 vxe 自己的重排，
// 所以要等横幅上屏后手动 recalculate，否则表体按没有横幅的高度铺满、分页被顶出卡片
watch(queryError, () => {
  nextTick(() => {
    const grid = props.api.grid as unknown as { recalculate?: (flag: boolean) => any };
    grid?.recalculate?.(true);
  });
});

const [Form, formApi] = useTableForm({
  handleSubmit: async () => {
    const formValues = formApi.form.values;
    formApi.setLatestSubmissionValues(toRaw(formValues));
    props.api.reload(formValues);
  },
  handleReset: async () => {
    await formApi.resetForm();
    const formValues = formApi.form.values;
    formApi.setLatestSubmissionValues(formValues);
    props.api.reload(formValues);
  },
  commonConfig: {
    componentProps: {
      class: 'w-full',
    },
  },
  showCollapseButton: true,
  submitButtonOptions: {
    content: $t('common.query'),
  },
  wrapperClass: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
});

const showTableTitle = computed(() => {
  return !!slots.tableTitle?.() || tableTitle.value;
});

const showToolbar = computed(() => {
  return (
    !!slots[TOOLBAR_ACTIONS]?.() ||
    !!slots[TOOLBAR_TOOLS]?.() ||
    showTableTitle.value
  );
});

const toolbarOptions = computed(() => {
  const slotActions = slots[TOOLBAR_ACTIONS]?.();
  const slotTools = slots[TOOLBAR_TOOLS]?.();

  const toolbarConfig: VxeGridPropTypes.ToolbarConfig = {
    tools:
      gridOptions.value?.toolbarConfig?.search && !!formOptions.value
        ? [
            {
              code: 'search',
              icon: 'vxe-icon--search',
              circle: true,
              status: showSearchForm.value ? 'primary' : undefined,
              title: $t('common.search'),
            },
          ]
        : [],
  };

  if (!showToolbar.value) {
    return { toolbarConfig };
  }

  // if (gridOptions.value?.toolbarConfig?.search) {
  // }
  // 强制使用固定的toolbar配置，不允许用户自定义
  // 减少配置的复杂度，以及后续维护的成本
  toolbarConfig.slots = {
    ...(slotActions || showTableTitle.value
      ? { buttons: TOOLBAR_ACTIONS }
      : {}),
    ...(slotTools ? { tools: TOOLBAR_TOOLS } : {}),
  };

  return { toolbarConfig };
});

const options = computed(() => {
  const globalGridConfig = VxeUI?.getConfig()?.grid ?? {};

  const mergedOptions: VxeTableGridProps = cloneDeep(
    mergeWithArrayOverride(
      {},
      toolbarOptions.value,
      // vxe 递归大类型会使 defu 泛型实例化超深（TS2589），按 Record 截断入参，
      // 合并结果仍由下方 VxeTableGridProps 注解约束。
      toRaw(gridOptions.value) as Record<string, any>,
      globalGridConfig,
    ),
  );

  if (mergedOptions.proxyConfig) {
    const { ajax } = mergedOptions.proxyConfig;
    mergedOptions.proxyConfig.enabled = !!ajax;
    // 不自动加载数据, 由组件控制
    mergedOptions.proxyConfig.autoLoad = false;
    // vxe 在 proxy 请求发出的瞬间就点亮自己的遮罩，本地快响应会闪一下；
    // 这里关掉它，遮罩改由本组件按 LOADING_DELAY 计时的 gridLoading 点亮
    mergedOptions.proxyConfig.showLoading = false;
  }

  if (mergedOptions.pagerConfig) {
    const mobileLayouts = [
      'PrevJump',
      'PrevPage',
      'Number',
      'NextPage',
      'NextJump',
    ] as any;
    const layouts = [
      'Total',
      'Sizes',
      'Home',
      ...mobileLayouts,
      'End',
    ] as readonly string[];
    mergedOptions.pagerConfig = mergeWithArrayOverride(
      {},
      mergedOptions.pagerConfig,
      {
        pageSize: 20,
        background: true,
        pageSizes: [10, 20, 30, 50, 100, 200],
        className: 'mt-2 w-full',
        layouts: isMobile.value ? mobileLayouts : layouts,
        size: 'mini' as const,
      },
    );
  }
  if (mergedOptions.formConfig) {
    mergedOptions.formConfig.enabled = false;
  }
  return mergedOptions;
});

// 单独成一条 computed：把它塞进 options 会让每次加载态翻转都重算 cloneDeep 的整包配置，
// 而 vxe 对 props.columns 是按引用监听的，重算一次就重装一次列
const gridLoading = computed(
  () => delayedLoading.value || !!gridOptions.value?.loading,
);

// 骨架行数上限跟分页大小同源（这里的 pageSize 已含本组件并入的默认值 20），再封顶；
// 实际铺几行由 table-skeleton 按遮罩实测高度折算，铺不满是遮罩本来就只有表体那一小条
const skeletonRows = computed(() => {
  const pageSize = Number(options.value?.pagerConfig?.pageSize) || 0;
  return Math.min(
    pageSize > 0 ? pageSize : MAX_SKELETON_ROWS,
    MAX_SKELETON_ROWS,
  );
});

function onToolbarToolClick(event: VxeGridDefines.ToolbarToolClickEventParams) {
  if (event.code === 'search') {
    props.api?.toggleSearchForm?.();
  }
  (
    gridEvents.value?.toolbarToolClick as VxeGridListeners['toolbarToolClick']
  )?.(event);
}

const events = computed(() => {
  return {
    ...gridEvents.value,
    toolbarToolClick: onToolbarToolClick,
  };
});

const delegatedSlots = computed(() => {
  const resultSlots: string[] = [];

  for (const key of Object.keys(slots)) {
    if (!['empty', 'form', 'loading', 'top', TOOLBAR_ACTIONS].includes(key)) {
      resultSlots.push(key);
    }
  }
  return resultSlots;
});

const delegatedFormSlots = computed(() => {
  const resultSlots: string[] = [];

  for (const key of Object.keys(slots)) {
    if (key.startsWith(FORM_SLOT_PREFIX)) {
      resultSlots.push(key);
    }
  }
  return resultSlots.map((key) => key.replace(FORM_SLOT_PREFIX, ''));
});

async function init() {
  await nextTick();
  const globalGridConfig = VxeUI?.getConfig()?.grid ?? {};
  const defaultGridOptions: VxeTableGridProps = mergeWithArrayOverride(
    {},
    toRaw(gridOptions.value),
    toRaw(globalGridConfig),
  );
  // vxe-table 递归大类型与 DeepPartial 展开相容性检查在部分 TS 版本下触发 TS2589，
  // 直接按目标成员类型断言，避免结构展开（与 api.ts setGridOptions 的传入方式等价）。
  props.api?.setState?.({
    gridOptions: defaultGridOptions as VxeGridProps['gridOptions'],
  });
  // form 由 vben-form 代替，所以需要保证query相关事件可以拿到参数
  extendProxyOptions(props.api, defaultGridOptions, () =>
    formApi.getLatestSubmissionValues(),
  );
  // 先包好 query 再发首个请求：VxeGrid 的 props 要到下一次渲染才换上包装后的函数，
  // 首屏那一次若走未包装的原始函数，hasLoaded / queryError 都不会被写入，
  // 加载态就只能由 vxe 自己的遮罩点亮（骨架没有原因，失败也没有留痕）。
  await nextTick();
  // 内部主动加载数据，防止form的默认值影响
  const autoLoad = defaultGridOptions.proxyConfig?.autoLoad;
  const enableProxyConfig = options.value.proxyConfig?.enabled;
  if (enableProxyConfig && autoLoad) {
    props.api.reload(formApi.form?.values ?? {});
  }

  // form 由 vben-form代替，所以不适配formConfig，这里给出警告
  const formConfig = gridOptions.value?.formConfig;
  // 处理某个页面加载多个Table时，第2个之后的Table初始化报出警告
  // 因为第一次初始化之后会把defaultGridOptions和gridOptions合并后缓存进State
  if (formConfig && formConfig.enabled) {
    console.warn(
      '[Vben Vxe Table]: The formConfig in the grid is not supported, please use the `formOptions` props',
    );
  }
}

// formOptions支持响应式
watch(
  formOptions,
  () => {
    formApi.setState((prev) => {
      const finalFormOptions: VbenFormProps = mergeWithArrayOverride(
        {},
        formOptions.value,
        prev,
      );
      return {
        ...finalFormOptions,
        collapseTriggerResize: !!finalFormOptions.showCollapseButton,
      };
    });
  },
  {
    immediate: true,
  },
);

onMounted(() => {
  props.api?.mount?.(gridRef.value, formApi);
  init();
});

onUnmounted(() => {
  formApi?.unmount?.();
  props.api?.unmount?.();
});
</script>

<template>
  <div :class="cn('bg-card h-full rounded-md', className)">
    <VxeGrid
      ref="gridRef"
      :class="
        cn(
          'p-2 pt-0',
          {
            'pt-0': showToolbar && !formOptions,
          },
          gridClass,
        )
      "
      v-bind="options"
      :loading="gridLoading"
      v-on="events"
    >
      <!-- 左侧操作区域或者title -->
      <template v-if="showToolbar" #toolbar-actions="slotProps">
        <slot v-if="showTableTitle" name="table-title">
          <div class="mr-1 pl-1 text-[1rem]">
            {{ tableTitle }}
            <VbenHelpTooltip v-if="tableTitleHelp" trigger-class="pb-1">
              {{ tableTitleHelp }}
            </VbenHelpTooltip>
          </div>
        </slot>
        <slot name="toolbar-actions" v-bind="slotProps"> </slot>
      </template>

      <!-- 继承默认的slot -->
      <template
        v-for="slotName in delegatedSlots"
        :key="slotName"
        #[slotName]="slotProps"
      >
        <slot :name="slotName" v-bind="slotProps"></slot>
      </template>

      <!-- form表单 -->
      <template #form>
        <div
          v-if="formOptions"
          v-show="showSearchForm !== false"
          class="relative rounded py-3 pb-4"
        >
          <slot name="form">
            <Form>
              <template
                v-for="slotName in delegatedFormSlots"
                :key="slotName"
                #[slotName]="slotProps"
              >
                <slot
                  :name="`${FORM_SLOT_PREFIX}${slotName}`"
                  v-bind="slotProps"
                ></slot>
              </template>
              <template #reset-before="slotProps">
                <slot name="reset-before" v-bind="slotProps"></slot>
              </template>
              <template #submit-before="slotProps">
                <slot name="submit-before" v-bind="slotProps"></slot>
              </template>
              <template #expand-before="slotProps">
                <slot name="expand-before" v-bind="slotProps"></slot>
              </template>
              <template #expand-after="slotProps">
                <slot name="expand-after" v-bind="slotProps"></slot>
              </template>
            </Form>
          </slot>
          <div
            class="bg-background-deep z-100 absolute -left-2 bottom-1 h-2 w-[calc(100%+1rem)] overflow-hidden md:bottom-2 md:h-3"
          ></div>
        </div>
      </template>
      <!-- 已有数据时刷新失败：旧数据留着可读，只在表格上方补一条带重试入口的提示 -->
      <template #top="slotProps">
        <div
          v-if="queryError && hasRowsOnError"
          class="bg-destructive/10 text-destructive border-destructive/40 mb-2 flex items-center gap-3 rounded-md border px-3 py-2 text-sm"
        >
          <span class="min-w-0 flex-1">{{ queryError }}</span>
          <VbenButton size="sm" @click="retryQuery">
            {{ $t('common.retry') }}
          </VbenButton>
        </div>
        <slot name="top" v-bind="slotProps"></slot>
      </template>
      <!-- loading -->
      <template #loading>
        <slot name="loading">
          <!-- vxe 每次装列 / 载数据都会点亮这层遮罩（isColLoading / isRowLoading），
               它既不看 proxy 的 showLoading 也不看我们的 250ms 阈值，底色已在 style.css 摘掉；
               所以可见性由这里按 delayedLoading 自己决定，快响应时槽里什么都不渲染 -->
          <div v-if="delayedLoading" class="absolute inset-0">
            <GridTableSkeleton
              v-if="showSkeleton"
              :columns="options.columns"
              :rows="skeletonRows"
            />
            <VbenLoading v-else :min-loading-time="0" :spinning="true" />
          </div>
        </slot>
      </template>
      <!-- 统一控状态 -->
      <template #empty>
        <slot name="empty">
          <!-- 首屏就失败：一行数据都没有，"暂无数据"会被读成查询成功但结果为空，必须换成错误态 -->
          <template v-if="queryError">
            <div class="text-destructive mt-4 px-4 text-center text-sm">
              {{ queryError }}
            </div>
            <VbenButton class="mt-3" size="sm" @click="retryQuery">
              {{ $t('common.retry') }}
            </VbenButton>
          </template>
          <template v-else>
            <EmptyIcon class="mx-auto" />
            <div class="mt-2">{{ $t('common.noData') }}</div>
          </template>
        </slot>
      </template>
    </VxeGrid>
  </div>
</template>
