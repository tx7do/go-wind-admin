<template>
  <Transition name="skeleton-fade">
    <div v-if="visible" class="page-skeleton">
      <!-- ====== Dashboard 骨架 ====== -->
      <template v-if="type === 'dashboard'">
        <!-- KPI 卡片行 -->
        <div class="sk-dash-row">
          <div v-for="i in 4" :key="i" class="sk-card sk-card--kpi" :style="stagger(i * 0.07)">
            <div class="sk-kpi-head">
              <div class="sk-bar sk-bar--s sk-bar--subtle" style="width: 72px" />
              <div class="sk-icon-box" />
            </div>
            <div class="sk-bar sk-bar--xl" style="width: 96px; margin: 10px 0 14px" />
            <div class="sk-kpi-foot">
              <div class="sk-bar sk-bar--xs sk-bar--subtle" style="width: 56px" />
              <div class="sk-tag sk-tag--sm" :class="i % 2 === 0 ? 'sk-tag--up' : 'sk-tag--down'" />
            </div>
          </div>
        </div>

        <!-- 主图表卡片 -->
        <div class="sk-card sk-card--chart-main">
          <div class="sk-tabs">
            <div class="sk-tab sk-tab--active" />
            <div class="sk-tab" />
          </div>
          <div class="sk-chart-area">
            <svg class="sk-waveform" viewBox="0 0 800 240" preserveAspectRatio="none">
              <defs>
                <linearGradient id="sk-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="currentColor" stop-opacity="0.12" />
                  <stop offset="100%" stop-color="currentColor" stop-opacity="0.02" />
                </linearGradient>
              </defs>
              <path
                d="M0,180 C80,60 160,120 240,80 C320,40 400,140 480,100 C560,60 640,130 720,50 L800,70 L800,240 L0,240 Z"
                fill="url(#sk-grad)"
              />
              <path
                d="M0,180 C80,60 160,120 240,80 C320,40 400,140 480,100 C560,60 640,130 720,50 L800,70"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                opacity="0.2"
              />
            </svg>
          </div>
        </div>

        <!-- 3 小图表卡片 -->
        <div class="sk-dash-row sk-dash-row--3">
          <div
            v-for="i in 3"
            :key="i"
            class="sk-card sk-card--chart-sm"
            :style="stagger(i * 0.08 + 0.2)"
          >
            <div class="sk-bar sk-bar--s" style="width: 80px; margin-bottom: 16px" />
            <div class="sk-chart-area sk-chart-area--sm">
              <svg class="sk-waveform" viewBox="0 0 400 200" preserveAspectRatio="none">
                <path
                  v-if="i === 1"
                  d="M0,100 C60,30 120,80 180,50 C240,20 300,90 400,60 L400,200 L0,200 Z"
                  fill="url(#sk-grad)"
                />
                <path
                  v-if="i === 1"
                  d="M0,100 C60,30 120,80 180,50 C240,20 300,90 400,60"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                  opacity="0.2"
                />
                <template v-if="i === 2">
                  <line
                    v-for="y in [40, 80, 120, 160]"
                    :key="y"
                    x1="60"
                    :y1="y"
                    x2="340"
                    :y2="y"
                    stroke="currentColor"
                    stroke-width="1"
                    opacity="0.06"
                  />
                  <rect
                    x="80"
                    y="50"
                    width="40"
                    height="130"
                    rx="4"
                    fill="currentColor"
                    opacity="0.06"
                  />
                  <rect
                    x="140"
                    y="90"
                    width="40"
                    height="90"
                    rx="4"
                    fill="currentColor"
                    opacity="0.06"
                  />
                  <rect
                    x="200"
                    y="30"
                    width="40"
                    height="150"
                    rx="4"
                    fill="currentColor"
                    opacity="0.08"
                  />
                  <rect
                    x="260"
                    y="70"
                    width="40"
                    height="110"
                    rx="4"
                    fill="currentColor"
                    opacity="0.06"
                  />
                </template>
                <template v-if="i === 3">
                  <circle
                    cx="200"
                    cy="100"
                    r="70"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="18"
                    opacity="0.06"
                  />
                  <circle
                    cx="200"
                    cy="100"
                    r="70"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="18"
                    opacity="0.1"
                    stroke-dasharray="110 330"
                    stroke-dashoffset="-55"
                  />
                </template>
              </svg>
            </div>
          </div>
        </div>
      </template>

      <!-- ====== 表格列表骨架（默认） ====== -->
      <template v-else>
        <!-- 面包屑 -->
        <div class="sk-breadcrumb">
          <div class="sk-bar sk-bar--xs sk-bar--subtle" style="width: 36px" />
          <div class="sk-breadcrumb__sep" />
          <div class="sk-bar sk-bar--xs sk-bar--subtle" style="width: 52px" />
          <div class="sk-breadcrumb__sep" />
          <div class="sk-bar sk-bar--xs" style="width: 64px" />
        </div>

        <!-- 工具栏 -->
        <div class="sk-toolbar">
          <div class="sk-toolbar__left">
            <div class="sk-btn sk-btn--primary">
              <div class="sk-btn__icon" />
            </div>
            <div class="sk-btn">
              <div class="sk-btn__icon sk-btn__icon--sm" />
            </div>
          </div>
          <div class="sk-toolbar__right">
            <div class="sk-btn-circle" />
            <div class="sk-btn-circle" />
            <div class="sk-btn-circle" />
          </div>
        </div>

        <!-- 搜索栏 -->
        <div class="sk-search">
          <div v-for="i in 3" :key="i" class="sk-field" :style="stagger(i * 0.06)">
            <div class="sk-field__label" />
            <div class="sk-field__input" :style="{ width: i === 3 ? '140px' : undefined }" />
          </div>
          <div class="sk-field__btns">
            <div class="sk-btn sk-btn--primary sk-btn--sm" />
            <div class="sk-btn sk-btn--ghost sk-btn--sm" />
          </div>
        </div>

        <!-- 表格卡片 -->
        <div class="sk-table-card">
          <!-- 表头 -->
          <div class="sk-thead">
            <div class="sk-checkbox" />
            <div class="sk-th" style="width: 18%">#</div>
            <div class="sk-th" style="width: 20%" />
            <div class="sk-th" style="width: 12%" />
            <div class="sk-th" style="width: 16%" />
            <div class="sk-th" style="width: 18%" />
            <div class="sk-th sk-th--end" />
          </div>
          <!-- 行 -->
          <div v-for="i in 8" :key="i" class="sk-trow" :style="stagger(i * 0.04)">
            <div class="sk-checkbox" />
            <!-- 序号列 -->
            <div class="sk-td">
              <div class="sk-bar sk-bar--xs sk-bar--subtle" style="width: 24px" />
            </div>
            <!-- 名称列：头像 + 文本 -->
            <div class="sk-td sk-td--name">
              <div class="sk-avatar" :class="`sk-avatar--c${(i % 4) + 1}`" />
              <div class="sk-name-group">
                <div class="sk-bar" :style="{ width: nameWidths[i - 1] }" />
              </div>
            </div>
            <!-- 状态标签列 -->
            <div class="sk-td">
              <div class="sk-tag" :class="tagColors[i - 1]" :style="{ width: tagWidths[i - 1] }" />
            </div>
            <!-- 组织列 -->
            <div class="sk-td">
              <div class="sk-bar sk-bar--s sk-bar--subtle" :style="{ width: orgWidths[i - 1] }" />
            </div>
            <!-- 时间列 -->
            <div class="sk-td">
              <div
                class="sk-bar sk-bar--s sk-bar--mono sk-bar--subtle"
                :style="{ width: dateWidths[i - 1] }"
              />
            </div>
            <!-- 操作列 -->
            <div class="sk-td sk-td--end">
              <div class="sk-link-bar" />
              <div class="sk-link-sep" />
              <div class="sk-link-bar" />
            </div>
          </div>
        </div>

        <!-- 分页器 -->
        <div class="sk-pagination">
          <div class="sk-bar sk-bar--xs sk-bar--subtle" style="width: 120px" />
          <div class="sk-pagi-pages">
            <div
              v-for="p in 5"
              :key="p"
              class="sk-pagi-btn"
              :class="{ 'sk-pagi-btn--active': p === 2 }"
            />
            <div class="sk-pagi-btn sk-pagi-btn--nav" />
          </div>
        </div>
      </template>
    </div>
  </Transition>
