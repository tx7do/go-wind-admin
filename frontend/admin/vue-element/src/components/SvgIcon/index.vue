<!--
  统一图标组件

  将项目中所有图标渲染统一到 Iconify 体系：
  - `lucide:users`       → @iconify/vue 渲染 Lucide 图标
  - `ep:setting`         → @iconify/vue 渲染 Element Plus 图标
  - `fa:user`            → @iconify/vue 渲染 Font Awesome 图标
  - `svg:menu`           → 内联 SVG 渲染本地 SVG 文件
  - `el-icon-Setting`    → 兼容旧格式，自动转为 `ep:setting`
  - `menu`               → 无前缀兜底，当作本地 SVG 处理

  扩展新图标集只需安装对应的 @iconify-json/xxx 包即可。
-->
<template>
  <!-- 本地 SVG 图标：通过内联 SVG 渲染 -->
  <!-- eslint-disable-next-line vue/no-v-html -- svgContent 来自构建时打包的本地 SVG 资源，非用户输入 -->
  <span
    v-if="isSvgIcon"
    :class="[attrs.class, 'svg-local-icon']"
    :style="svgStyle"
    v-html="svgContent"
  />
  <!-- Iconify 图标：用 @iconify/vue 组件渲染 -->
  <Icon v-else :icon="resolvedIcon" :width="iconSize" :height="iconSize" :class="attrs.class" />
</template>

<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { computed, ref, watchEffect } from "vue";

defineOptions({ inheritAttrs: false });

const props = defineProps<{
  /** 图标名称，支持 Iconify 格式（lucide:users）和旧格式（el-icon-Setting） */
  icon?: string;
  /** 图标尺寸 */
  size?: number | string;
}>();

const attrs = useAttrs();

// 预加载所有本地 SVG 文件（Vite raw import）
const svgModules = import.meta.glob("../../assets/icons/*.svg", {
  query: "?raw",
  eager: true,
  import: "default",
}) as Record<string, string>;

/**
 * 将旧格式图标名转为 Iconify 标准格式
 */
const resolvedIcon = computed(() => {
  if (!props.icon) return "";

  // 旧格式兼容：el-icon-Setting → ep:setting
  if (props.icon.startsWith("el-icon-")) {
    const name = props.icon.replace("el-icon-", "");
    return `ep:${name.charAt(0).toLowerCase() + name.slice(1)}`;
  }

  // 已经是 Iconify 格式（lucide:users, fa:user, ep:setting 等）
  if (props.icon.includes(":")) {
    return props.icon;
  }

  // 无前缀：本地 SVG 图标
  return `svg:${props.icon}`;
});

/**
 * 判断是否使用本地 SVG 方式渲染
 */
const isSvgIcon = computed(() => {
  return resolvedIcon.value.startsWith("svg:");
});

/**
 * 提取本地 SVG 名称
 */
const svgName = computed(() => {
  if (!isSvgIcon.value) return "";
  return resolvedIcon.value.replace("svg:", "");
});

/**
 * 从预加载的模块中获取 SVG 内容
 */
const svgRaw = computed(() => {
  const name = svgName.value;
  if (!name) return "";

  // 匹配任意路径下的 {name}.svg
  for (const [path, content] of Object.entries(svgModules)) {
    if (path.endsWith(`/${name}.svg`)) {
      return content as string;
    }
  }
  return "";
});

/**
 * SVG 内容：注入 width="1em" height="1em" 确保继承父容器尺寸
 */
const svgContent = computed(() => {
  const raw = svgRaw.value;
  if (!raw) return "";
  // 统一设置 width="1em" height="1em"，确保 SVG 通过 font-size 控制尺寸
  return raw.replace(/<svg\b[^>]*>/, (match) => {
    // 先移除已有的 width/height 属性，再统一注入
    const cleaned = match.replace(/\s+(width|height)="[^"]*"/g, "");
    return cleaned.replace(/<svg\b/, '<svg width="1em" height="1em"');
  });
});

/**
 * 解析后的图标尺寸（Iconify 组件用）
 * 不传 size 时为 undefined，让 Iconify 使用默认 1em
 */
const iconSize = computed(() => props.size || undefined);

/**
 * 内联样式：通过 font-size 控制 SVG 尺寸（1em 基准），color 控制 SVG 颜色（currentColor）
 */
const svgStyle = computed(() => {
  const base: Record<string, string> = {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    verticalAlign: "middle",
  };
  if (props.size) {
    base.width = `${props.size}px`;
    base.height = `${props.size}px`;
    base.fontSize = `${props.size}px`;
  }
  return base;
});
</script>
