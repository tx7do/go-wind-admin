import type { EChartsOption } from "echarts";

import type EchartsUI from "./echarts-ui.vue";

import type { Ref } from "vue";
import { computed, watch } from "vue";

import { usePreferences } from "@go-wind/admin-element";

import {
  tryOnUnmounted,
  useDebounceFn,
  useResizeObserver,
  useTimeoutFn,
  useWindowSize,
} from "@vueuse/core";

import echarts from "./echarts";

type EchartsUIType = typeof EchartsUI | undefined;

type EchartsThemeType = "dark" | "light" | null;

function useEcharts(chartRef: Ref<EchartsUIType>) {
  let chartInstance: echarts.ECharts | null = null;
  let cacheOptions: EChartsOption = {};

  const { isDark } = usePreferences();
  const { height, width } = useWindowSize();
  const resizeHandler: () => void = useDebounceFn(resize, 200);

  const getOptions = computed((): EChartsOption => {
    if (!isDark.value) {
      return {};
    }

    return {
      backgroundColor: "transparent",
    };
  });

  // 容器是否已有真实布局尺寸。注意 chartRef 是 EchartsUI 组件实例，
  // 必须量 $el 的 clientWidth/Height——实例本身没有 offsetHeight，
  // 旧实现 chartRef.value?.offsetHeight === 0 恒为 false，是永不生效的死检查。
  const layoutSize = (): { el?: HTMLElement; w: number; h: number } => {
    const el = chartRef?.value?.$el as HTMLElement | undefined;
    return { el, w: el?.clientWidth ?? 0, h: el?.clientHeight ?? 0 };
  };

  const initCharts = (t?: EchartsThemeType) => {
    const { el, w, h } = layoutSize();
    if (!el || w === 0 || h === 0) {
      return;
    }
    chartInstance = echarts.init(el, t || isDark.value ? "dark" : null);

    return chartInstance;
  };

  const renderEcharts = (options: EChartsOption, clear = true) => {
    cacheOptions = options;
    const currentOptions = {
      ...options,
      ...getOptions.value,
    };
    return new Promise((resolve) => {
      const tick = () => {
        const { el, w, h } = layoutSize();
        // 未挂载或容器尚未完成布局（v-if 切换瞬间、隐藏容器、后台标签页）时
        // 不能 echarts.init——画布尺寸会是 0 并告警 "Can't get DOM width or height"，
        // 轮询到真实尺寸后再渲染；useTimeoutFn 随组件卸载自动停止
        if (!el || w === 0 || h === 0) {
          useTimeoutFn(tick, 30);
          return;
        }
        // v-if 切走再切回（图表/表格视图切换、折叠展开）会重挂载出全新 DOM，
        // 旧实例仍附着在已脱离文档的节点上，setOption 画得再对也不可见——必须重建
        if (chartInstance && (chartInstance.getDom() !== el || chartInstance.isDisposed())) {
          chartInstance.dispose();
          chartInstance = null;
        }
        if (!chartInstance) {
          chartInstance = echarts.init(el, isDark.value ? "dark" : null);
        }
        if (chartInstance.isDisposed()) {
          resolve(null);
          return;
        }
        if (clear) {
          chartInstance?.clear();
        }
        chartInstance?.setOption(currentOptions);
        resolve(null);
      };
      tick();
    });
  };

  function resize() {
    chartInstance?.resize({
      animation: {
        duration: 300,
        easing: "quadraticIn",
      },
    });
  }

  watch([width, height], () => {
    resizeHandler?.();
  });

  useResizeObserver(chartRef as never, resizeHandler);

  watch(isDark, () => {
    if (chartInstance && !chartInstance.isDisposed()) {
      chartInstance.dispose();
      chartInstance = null;
      initCharts();
      renderEcharts(cacheOptions);
      resize();
    }
  });

  tryOnUnmounted(() => {
    // 销毁实例，释放资源；同时置空引用，防止组件卸载后
    // 仍在排队的延迟渲染回调触达已销毁实例（报 "has been disposed"）
    chartInstance?.dispose();
    chartInstance = null;
  });
  return {
    renderEcharts,
    resize,
  };
}

export { useEcharts };

export type { EchartsUIType };