</template>

<script lang="ts">
export type SkeletonType = "table" | "dashboard";

// 根据路由路径自动推断骨架屏类型（供外部使用）
export function resolveSkeletonType(path: string): SkeletonType {
  if (path.includes("/analytics") || path.includes("/dashboard")) return "dashboard";
  return "table";
}
</script>

<script lang="ts" setup>
withDefaults(
  defineProps<{
    visible: boolean;
    type?: "table" | "dashboard";
  }>(),
  { type: "table" }
);

// 各列随机宽度
const nameWidths = ["52%", "64%", "48%", "72%", "56%", "60%", "44%", "68%"];
const tagWidths = ["48px", "56px", "40px", "64px", "44px", "52px", "48px", "60px"];
const orgWidths = ["60%", "45%", "70%", "50%", "65%", "55%", "40%", "58%"];
const dateWidths = ["72%", "68%", "76%", "64%", "72%", "68%", "76%", "70%"];
const tagColors = [
  "sk-tag--success",
  "sk-tag--primary",
  "sk-tag--warning",
  "sk-tag--success",
  "sk-tag--danger",
  "sk-tag--primary",
  "sk-tag--warning",
  "sk-tag--success",
];

// 交错延迟
function stagger(delay: number) {
  return { animationDelay: `${delay}s` };
}
</script>

