<template>
  <div>
    <div
      class="command-palette-trigger"
      role="button"
      tabindex="0"
      :aria-label="$t('common.commandPalette.openSearch')"
      @click="open"
      @keydown.enter.prevent="open"
      @keydown.space.prevent="open"
    >
      <div class="command-palette-trigger__left">
        <SvgIcon icon="search" />
        <span class="command-palette-trigger__text">
          {{ $t("common.commandPalette.searchMenu") }}
        </span>
      </div>
      <kbd class="command-palette-trigger__kbd">Ctrl K</kbd>
    </div>

    <el-dialog
      v-model="visible"
      width="600px"
      :close-on-click-modal="true"
      :show-close="false"
      @close="close"
    >
      <div class="command-palette-dialog">
        <!-- 头部：搜索图标 + 无边框输入（同一 baseline，规格见 docs/design-language.md「全局搜索面板」） -->
        <div class="command-palette-inputbar">
          <SvgIcon icon="search" :size="16" class="command-palette-inputbar__icon" />
          <el-input
            ref="inputRef"
            v-model="keyword"
            class="command-palette-input"
            :placeholder="$t('common.commandPalette.searchMenu')"
            @input="onSearch"
            @keydown="handleInputKeydown"
          />
        </div>

        <!-- 结果区（限高内滚） -->
        <div class="command-palette-results">
          <!-- 无关键词：最近搜索 -->
          <template v-if="!keyword.trim()">
            <div
              v-if="history.length === 0"
              class="command-palette-empty"
            >
              {{ $t("common.commandPalette.noHistory") }}
            </div>
            <template v-else>
              <div class="command-palette-section">
                {{ $t("common.commandPalette.recent") }}
              </div>
              <ul class="command-palette-list">
                <li
                  v-for="(item, idx) in history"
                  :key="item.path"
                  :class="['command-palette-item', { 'is-active': activeIndex === idx }]"
                  @mouseenter="activeIndex = idx"
                  @click="onGo(item)"
                >
                  <SvgIcon
                    v-if="item.icon"
                    :icon="item.icon"
                    :size="15"
                    class="command-palette-item__icon"
                  />
                  <div class="command-palette-item__title">{{ item.title }}</div>
                  <SvgIcon
                    icon="close"
                    class="command-palette-item__remove"
                    @click.stop="removeHistory(idx)"
                  />
                </li>
              </ul>
            </template>
          </template>

          <!-- 有关键词：本地菜单命中 + 语义搜索独立小节 -->
          <template v-else>
            <div
              v-if="results.length === 0 && !semanticLoading && semanticResults.length === 0"
              class="command-palette-empty"
            >
              {{ $t("common.commandPalette.noResults") }}
            </div>

            <ul v-if="results.length" class="command-palette-list">
              <li
                v-for="(item, idx) in results"
                :key="item.path"
                :class="['command-palette-item', { 'is-active': activeIndex === idx }]"
                @mouseenter="activeIndex = idx"
                @click="onGo(item)"
              >
                <SvgIcon
                  v-if="item.icon"
                  :icon="item.icon"
                  :size="15"
                  class="command-palette-item__icon"
                />
                <div class="command-palette-item__title">{{ item.title }}</div>
              </li>
            </ul>

            <template v-if="semanticLoading || semanticResults.length">
              <div class="command-palette-section">
                {{ $t("common.commandPalette.semanticTitle") }}
              </div>
              <div
                v-if="semanticLoading"
                class="command-palette-empty command-palette-empty--tight"
              >
                {{ $t("common.commandPalette.searching") }}
              </div>
              <ul v-else class="command-palette-list">
                <li
                  v-for="(item, j) in semanticResults"
                  :key="item.path"
                  :class="['command-palette-item', { 'is-active': activeIndex === results.length + j }]"
                  @mouseenter="activeIndex = results.length + j"
                  @click="onGo(item)"
                >
                  <div class="command-palette-item__title">{{ item.title }}</div>
                  <div class="command-palette-item__route">{{ item.path }}</div>
                </li>
              </ul>
            </template>
          </template>
        </div>

        <div class="command-palette-hints">
          <div class="command-palette-hint">
            <div class="command-palette-hint__key"><SvgIcon icon="up" /></div>
            <div class="command-palette-hint__key"><SvgIcon icon="down" /></div>
            <span class="command-palette-hint__text">{{ $t("common.commandPalette.switch") }}</span>
          </div>
          <div class="command-palette-hint">
            <div class="command-palette-hint__key"><SvgIcon icon="enter" /></div>
            <span class="command-palette-hint__text">{{ $t("common.commandPalette.select") }}</span>
          </div>
          <div class="command-palette-hint">
            <div class="command-palette-hint__key"><SvgIcon icon="esc" /></div>
            <span class="command-palette-hint__text">{{ $t("common.commandPalette.close") }}</span>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import SvgIcon from "@/components/SvgIcon/index.vue";
