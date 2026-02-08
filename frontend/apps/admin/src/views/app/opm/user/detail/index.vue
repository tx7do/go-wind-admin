<script lang="ts" setup>
import { computed, ref } from 'vue';
import { useRoute } from 'vue-router';

import { Page, useVbenModal } from '@vben/common-ui';
import { LucideArrowLeft } from '@vben/icons';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { router } from '#/router';
import { useUserListStore } from '#/stores';
import { TabEnum } from '#/views/app/opm/user/detail/types';

import ApiLogPage from './api-log-page.vue';
import BasicInfoPage from './basic-info-page.vue';
import EditPasswordModal from './components/edit-password-modal.vue';
import InternalMessagePage from './internal-message-page.vue';

const activeTab = ref<TabEnum>(TabEnum.BASIC_INFO);

const route = useRoute();

const userId = computed(() => {
  const id = route.params.id ?? -1;
  return Number(id);
});

const userListStore = useUserListStore();

const [Modal, modalApi] = useVbenModal({
  // 连接抽离的组件
  connectedComponent: EditPasswordModal,
});

/* 打开模态窗口 */
function openModal(create: boolean, userId?: any) {
  modalApi.setData({
    create,
    userId,
  });

  modalApi.open();
}

/**
 * 返回上一级页面
 */
function goBack() {
  router.push('/opm/users');
}

/**
 * 禁用账户
 */
async function handleBanAccount() {
  try {
    await userListStore.updateUser(userId.value, { status: 'DISABLED' });

    notification.success({
      message: $t('ui.notification.update_status_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.update_status_failed'),
    });
  }
}

/**
 * 编辑密码
 */
function handleEditPassword() {
  openModal(true, userId);
}
</script>

<template>
  <Page content-class="flex flex-col gap-4">
    <template #title>
      <div
        style="
          display: flex;
          justify-content: flex-start;
          align-items: center;
          gap: 10px;
        "
      >
        <a-button type="text" @click="goBack">
          <template #icon>
            <LucideArrowLeft class="text-align:center" />
          </template>
        </a-button>
        <span>{{ $t('page.user.detail.title', { userId }) }}</span>
      </div>
    </template>
    <template #extra>
      <a-popconfirm
        :cancel-text="$t('ui.button.cancel')"
        :ok-text="$t('ui.button.ok')"
        :title="
          $t('ui.text.do_you_want_disable', {
            moduleName: $t('page.user.moduleName'),
          })
        "
        @confirm="handleBanAccount"
      >
        <a-button class="mr-2" danger type="primary">
          {{ $t('page.user.button.banAccount') }}
        </a-button>
      </a-popconfirm>
      <a-button class="mr-2" type="primary" @click="handleEditPassword">
        {{ $t('page.user.button.editPassword') }}
      </a-button>
    </template>
    <template #description>
      <a-tabs
        v-model:active-key="activeTab"
        :tab-bar-style="{ marginBottom: 0 }"
      >
        <a-tab-pane
          :key="TabEnum.BASIC_INFO"
          :tab="$t('page.user.detail.tab.basicInfo')"
        />
        <a-tab-pane
          :key="TabEnum.API_AUDIT_LOG"
          :tab="$t('page.user.detail.tab.apiAuditLog')"
        />
        <a-tab-pane
          :key="TabEnum.INTERNAL_MESSAGE"
          :tab="$t('page.user.detail.tab.internalMessage')"
        />
      </a-tabs>
    </template>

    <a-card v-show="activeTab === TabEnum.BASIC_INFO">
      <BasicInfoPage :user-id="userId" />
    </a-card>
    <a-card v-show="activeTab === TabEnum.API_AUDIT_LOG">
      <ApiLogPage :user-id="userId" />
    </a-card>
    <a-card v-show="activeTab === TabEnum.INTERNAL_MESSAGE">
      <InternalMessagePage :user-id="userId" />
    </a-card>

    <Modal />
  </Page>
</template>

<style></style>
