<template>
  <div class="basic-info-page">
    <!-- 基本信息卡片 -->
    <div class="basic-info-container">
      <!-- 头像与状态 -->
      <div class="avatar-section">
        <ElAvatar
          :size="140"
          :src="data?.avatar ?? ''"
          class="avatar"
          :style="!data?.avatar ? { backgroundColor: getAvatarColor() } : {}"
        >
          <!-- 头像加载失败/无头像时显示姓名首字母 -->
          <span class="avatar-placeholder">
            {{ data?.username?.substring(0, 1) || "?" }}
          </span>
        </ElAvatar>
      </div>

      <!-- 详细信息列表 -->
      <ElDescriptions class="info-list" :column="2" border>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.username')">
          {{ data?.username }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.realname')">
          {{ data?.realname }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.nickname')">
          {{ data?.nickname }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.gender')">
          <ElTag :color="genderToColor(data?.gender)" effect="dark" round>
            {{ genderToName(data?.gender) }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.roleNames')">
          <ElTag
            v-for="role in data?.roleNames"
            :key="role"
            class="tag-item"
            :style="{
              backgroundColor: getRandomColor(role),
              color: '#333',
              border: 'none',
            }"
          >
            {{ role }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.mobile')">
          {{ data?.mobile }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.email')">
          {{ data?.email }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.region')">
          {{ data?.region }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.address')">
          {{ data?.address }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.tenantName')">
          {{ data?.tenantName }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.orgUnitName')">
          <ElTag
            v-for="orgUnit in data?.orgUnitNames"
            :key="orgUnit"
            class="tag-item"
            :style="{
              backgroundColor: getRandomColor(orgUnit),
              color: '#333',
              border: 'none',
            }"
          >
            {{ orgUnit }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.positionName')">
          <ElTag
            v-for="position in data?.positionNames"
            :key="position"
            class="tag-item"
            :style="{
              backgroundColor: getRandomColor(position),
              color: '#333',
              border: 'none',
            }"
          >
            {{ position }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('common.table.createdAt')">
          {{ formatDateTime(data?.createdAt ?? "") }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.lastLoginAt')">
          {{ data?.lastLoginAt }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.user.detail.desc.lastLoginIp')">
          {{ data?.lastLoginIp }}
        </ElDescriptionsItem>
      </ElDescriptions>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";

import { ElDescriptions, ElDescriptionsItem, ElAvatar, ElTag } from "element-plus";
import { formatDateTime } from "@/utils";
import { $t } from "@/i18n";

import { type identityservicev1_User as User } from "@/api/generated/admin/service/v1";
import { genderToColor, genderToName, fetchUser } from "@/api/composables";
import { getCharColor, getRandomColor } from "@/utils/color";

const props = defineProps({
  userId: { type: Number, default: undefined },
});

const data = ref<User>();

// 获取首字母（默认用'?'）
const getFirstChar = computed(() => {
  if (!data.value?.username) return "?";
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
    data.value = await fetchUser({ id: props.userId });
  }
}

reload();
</script>

<style scoped>
.basic-info-page {
  width: 100%;
  height: 100%;
}

.basic-info-container {
  display: flex;
  gap: 32px;
  padding: 24px;
  flex-wrap: wrap;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

/* 首字母占位样式 */
.avatar-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 85px;
  font-weight: 700;
  color: #fff;
  line-height: 1;
  text-transform: uppercase;
}

.info-list {
  flex: 1;
  min-width: 400px;
}

.tag-item {
  margin-bottom: 4px;
  margin-right: 4px;
}
</style>