import { useCommandPalette } from "./useCommandPalette";

const {
  visible,
  keyword,
  results,
  semanticResults,
  semanticLoading,
  history,
  activeIndex,
  inputRef,
  open,
  close,
  onSearch,
  onSelect,
  onNavigate,
  onGo,
  removeHistory,
} = useCommandPalette();

const handleInputKeydown: (evt: KeyboardEvent | Event) => any = (evt) => {
  if (!(evt instanceof KeyboardEvent)) return;
  const e = evt;
  const key = e.key.toLowerCase();

  if (key === "escape") {
    e.preventDefault();
    close();
    return;
  }

  if (key === "arrowup") {
    e.preventDefault();
    onNavigate("up");
    return;
  }

  if (key === "arrowdown") {
    e.preventDefault();
    onNavigate("down");
    return;
  }

  if (key === "enter") {
    e.preventDefault();
    if (activeIndex.value < 0) activeIndex.value = 0;
    onSelect();
  }
};
</script>

<style scoped>
.command-palette-trigger {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  height: 32px;
  padding: 0 12px;
  user-select: none;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 999px;
}

.command-palette-trigger__left {
  display: flex;
  gap: 8px;
  align-items: center;
}

.command-palette-trigger__left :deep(.svg-local-icon) {
  color: var(--el-text-color-secondary) !important;
}

.command-palette-trigger__text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.command-palette-trigger__kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px 8px;
  font-size: 12px;
  line-height: 1;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.command-palette-trigger:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}

.command-palette-trigger:hover {
  border-color: var(--el-border-color);
  background: var(--el-fill-color-lighter);
}

/* 窄视口收成纯图标（2026-11-08）：胶囊全宽 ~166px 是头部最贵的大件，挤出空间时
   面包屑被压到「图标+单字」药丸（用户截图的「概览📊 分」）。文本+kbd 隐藏后胶囊
   ~40px，面包屑拿回自然宽度（三项全显）。aria-label 已有，可发现性不受影响。
   头部横跨全视口，视口 MQ ≈ 头部宽度（侧栏展开时会提前触发，属可接受近似）。 */
@media (max-width: 1000px) {
  .command-palette-trigger {
    padding: 0 9px;
    gap: 0;
  }

  .command-palette-trigger__text,
  .command-palette-trigger__kbd {
    display: none;
  }
}

.command-palette-dialog {
  display: flex;
  flex-direction: column;
}

.command-palette-inputbar {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 6px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.command-palette-inputbar__icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
}

/* el-dialog 传送门外/内 scope 属性继承不稳（el-input 组件根元素可能丢父 scope id），
   锚在自家纯 div 上穿透，保证命中；特异性压到 (0,5,0) 才能盖过
   _dark-mode.scss 的 html.dark ... !important 焦点环（(0,4,1)） */
.command-palette-dialog :deep(.command-palette-input.el-input .el-input__wrapper) {
  background: transparent !important;
  border-radius: 0;
  box-shadow: none !important;
  padding-left: 0;
}

.command-palette-results {
  max-height: 48vh;
  overflow: auto;
  padding: 8px 0;
}

.command-palette-section {
  padding: 6px 16px 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.command-palette-empty {
  padding: 24px 0;
  color: var(--el-text-color-secondary);
  text-align: center;
}

.command-palette-empty--tight {
  padding: 12px 0;
}

.command-palette-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 0 8px;
  margin: 0;
  list-style: none;
}

.command-palette-item {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 9px 10px;
  cursor: pointer;
  border-radius: 6px;
}

.command-palette-item:hover {
  background: var(--el-fill-color-light);
}

/* 键盘/悬停选中 = 主色实底 + 白字（同侧栏菜单选中惯例，禁淡底） */
.command-palette-item.is-active {
  background: var(--el-color-primary);
}

.command-palette-item__icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
}

.command-palette-item__title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  font-size: 14px;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-palette-item__route {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.command-palette-item__remove {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
}

.command-palette-item__remove:hover {
  color: var(--gowind-primary-text);
}

.command-palette-item.is-active .command-palette-item__title {
  color: var(--el-color-white);
}

.command-palette-item.is-active .command-palette-item__icon {
  color: var(--el-color-white);
}

.command-palette-item.is-active .command-palette-item__route {
  color: color-mix(in srgb, var(--el-color-white) 80%, transparent);
}

.command-palette-item.is-active .command-palette-item__remove {
  color: var(--el-color-white);
}

.command-palette-hints {
  display: flex;
  gap: 14px;
  align-items: center;
  padding: 10px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.command-palette-hint {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.command-palette-hint__key {
  display: inline-flex;
  gap: 3px;
  align-items: center;
  justify-content: center;
  height: 24px;
  padding: 0 8px;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.command-palette-hint__key :deep(.svg-local-icon) {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.command-palette-hint__text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
