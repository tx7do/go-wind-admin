<template>
  <div class="user-detail-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <ElButton type="text" @click="goBack">
          <template #icon>
            <ArrowLeft />
          </template>
        </ElButton>
        <span class="page-title">{{ $t("pages.user.detail.title", { userId }) }}</span>
      </div>
      <div class="header-right">
        <ElButton type="danger" @click="handleBanAccount">
          {{ $t("pages.user.button.banAccount") }}
        </ElButton>
        <ElButton type="primary" @click="handleEditPassword">
          {{ $t("pages.user.button.editPassword") }}
        </ElButton>
      </div>
    </div>

    <!-- 标签页 -->
    <div class="page-content">
      <ElTabs v-model="activeTab" class="detail-tabs">
        <ElTabPane :label="$t('pages.user.detail.tab.basicInfo')" :name="TabEnum.BASIC_INFO" />
        <ElTabPane :label="$t('pages.user.detail.tab.apiAuditLog')" :name="TabEnum.API_AUDIT_LOG" />
        <ElTabPane
          :label="$t('pages.user.detail.tab.internalMessage')"
          :name="TabEnum.INTERNAL_MESSAGE"
        />
      </ElTabs>

      <!-- 标签页内容 -->
      <ElCard v-show="activeTab === TabEnum.BASIC_INFO" class="tab-content">
        <BasicInfoPage :user-id="userId" />
      </ElCard>
      <ElCard v-show="activeTab === TabEnum.API_AUDIT_LOG" class="tab-content">
        <ApiLogPage :user-id="userId" />
      </ElCard>
      <ElCard v-show="activeTab === TabEnum.INTERNAL_MESSAGE" class="tab-content">
        <InternalMessagePage :user-id="userId" />
      </ElCard>
    </div>

    <!-- 编辑密码弹窗 -->
    <EditPasswordModal
      v-model="dialogVisible"
      :data="dialogData"
      @success="handlePasswordSuccess"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox, ElButton, ElTabs, ElTabPane, ElCard } from "element-plus";
import { ArrowLeft } from "@element-plus/icons-vue";
import { $t } from "@/i18n";

import { useUpdateUser } from "@/api/composables";
import { TabEnum } from "@/views/app/opm/user/detail/types";

import ApiLogPage from "./api-log-page.vue";
import BasicInfoPage from "./basic-info-page.vue";
import EditPasswordModal from "./components/edit-password-modal.vue";
import InternalMessagePage from "./internal-message-page.vue";

const activeTab = ref<TabEnum>(TabEnum.BASIC_INFO);
const route = useRoute();
const router = useRouter();

const userId = computed(() => {
  const id = route.params.id ?? -1;
  return Number(id);
});

const { mutateAsync: updateUser } = useUpdateUser();

// 弹窗控制
const dialogVisible = ref(false);
const dialogData = ref({ create: false, userId: undefined });

/* 打开模态窗口 */
function openModal(create: boolean, userId?: any) {
  dialogData.value = { create, userId };
  dialogVisible.value = true;
}

/**
 * 返回上一级页面
 */
function goBack() {
  router.push("/opm/users");
}

/**
 * 禁用账户
 */
async function handleBanAccount() {
  try {
    await ElMessageBox.confirm(
      $t("common.text.do_you_want_disable", {
        moduleName: $t("pages.user.moduleName"),
      }),
      $t("common.dialog.confirm"),
      {
        confirmButtonText: $t("common.button.ok"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    );

    await updateUser({ id: userId.value, values: { status: "DISABLED" } });
    ElMessage.success($t("common.notification.update_status_success"));
  } catch {
    // 用户取消
  }
}

/**
 * 编辑密码
 */
function handleEditPassword() {
  openModal(true, userId);
}

/**
 * 密码修改成功
 */
function handlePasswordSuccess() {
  dialogVisible.value = false;
  ElMessage.success($t("common.notification.update_success"));
}
</script>

<style lang="scss" scoped>
.user-detail-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: var(--el-bg-color);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--el-border-color);

  .header-left {
    display: flex;
    align-items: center;
    gap: 10px;

    .page-title {
      font-size: 16px;
      font-weight: 500;
    }
  }

  .header-right {
    display: flex;
    gap: 8px;
  }
}

.page-content {
  flex: 1;
  padding: 16px 20px;
  overflow: auto;

  .detail-tabs {
    margin-bottom: 16px;
  }

  .tab-content {
    // 内容区样式
  }
}
</style>
