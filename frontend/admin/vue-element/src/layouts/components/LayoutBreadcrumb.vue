<template>
  <div ref="wrapRef" v-show="visible && !hiddenByWidth" class="breadcrumb-wrap">
    <el-breadcrumb class="breadcrumb" :class="breadcrumbClass" separator=">">
    <!-- 首页图标 -->
    <el-breadcrumb-item
      v-if="breadcrumbPrefs.showHome"
      :to="{ path: '/' }"
      class="breadcrumb__home"
    >
      <SvgIcon
        v-if="breadcrumbPrefs.showIcon"
        icon="lucide:house"
        :size="14"
        class="breadcrumb__icon"
      />
      <span class="breadcrumb__text">{{ $t("common.breadcrumb.home") }}</span>
    </el-breadcrumb-item>
    <el-breadcrumb-item v-for="(item, index) in breadcrumbs" :key="item.path">
      <span
        v-if="item.redirect === 'noredirect' || index === breadcrumbs.length - 1"
        class="breadcrumb__current"
      >
        <SvgIcon
          v-if="breadcrumbPrefs.showIcon && item.meta?.icon"
          :icon="item.meta.icon as string"
          :size="14"
          class="breadcrumb__item-icon"
        />
        <span class="breadcrumb__text">{{
          translateRouteTitle((item.meta.title as string) ?? "")
        }}</span>
      </span>
      <a v-else @click.prevent="handleLink(item)">
        <SvgIcon
          v-if="breadcrumbPrefs.showIcon && item.meta?.icon"
          :icon="item.meta.icon as string"
          :size="14"
          class="breadcrumb__item-icon"
        />
        <span class="breadcrumb__text">{{
          translateRouteTitle((item.meta.title as string) ?? "")
        }}</span>
      </a>
    </el-breadcrumb-item>
  </el-breadcrumb>
  </div>
</template>

<script setup lang="ts">
import { RouteLocationMatched } from "vue-router";
import { compile } from "path-to-regexp";

import { router } from "@/router";
import { translateRouteTitle } from "@/core/i18n";
import { preferences } from "@/core/preferences";
import SvgIcon from "@/components/SvgIcon/index.vue";

const currentRoute = useRoute();
const pathCompile = (path: string) => {
  const { params } = currentRoute;
  const toPath = compile(path);
  return toPath(params);
};

// 面包屑偏好
const breadcrumbPrefs = computed(() => preferences.breadcrumb);

const breadcrumbs = ref<Array<RouteLocationMatched>>([]);

// 是否可见：启用 + 不只有一个时隐藏检查
const visible = computed(() => {
  if (!breadcrumbPrefs.value.enable) return false;
  return !(breadcrumbPrefs.value.hideOnlyOne && breadcrumbs.value.length <= 1);
});

// 面包屑样式类
const breadcrumbClass = computed(() => {
  return {
    "breadcrumb--background": breadcrumbPrefs.value.styleType === "background",
  };
});

// ======== 过窄整体隐藏（对齐 vben：宁可无面包屑，不要挤压变形） ========
// 自然宽度（scrollWidth）超过容器可用宽度 → 整体隐藏；容器变宽到能容纳上次
// 自然宽度 → 恢复。双观测：自身（溢出判定）+ 父容器（隐藏后自身 display:none
// 不再触发 RO，靠父容器尺寸变化唤醒复检）。
const wrapRef = ref<HTMLDivElement | null>(null);
const hiddenByWidth = ref(false);
const checkRef = ref<() => void>(() => {});
let lastNaturalWidth = 0;

