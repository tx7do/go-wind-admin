<template>
  <el-dropdown ref="dropdownRef" class="notice__dropdown" trigger="click">
    <div class="notice__trigger">
      <el-badge v-if="unreadTotal > 0" :value="unreadTotal" :max="99">
        <SvgIcon icon="bell" />
      </el-badge>

      <SvgIcon v-else icon="bell" />
    </div>

    <template #dropdown>
      <div class="notice-dropdown">
        <!-- 头部：标题 + 未读数 + 全部已读 -->
        <div class="notice-dropdown__header">
          <div class="notice-dropdown__heading">
            <span>{{ t("core.notice.title") }}</span>
            <span v-if="unreadTotal > 0" class="notice-dropdown__unread">
              {{ t("core.notice.unreadCount", { count: unreadTotal }) }}
            </span>
          </div>
          <el-button
            text
            type="primary"
            size="small"
            :loading="markAllLoading"
            :disabled="unreadTotal <= 0"
            @click="readAll"
          >
            <el-icon class="mr-4px"><Check /></el-icon>
            {{ t("core.notice.markAllRead") }}
          </el-button>
        </div>

        <!-- 消息列表：最近 5 条已读+未读混合，未读带圆点加粗 -->
        <div v-loading="listLoading" class="notice-dropdown__body">
          <template v-if="list.length > 0">
            <div
              v-for="item in list"
              :key="item.id"
              class="notice-item"
              @click="goInbox"
            >
              <span
                class="notice-item__dot"
                :class="{ 'is-unread': isUnread(item) }"
              />

              <div class="notice-item__main">
                <div
                  class="notice-item__title"
                  :class="{ 'is-unread': isUnread(item) }"
                >
                  {{ item.title || "-" }}
                </div>
                <div v-if="stripHtml(item.content)" class="notice-item__excerpt">
                  {{ stripHtml(item.content) }}
                </div>
              </div>

              <span class="notice-item__time">{{ fromNow(item.createdAt) }}</span>
            </div>
          </template>

          <!-- 空状态 -->
          <el-empty
            v-else
            class="notice-dropdown__empty"
            :image-size="60"
            :description="t('core.notice.empty')"
          />
        </div>

        <!-- 底部：进入收件箱 -->
        <div class="notice-dropdown__footer">
          <el-button link type="primary" size="small" @click="goInbox">
            {{ t("core.notice.viewAll") }}
          </el-button>
        </div>
      </div>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Check } from "@element-plus/icons-vue";
import type { ElDropdown } from "element-plus";
// dayjs 语言包按需显式加载：不 import 时 locale("zh-cn") 静默无效，相对时间回落英文
import "dayjs/locale/zh-cn";
import "dayjs/plugin/relativeTime";
import SvgIcon from "@/components/SvgIcon/index.vue";
import { dateUtil } from "@/utils";
import { useNotice } from "./useNotice";
import type { internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient } from "@/api/generated/admin/service/v1";

const { t, locale } = useI18n();
const { list, unreadTotal, listLoading, markAllLoading, readAll, goMore } =
  useNotice();

const dropdownRef = ref<InstanceType<typeof ElDropdown>>();

function isUnread(item: InternalMessageRecipient): boolean {
  return item.status === "RECEIVED";
}

/**
 * 去除 HTML 标签取纯文本摘要（46 字符封顶）。
 * 结果经插值渲染为纯文本，无注入面。
 */
function stripHtml(html?: string): string {
  if (!html) return "";
  return html
    .replace(/<[^>]*>/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 46);
}

/** 相对时间（如"5 分钟前"），跟随界面语言 */
function fromNow(value?: string): string {
  if (!value) return "-";
  const d = dateUtil(value);
  if (!d.isValid()) return "-";
  return String(locale.value).startsWith("zh")
    ? d.locale("zh-cn").fromNow()
    : d.fromNow();
}

/** 跳转收件箱页，并收起浮层（自定义面板点击不会随路由跳转自动关闭） */
function goInbox() {
  dropdownRef.value?.handleClose();
  goMore();
}
</script>

<style lang="scss" scoped>
.notice {
  &__dropdown {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
  }

  &__trigger {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    cursor: pointer;

    // 顶部栏铃铛 hover 摆动（vben notification.vue 同款 bell-ring）；
    // svg 渲染在 SvgIcon 子组件内，跨组件作用域须 :deep()
    &:hover :deep(.svg-local-icon svg) {
      animation: bell-ring 1s both;
      transform-origin: top;
    }
  }
}

@keyframes bell-ring {
  0%,
  100% {
    transform: rotateZ(0deg);
  }

  15% {
    transform: rotateZ(10deg);
  }

  30% {
    transform: rotateZ(-10deg);
  }

  45% {
    transform: rotateZ(5deg);
  }

  60% {
    transform: rotateZ(-5deg);
  }

  75% {
    transform: rotateZ(2deg);
  }
}

// 下拉面板（对齐 react 端 notification-popover：336 宽 + 10/10/6 内边距）
.notice-dropdown {
  width: 336px;
  box-sizing: border-box;
  padding: 10px 10px 6px;
  background-color: var(--el-bg-color);
  border-radius: 8px;
  box-shadow: var(--el-box-shadow-light);

  // 头部
  &__header {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    padding: 2px 2px 10px;
    margin-bottom: 4px;
    border-bottom: 1px solid var(--el-border-color);
  }

  &__heading {
    display: flex;
    gap: 8px;
    align-items: center;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__unread {
    padding: 0 8px;
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
    border-radius: 10px;
  }

  // 消息列表
  &__body {
    min-height: 60px;
  }

  // 空状态
  &__empty {
    padding: 16px 0;
  }

  // 底部
  &__footer {
    padding-top: 8px;
    margin-top: 4px;
    text-align: center;
    border-top: 1px solid var(--el-border-color);
  }
}

// 消息项
.notice-item {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 9px 8px;
  cursor: pointer;
  border-radius: 8px;
  transition: background-color 0.2s ease;

  &:hover {
    background-color: var(--el-fill-color-light);
  }

  &__dot {
    width: 6px;
    height: 6px;
    margin-top: 6px;
    flex-shrink: 0;
    background: transparent;
    border-radius: 50%;

    &.is-unread {
      background-color: var(--el-color-primary);
    }
  }

  &__main {
    flex: 1;
    min-width: 0;
  }

  &__title {
    overflow: hidden;
    font-size: 13px;
    font-weight: 400;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;

    &.is-unread {
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  &__excerpt {
    margin-top: 2px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__time {
    flex-shrink: 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}
</style>
