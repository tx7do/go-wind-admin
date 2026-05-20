<template>
  <el-dropdown class="notice__dropdown" trigger="click">
    <div class="notice__trigger">
      <el-badge v-if="unreadTotal > 0" :value="unreadTotal" :max="99">
        <div class="i-svg:bell" />
      </el-badge>

      <div v-else class="i-svg:bell" />
    </div>

    <template #dropdown>
      <div class="notice-dropdown">
        <!-- 头部 -->
        <div class="notice-dropdown__header">
          <span class="notice-dropdown__title">{{ t("core.notice.title") }}</span>
          <el-icon class="notice-dropdown__icon" :size="20">
            <Bell />
          </el-icon>
        </div>

        <!-- 消息列表 -->
        <div class="notice-dropdown__body">
          <template v-if="list.length > 0">
            <div
              v-for="item in list"
              :key="item.id"
              class="notice-item"
              @click="read(item?.id || '')"
            >
              <!-- 头像 -->
              <div class="notice-item__avatar">
                <el-avatar :size="40" :src="defaultAvatar">
                  <el-icon :size="20"><UserFilled /></el-icon>
                </el-avatar>
              </div>

              <!-- 内容 -->
              <div class="notice-item__content">
                <div class="notice-item__header">
                  <h4 class="notice-item__title">{{ item.title }}</h4>
                </div>

                <!-- 摘要内容（如果有） -->
                <p v-if="item.content" class="notice-item__excerpt">
                  {{ stripHtml(item.content) }}
                </p>

                <!-- 时间 -->
                <div class="notice-item__footer">
                  <span class="notice-item__time">{{ item.createdAt }}</span>
                </div>
              </div>
            </div>
          </template>

          <!-- 空状态 -->
          <template v-else>
            <div class="notice-dropdown__empty">
              <el-empty :image-size="80" :description="t('core.notice.empty')" />
            </div>
          </template>
        </div>

        <!-- 底部操作栏 -->
        <div class="notice-dropdown__footer">
          <div class="notice-dropdown__footer-left">
            <el-button v-if="list.length > 0" text type="primary" size="small" @click="readAll">
              {{ t("core.notice.markAllRead") }}
            </el-button>
            <el-button v-if="list.length > 0" text type="danger" size="small" @click="clearAll">
              {{ t("core.notice.clearAll") }}
            </el-button>
          </div>
          <el-button type="primary" size="small" @click="goMore">
            {{ t("core.notice.viewAll") }}
          </el-button>
        </div>
      </div>
    </template>
  </el-dropdown>

  <el-dialog
    v-model="dialogVisible"
    :title="detail?.title ?? '通知详情'"
    width="800px"
    custom-class="notification-detail"
  >
    <div v-if="detail" class="p-x-20px">
      <div class="flex-y-center mb-16px text-13px text-color-secondary">
        <span class="flex-y-center">
          <el-icon><User /></el-icon>
          {{ detail.publisherName }}
        </span>
        <span class="ml-2 flex-y-center">
          <el-icon><Timer /></el-icon>
          {{ detail.publishTime }}
        </span>
      </div>

      <div class="max-h-60vh pt-16px mb-24px overflow-y-auto border-t border-solid border-color">
        <div v-html="detail.content"></div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { Bell, UserFilled } from "@element-plus/icons-vue";
import defaultAvatar from "@/assets/images/default-avatar.png";
import { useNotice } from "./useNotice";

const { t } = useI18n();
const { list, unreadTotal, detail, dialogVisible, read, readAll, clearAll, goMore } = useNotice();

/**
 * 去除 HTML 标签，获取纯文本摘要
 */
function stripHtml(html: string): string {
  const tmp = document.createElement("div");
  tmp.innerHTML = html;
  const text = tmp.textContent || tmp.innerText || "";
  // 限制长度为 50 个字符
  return text.length > 50 ? text.substring(0, 50) + "..." : text;
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
  }
}

// 下拉面板样式
.notice-dropdown {
  width: 420px;
  max-height: 600px;
  display: flex;
  flex-direction: column;
  background-color: var(--el-bg-color);
  border-radius: 8px;
  box-shadow: var(--el-box-shadow-light);

  // 头部
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    .notice-dropdown__title {
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .notice-dropdown__icon {
      color: var(--el-text-color-secondary);
      cursor: pointer;

      &:hover {
        color: var(--el-color-primary);
      }
    }
  }

  // 消息列表
  &__body {
    flex: 1;
    overflow-y: auto;
    max-height: 480px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background-color: var(--el-border-color);
      border-radius: 3px;

      &:hover {
        background-color: var(--el-border-color-darker);
      }
    }
  }

  // 空状态
  &__empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 300px;
  }

  // 底部操作栏
  &__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 20px;
    border-top: 1px solid var(--el-border-color-lighter);
    background-color: var(--el-fill-color-lighter);

    &-left {
      display: flex;
      align-items: center;
      gap: 8px;
    }
  }
}

// 消息项样式
.notice-item {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  cursor: pointer;
  transition: background-color 0.2s ease;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:hover {
    background-color: var(--el-fill-color-light);
  }

  &:last-child {
    border-bottom: none;
  }

  // 头像
  &__avatar {
    flex-shrink: 0;

    :deep(.el-avatar) {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    }
  }

  // 内容区域
  &__content {
    flex: 1;
    min-width: 0;
  }

  &__header {
    margin-bottom: 4px;
  }

  &__title {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  // 摘要内容
  &__excerpt {
    margin: 6px 0;
    font-size: 13px;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  // 底部时间
  &__footer {
    margin-top: 8px;
  }

  &__time {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
  }
}
</style>