onMounted(() => {
  const el = wrapRef.value;
  if (!el || !window.ResizeObserver) return;
  const check = () => {
    // 判据 = 「文本层是否被截断」：内层面包屑自带收缩+省略号（.breadcrumb__text），
    // 挤压会被它吸收，scrollWidth 永不溢出——只要任一文案 span 被截断
    // （scrollWidth > clientWidth），就整体隐藏（对齐 vben：宁可无面包屑）
    const inner = el.querySelector(".breadcrumb") ?? el;
    const textSpans = Array.from(inner.querySelectorAll(".breadcrumb__text"));
    const truncated = textSpans.some((s) => s.scrollWidth > s.clientWidth + 1);
    const overflows = inner.scrollWidth > el.clientWidth + 1;
    const natural = Math.max(
      inner.scrollWidth,
      el.scrollWidth,
      ...textSpans.map((s) => s.scrollWidth),
      1
    );
    if (!hiddenByWidth.value) {
      if (truncated || overflows) {
        lastNaturalWidth = natural;
        hiddenByWidth.value = true;
      }
    } else {
      const parentW = (el.parentElement ?? el).clientWidth;
      if (parentW >= lastNaturalWidth - 1) {
        hiddenByWidth.value = false;
      }
    }
  };
  checkRef.value = check;
  const ro = new ResizeObserver(check);
  ro.observe(el);
  if (el.parentElement) ro.observe(el.parentElement);
  onBeforeUnmount(() => ro.disconnect());
});

// 路由切换 → 项文案变化 → 自然宽变化：显式复检（ResizeObserver 不感知文本变化，
// 隐藏标签页里它也不推进，靠这条兜底；flush:post 保证读到更新后的 DOM）
watch([breadcrumbs, visible], () => checkRef.value?.(), { flush: "post", immediate: true });

function getBreadcrumb() {
  breadcrumbs.value = currentRoute.matched.filter(
    (item) =>
      item.meta && item.meta.title && item.meta.breadcrumb !== false && !item.meta.hideInBreadcrumb
  );
}

function handleLink(item: any) {
  const { redirect, path } = item;
  if (redirect) {
    router.push(redirect).then(
      () => {},
      (err) => {
        console.warn(err);
      }
    );
    return;
  }
  router.push(pathCompile(path)).then(
    () => {},
    (err) => {
      console.warn(err);
    }
  );
}

watch(
  () => currentRoute.path,
  () => {
    getBreadcrumb();
  }
);

onBeforeMount(() => {
  getBreadcrumb();
});
</script>

<style lang="scss" scoped>
// 过窄隐藏的测量壳：可收缩到头部分配宽，内部 el-breadcrumb 溢出量即「自然宽 − 可用宽」；
// align-items:center 修正 EP 根元素行高把壳撑高后内容顶在上方的不居中（实测 dy −4.3px）
.breadcrumb-wrap {
  display: flex;
  align-items: center;
  min-width: 0;
  overflow: hidden;
}

.breadcrumb {
  display: flex;
  align-items: center;
  font-size: 13px;

  // background 风格
  &--background {
    :deep(.el-breadcrumb__item) {
      .el-breadcrumb__inner {
        padding: 2px 8px;
        border-radius: 4px;
        transition: background-color 0.2s;
      }

      &:not(:last-child) .el-breadcrumb__inner {
        background-color: var(--el-fill-color-light);
      }

      &:not(:last-child) .el-breadcrumb__inner:hover {
        background-color: var(--el-fill-color);
      }
    }
  }

  &__home {
    :deep(.el-breadcrumb__inner) {
      display: inline-flex;
      align-items: center;
    }
  }

  &__icon,
  &__item-icon {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
    color: currentColor;
  }
}
</style>

<!--
  面包屑颜色/字重样式用非 scoped 全局块，确保 :last-child 等选择器
  的特异性能覆盖 Element Plus 默认样式，不再依赖 scoped + :deep() 组合
