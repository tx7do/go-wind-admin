<script setup lang="ts">
import { computed, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { formatDateTime } from '@vben/utils';

import { Avatar, Descriptions, DescriptionsItem } from 'ant-design-vue';

import { type userservicev1_User as User } from '#/generated/api/admin/service/v1';
import { genderToColor, genderToName, useUserListStore } from '#/stores';
import { getCharColor, getRandomColor } from '#/utils/color';

const props = defineProps({
  userId: { type: Number, default: undefined },
});

const userListStore = useUserListStore();

const data = ref<User>();

// 获取首字母（默认用'?'）
const getFirstChar = computed(() => {
  if (!data.value?.username) return '?';
  return data.value.username.slice(0, 1).toUpperCase();
});

// 根据首字母生成固定随机色
const getAvatarColor = () => {
  return getCharColor(getFirstChar.value);
};

/**
 * 重新加载用户信息
 */
async function reload() {
  if (props.userId) {
    data.value = await userListStore.getUser(props.userId);
  }
}

reload();
</script>

<template>
  <Page>
    <!-- 基本信息卡片 -->
    <a-card class="mt-4">
      <div class="basic-info-container">
        <!-- 头像与状态 -->
        <div class="avatar-section">
          <Avatar
            :src="data?.avatar ?? ''"
            class="avatar"
            :style="!data?.avatar ? { backgroundColor: getAvatarColor() } : {}"
          >
            <!-- 头像加载失败/无头像时显示姓名首字母，添加占位样式 -->
            <span class="avatar-placeholder">
              {{ data?.username?.substring(0, 1) || '?' }}
            </span>
          </Avatar>
        </div>

        <!-- 详细信息列表 -->
        <Descriptions class="info-list">
          <DescriptionsItem :label="$t('page.user.detail.desc.username')">
            {{ data?.username }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.realname')">
            {{ data?.realname }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.nickname')">
            {{ data?.nickname }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.gender')">
            <a-tag :color="genderToColor(data?.gender)">
              {{ genderToName(data?.gender) }}
            </a-tag>
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.roleNames')">
            <a-tag
              v-for="role in data?.roleNames"
              :key="role"
              class="mb-1 mr-1"
              :style="{
                backgroundColor: getRandomColor(role), // 随机背景色
                color: '#333', // 深色文字（适配浅色背景）
                border: 'none', // 可选：去掉边框更美观
              }"
            >
              {{ role }}
            </a-tag>
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.mobile')">
            {{ data?.mobile }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.email')">
            {{ data?.email }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.region')">
            {{ data?.region }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.address')">
            {{ data?.address }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.tenantName')">
            {{ data?.tenantName }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.orgUnitName')">
            <a-tag
              v-for="orgUnit in data?.orgUnitNames"
              :key="orgUnit"
              class="mb-1 mr-1"
              :style="{
                backgroundColor: getRandomColor(orgUnit), // 随机背景色
                color: '#333', // 深色文字（适配浅色背景）
                border: 'none', // 可选：去掉边框更美观
              }"
            >
              {{ orgUnit }}
            </a-tag>
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.positionName')">
            <a-tag
              v-for="position in data?.positionNames"
              :key="position"
              class="mb-1 mr-1"
              :style="{
                backgroundColor: getRandomColor(position), // 随机背景色
                color: '#333', // 深色文字（适配浅色背景）
                border: 'none', // 可选：去掉边框更美观
              }"
            >
              {{ position }}
            </a-tag>
          </DescriptionsItem>
          <DescriptionsItem :label="$t('ui.table.createdAt')">
            {{ formatDateTime(data?.createdAt ?? '') }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.lastLoginAt')">
            {{ data?.lastLoginAt }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.user.detail.desc.lastLoginIp')">
            {{ data?.lastLoginIp }}
          </DescriptionsItem>
        </Descriptions>
      </div>
    </a-card>
  </Page>
</template>

<style scoped>
.basic-info-container {
  display: flex;
  gap: 32px; /* 头像与信息的间距 */
  padding: 24px;
  flex-wrap: wrap; /* 小屏幕自动换行 */
}

.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.avatar {
  width: 140px;
  height: 140px;
  border-radius: 50%; /* 确保是正圆形（部分组件可能默认非圆形） */
  display: inline-flex;
  align-items: center;
  justify-content: center;
  overflow: hidden; /* 防止头像或文字溢出圆形 */
}

/* 首字母占位样式：占满容器并居中 */
.avatar-placeholder {
  /* 充满整个头像容器 */
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 85px;
  font-weight: 700;
  color: #fff;
  line-height: 1; /* 消除行高带来的垂直偏移 */
  text-transform: uppercase; /* 统一转为大写，视觉更规整 */
}

.status-badge {
  padding: 4px 12px;
  font-size: 14px;
}

.info-list {
  flex: 1;
  min-width: 400px; /* 确保小屏幕不挤压 */
}

/* 描述项样式优化 */
:deep(.ant-descriptions-item) {
  padding: 12px 0;
}

:deep(.ant-descriptions-item-label) {
  font-weight: 500;
  width: 120px;
}

.mt-4 {
  margin-top: 16px;
}
</style>