<style lang="scss" scoped>
// =============== 容器 ===============
.page-skeleton {
  position: absolute;
  inset: 0;
  padding: 20px;
  pointer-events: none;
  z-index: 1;
  overflow: hidden;
}

// =============== 基础骨架条 ===============
// 所有可见骨架元素共享的微光脉冲动画
@mixin skeleton-pulse {
  position: relative;
  overflow: hidden;
  background: var(--el-fill-color-light);
  border-radius: 3px;

  &::after {
    content: "";
    position: absolute;
    inset: 0;
    background: linear-gradient(
      90deg,
      transparent 0%,
      var(--el-fill-color) 40%,
      var(--el-fill-color-lighter) 60%,
      transparent 100%
    );
    background-size: 300% 100%;
    animation: sk-shimmer 2.4s cubic-bezier(0.4, 0, 0.2, 1) infinite;
  }
}

// 暗黑模式微光调整
:global(html.dark) {
  .sk-bar,
  .sk-field__input,
  .sk-field__label,
  .sk-icon-box,
  .sk-avatar,
  .sk-tag,
  .sk-checkbox,
  .sk-btn,
  .sk-btn-circle,
  .sk-pagi-btn,
  .sk-chart-area,
  .sk-tab {
    &::after {
      background: linear-gradient(
        90deg,
        transparent 0%,
        rgba(255, 255, 255, 0.06) 40%,
        rgba(255, 255, 255, 0.1) 60%,
        transparent 100%
      );
      background-size: 300% 100%;
    }
  }
}

// --- 骨架条 ---
.sk-bar {
  @include skeleton-pulse;
  height: 12px;
  flex-shrink: 0;
  animation: sk-appear 0.5s ease both;

  &--xs {
    height: 10px;
  }
  &--s {
    height: 11px;
  }
  &--xl {
    height: 26px;
  }
  &--subtle {
    opacity: 0.65;
  }
  &--mono {
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.3px;
  }
}

// --- 头像 ---
.sk-avatar {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  flex-shrink: 0;
  @include skeleton-pulse;

  &--c1 {
    border-radius: 50%;
  }
  &--c2 {
    border-radius: 8px;
  }
  &--c3 {
    border-radius: 50%;
  }
  &--c4 {
    border-radius: 8px;
  }
}

// --- 标签 ---
.sk-tag {
  @include skeleton-pulse;
  height: 22px;
  border-radius: 4px;
  flex-shrink: 0;
  animation: sk-appear 0.5s ease both;

  &--sm {
    height: 18px;
    width: 48px;
  }
  &--primary {
    background: var(--el-color-primary-light-8);
  }
  &--success {
    background: var(--el-color-success-light-8);
  }
  &--warning {
    background: var(--el-color-warning-light-8);
  }
  &--danger {
    background: var(--el-color-danger-light-8);
  }
  &--up {
    background: var(--el-color-success-light-8);
    width: 48px;
    border-radius: 4px;
  }
  &--down {
    background: var(--el-color-danger-light-8);
    width: 48px;
    border-radius: 4px;
  }
}

// --- 复选框 ---
.sk-checkbox {
  width: 15px;
  height: 15px;
  border-radius: 3px;
  border: 1.5px solid var(--el-border-color);
  flex-shrink: 0;
  margin: 0 14px 0 16px;
  transition: border-color 0.2s;
}

// --- 按钮 ---
.sk-btn {
  @include skeleton-pulse;
  height: 32px;
  border-radius: 6px;
  width: 72px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  animation: sk-appear 0.4s ease both;

  &--primary {
    background: var(--el-color-primary-light-7);
  }
  &--ghost {
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
  }
  &--sm {
    width: 64px;
    height: 32px;
  }

  &__icon {
    width: 14px;
    height: 14px;
    border-radius: 2px;
    background: rgba(255, 255, 255, 0.6);

    &--sm {
      width: 10px;
      height: 10px;
    }
  }
}

.sk-btn-circle {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  @include skeleton-pulse;
  flex-shrink: 0;
  background: transparent;
  border: 1px solid var(--el-border-color-lighter);
}

// --- 链接（操作列） ---
.sk-link-bar {
  width: 28px;
  height: 8px;
  border-radius: 2px;
  background: var(--el-color-primary-light-5);
  opacity: 0.4;
  flex-shrink: 0;
}

.sk-link-sep {
  width: 1px;
  height: 12px;
  background: var(--el-border-color-lighter);
  margin: 0 8px;
  flex-shrink: 0;
}

