<template>
  <div class="page-container">
    <div class="account-list">
      <div v-for="item in bindList" :key="item.key" class="account-item">
        <div class="account-item-content">
          <!-- 左侧：图标和标题 -->
          <div class="item-left">
            <IconifyIcon
              :icon="item.avatar"
              :width="28"
              :height="28"
              color="var(--el-text-color-regular)"
              class="item-avatar"
            />
            <div class="item-info">
              <span class="item-title">{{ item.title }}</span>
              <span class="item-description">{{ item.description }}</span>
            </div>
          </div>
          <!-- 右侧：操作 -->
          <div class="item-actions">
            <template v-if="item.key === 'email'">
              <ElLink type="primary" underline="never" class="item-link" @click="openBind">
                {{ user?.email ? $t("pages.user.accountBind.rebind") : $t("pages.user.accountBind.bindNow") }}
              </ElLink>
            </template>
            <ElLink type="primary" underline="never" class="item-link" @click="emit('switchTab', '1')">
              {{ $t("pages.user.accountBind.modify") }}
            </ElLink>
          </div>
        </div>
      </div>
    </div>

    <!-- 绑定邮箱抽屉：发送验证码 → 输入验证码完成绑定 -->
    <ElDrawer
      v-model="bindOpen"
      :title="$t('pages.user.accountBind.bindEmailTitle')"
      size="480px"
      :append-to-body="true"
      :destroy-on-close="true"
    >
      <ElForm label-position="top">
        <ElFormItem :label="$t('pages.user.accountBind.newEmail')" required>
          <ElInput v-model="bindEmail" :placeholder="$t('pages.user.accountBind.newEmailPlaceholder')" />
        </ElFormItem>
        <ElFormItem :label="$t('pages.user.accountBind.vcode')" required>
          <div class="vcode-row">
            <ElInput v-model="bindCode" :placeholder="$t('pages.user.accountBind.vcodePlaceholder')" />
            <ElButton :disabled="codeSent" :loading="sending" @click="handleSendCode">
              {{ codeSent ? $t("pages.user.accountBind.codeSent") : $t("pages.user.accountBind.sendVcode") }}
            </ElButton>
          </div>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="bindOpen = false">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="verifying" @click="handleVerify">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </template>
    </ElDrawer>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { Icon as IconifyIcon } from "@iconify/vue";
import { ElMessage } from "element-plus";

import { useGetUserProfile } from "@/api/composables";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { $t } from "@go-wind/admin-element";

const emit = defineEmits<{ switchTab: [key: string] }>();

const { data: user } = useGetUserProfile();

const maskEmail = (email: string) => {
  const at = email.lastIndexOf("@");
  if (at <= 0) return email;
  return `${email.slice(0, Math.min(2, at))}***${email.slice(at)}`;
};

const maskPhone = (phone: string) => {
  if (phone.length < 7) return phone;
  return `${phone.slice(0, 3)}****${phone.slice(-4)}`;
};

interface BindItem {
  key: string;
  title: string;
  description: string;
  avatar: string;
}

const bindList = computed<BindItem[]>(() => {
  const email = user.value?.email ?? "";
  const mobile = user.value?.mobile ?? "";

  return [
    {
      key: "email",
      title: $t("pages.user.accountBind.email"),
      description: email
        ? $t("pages.user.accountBind.bound", { value: maskEmail(email) })
        : $t("pages.user.accountBind.unbound"),
      avatar: "ri:mail-fill",
    },
    {
      key: "phone",
      title: $t("pages.user.accountBind.phone"),
      description: mobile
        ? $t("pages.user.accountBind.bound", { value: maskPhone(mobile) })
        : $t("pages.user.accountBind.unbound"),
      avatar: "ri:smartphone-fill",
    },
  ];
});

// ===== 绑定邮箱（发送验证码 → 校验并写入 EMAIL 凭证）=====
const bindOpen = ref(false);
const bindEmail = ref("");
const bindCode = ref("");
const codeSent = ref(false);
const sending = ref(false);
const verifying = ref(false);

function openBind() {
  bindEmail.value = user.value?.email ?? "";
  bindCode.value = "";
  codeSent.value = false;
  bindOpen.value = true;
}

async function handleSendCode() {
  const email = bindEmail.value.trim();
  if (!email) {
    ElMessage.error($t("pages.user.accountBind.newEmailRequired"));
    return;
  }
  sending.value = true;
  try {
    await apiClient.userProfileService.BindContact({ email: { email } });
    codeSent.value = true;
    ElMessage.success($t("pages.user.accountBind.codeSent"));
  } catch (error: any) {
    console.error("Failed to send bind code:", error);
    ElMessage.error(error?.message || $t("pages.user.accountBind.vcodeSendFailed"));
  } finally {
    sending.value = false;
  }
}

async function handleVerify() {
  const email = bindEmail.value.trim();
  if (!email || !bindCode.value) {
    ElMessage.error($t("pages.user.accountBind.vcodeRequired"));
    return;
  }
  verifying.value = true;
  try {
    await apiClient.userProfileService.VerifyContact({
      email: { email, code: bindCode.value },
    });
    ElMessage.success($t("pages.user.accountBind.bindSuccess"));
    bindOpen.value = false;
    // 刷新用户信息以更新绑定状态展示（本页读 userProfile，顶栏头像等读 getMe）
    queryClient.invalidateQueries({ queryKey: ["getMe"] });
    queryClient.invalidateQueries({ queryKey: ["userProfile"] });
  } catch (error: any) {
    console.error("Failed to verify bind code:", error);
    ElMessage.error(error?.message || $t("pages.user.accountBind.bindFailed"));
  } finally {
    verifying.value = false;
  }
}
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 800px;
}

.account-list {
  padding-top: 20px;
}

.account-item {
  padding: 16px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }
}

.account-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.item-left {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.item-avatar {
  margin-right: 16px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.item-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.item-description {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.item-actions {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-shrink: 0;
}

.item-link {
  flex-shrink: 0;
  font-size: 14px;
}

.vcode-row {
  display: flex;
  gap: 8px;
  width: 100%;
}
</style>