-->
<style lang="scss">
.breadcrumb {
  // 与按钮高度匹配，line-height: normal 让 flex 的 align-items: center 负责垂直居中
  line-height: normal !important;

  // 垂直居中修正（2026-11-08）：_layouts.scss 给 .breadcrumb 留了 margin-bottom 8px
  // （筛选区旧排版遗产），flex 头部按 margin 盒居中 → 可见盒顶在上沿（实测 dy −4.3px）。
  // 归零后可见盒与按钮同中心。
  margin-bottom: 0 !important;

  // ======== 窄屏单行化（2026-11-08，react 3223e3f5 同款根修） ========
  // 头部可用宽一窄时，EP 面包屑的项被压到两字符宽、CJK 逐字换行竖排成
  // 「首/页」「概/览」。这里改全局单行：根不放大头部（min-width:0 + overflow:hidden
  // 兜底硬裁），首页项钉死不缩，其余项等比收缩、文本截断出省略号；收缩项 36px
  // 药丸下限，挤压时收成「图标+…」而非消失。空间充足时零视觉变化。
  min-width: 0;
  overflow: hidden;
  flex-wrap: nowrap;

  .el-breadcrumb__item {
    display: flex;
    align-items: center;
    white-space: nowrap;
  }

  // 首页项钉死不缩；分隔符（EP 把分隔符渲染在 item 内部）同样不缩
  .el-breadcrumb__item.breadcrumb__home,
  .el-breadcrumb__separator {
    flex-shrink: 0;
  }

  // 可收缩项的药丸下限：图标 16 + gap 4 + "…" ~10 + 分隔符 ~23 + padding 8，
  // 保证被挤压的项收成「图标+…」而不是直接消失（消失会让相邻分隔符邻接成 //）
  .el-breadcrumb__item:not(.breadcrumb__home) {
    min-width: 36px;
  }

  // 收缩链路逐层放行：item → inner → 文本 span。中间层不裁剪，
  // 省略号只在文本层渲染（图标在文本前面）。
  .el-breadcrumb__inner {
    display: inline-flex;
    align-items: center;
    min-width: 0;
    max-width: 100%;
  }

  .breadcrumb__text {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  // home图标居中对齐
  .breadcrumb__home .el-breadcrumb__inner {
    display: inline-flex !important;
    align-items: center !important;
  }

  // 非当前页：中灰 + 常规字重
  .el-breadcrumb__inner,
  .el-breadcrumb__inner a {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-weight: 400 !important;
    font-size: 13px !important; // 对齐 react 面包屑基准字号（2026-11-08，原 15px）
    color: var(--el-text-color-regular) !important;
    transition: color 0.2s ease;
    line-height: normal !important;
    height: auto !important;
    padding: 0; // 间距交给分隔符 margin，对齐 react（antd 分隔符 padding 0 8px）
  }

  // 当前页（最后一项）：主文字色 + 半粗体，视觉焦点
  .el-breadcrumb__item:last-child .el-breadcrumb__inner {
    color: var(--el-text-color-primary) !important;
    font-weight: 600 !important;
  }

  // 当前页（最后一项）内部 .breadcrumb__current span 也必须与 inner 同色
  .el-breadcrumb__item:last-child .breadcrumb__current {
    color: var(--el-text-color-primary) !important;
    font-weight: 600 !important;
  }

  // 可点击链接 hover
  .el-breadcrumb__inner a:hover {
    color: var(--gowind-primary-text) !important;
  }

  // 分隔符
  .el-breadcrumb__separator {
    margin: 0 8px; // 对齐 react（antd 分隔符 padding 0 8px，原 EP 默认 0 9px）
    color: var(--el-text-color-secondary);
  }

  // background 风格：浅色分支使用 Element Plus fill token，避免白底白芯片在亮色模式下不可见
  // 暗色模式由下方 html.dark & 分支覆盖，保持原有外观
  &.breadcrumb--background {
    .el-breadcrumb__item:not(:last-child) .el-breadcrumb__inner {
      background-color: var(--el-fill-color-light);
    }
    .el-breadcrumb__item:not(:last-child) .el-breadcrumb__inner:hover {
      background-color: var(--el-fill-color);
    }
  }

  // ======== 暗色模式 ========
  // 文字色不再在此重写：上面各条已改用 --el-text-color-* 变量，暗色分支由
  // styles/_dark-mode.scss 统一供值（此前这里写死 #d9d9d9/#ffffff/#595959，
  // 其中分隔符 #595959 在 L1 表面 #111827 上实测仅 2.53:1）。
  html.dark & {
    &.breadcrumb--background {
      .el-breadcrumb__item:not(:last-child) .el-breadcrumb__inner {
        background-color: rgba(255, 255, 255, 0.06);
      }
      .el-breadcrumb__item:not(:last-child) .el-breadcrumb__inner:hover {
        background-color: rgba(255, 255, 255, 0.1);
      }
    }
  }
}
</style>