// --- 动画 ---
@keyframes sk-shimmer {
  0% {
    background-position: 300% 0;
  }
  100% {
    background-position: -300% 0;
  }
}

@keyframes sk-appear {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

// =============== Dashboard 骨架 ===============
.sk-dash-row {
  display: grid;
  gap: 16px;
  margin-bottom: 16px;
  grid-template-columns: repeat(4, 1fr);

  &--3 {
    grid-template-columns: repeat(3, 1fr);
  }
}

.sk-card {
  background: var(--el-bg-color);
  border-radius: 10px;
  border: 1px solid var(--el-border-color-lighter);
  padding: 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03);

  &--kpi {
    animation: sk-appear 0.5s ease both;
  }

  &--chart-main {
    margin-bottom: 16px;
    animation: sk-appear 0.5s ease both;
  }

  &--chart-sm {
    animation: sk-appear 0.5s ease both;
  }
}

:global(html.dark) {
  .sk-card {
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.15);
  }
}

.sk-kpi-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 2px;
}

.sk-icon-box {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  @include skeleton-pulse;
  background: var(--el-color-primary-light-9);
}

.sk-kpi-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 10px;
  margin-top: 4px;
  border-top: 1px solid var(--el-border-color-extra-light);
}

.sk-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 14px;
}

.sk-tab {
  width: 64px;
  height: 28px;
  border-radius: 6px;
  @include skeleton-pulse;

  &--active {
    background: var(--el-color-primary-light-8);
  }
}

.sk-chart-area {
  height: 320px;
  border-radius: 4px;
  background: var(--el-fill-color-lighter);
  position: relative;
  overflow: hidden;
  margin: 0 -4px;

  &--sm {
    height: 240px;
  }
}

.sk-waveform {
  position: absolute;
  inset: 20px;
  width: calc(100% - 40px);
  height: calc(100% - 40px);
  color: var(--el-color-primary-light-5);
}

// =============== 表格列表骨架 ===============
// 面包屑
.sk-breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;

  &__sep {
    width: 0;
    height: 0;
    border-left: 4px solid var(--el-border-color);
    border-top: 3px solid transparent;
    border-bottom: 3px solid transparent;
    opacity: 0.4;
  }
}

// 工具栏
.sk-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;

  &__left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__right {
    display: flex;
    align-items: center;
    gap: 6px;
  }
}

// 搜索栏
.sk-search {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 0 16px;
  padding: 16px 20px;
  margin-bottom: 14px;
  background: var(--el-bg-color);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
}

:global(html.dark) {
  .sk-search {
    background-color: #14171c;
    border-color: #23272f;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }

  .sk-table-card {
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.18);
  }
}

.sk-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-width: 160px;
  animation: sk-appear 0.5s ease both;

  &__label {
    width: 48px;
    height: 10px;
    border-radius: 2px;
    background: var(--el-fill-color-lighter);
    @include skeleton-pulse;
  }

  &__input {
    height: 32px;
    border-radius: 6px;
    @include skeleton-pulse;
    animation: sk-appear 0.5s ease both;
  }

  &__btns {
    display: flex;
    gap: 8px;
    align-items: center;
    padding-top: 16px;
  }
}

// 表格卡片
.sk-table-card {
  background: var(--el-bg-color);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
}

// 表头
.sk-thead {
  display: flex;
  align-items: center;
  padding: 12px 0;
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.sk-th {
  height: 12px;
  border-radius: 2px;
  background: var(--el-fill-color);
  margin: 0 12px;
  flex: 1;

  &--end {
    flex: 0 0 auto;
    width: 100px;
    margin-right: 16px;
  }
}

// 行
.sk-trow {
  display: flex;
  align-items: center;
  padding: 13px 0;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  animation: sk-appear 0.5s ease both;

  &:last-child {
    border-bottom: none;
  }
}

.sk-td {
  flex: 1;
  margin: 0 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;

  &--name {
    flex: 0 0 auto;
    width: 20%;
  }

  &--end {
    flex: 0 0 auto;
    width: 100px;
    margin-right: 16px;
    justify-content: flex-end;
  }
}

.sk-name-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

// 分页器
.sk-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 14px;
}

.sk-pagi-pages {
  display: flex;
  gap: 4px;
}

.sk-pagi-btn {
  width: 30px;
  height: 30px;
  border-radius: 4px;
  @include skeleton-pulse;

  &--active {
    background: var(--el-color-primary-light-7);
  }

  &--nav {
    width: 30px;
    height: 30px;
    border-radius: 4px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
  }
}

// =============== 过渡 ===============
.skeleton-fade-enter-active {
  transition: opacity 0.2s ease;
}

.skeleton-fade-leave-active {
  transition: opacity 0.12s ease;
}

.skeleton-fade-enter-from,
.skeleton-fade-leave-to {
  opacity: 0;
}
</style>
