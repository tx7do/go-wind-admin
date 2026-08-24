<template>
  <div class="mfa-setting-item">
    <div class="setting-item-content">
      <span class="item-title">{{ $t("pages.user.secureSetting.mfaDevice") }}</span>
      <ElButton
        v-if="!hasTotp"
        type="primary"
        size="small"
        :loading="startEnroll.isPending.value"
        @click="handleStartEnroll"
      >
        {{ $t("pages.user.secureSetting.mfaStartBind") }}
      </ElButton>
      <ElButton
        v-else
        type="danger"
        size="small"
        plain
        :loading="disableMfa.isPending.value"
        @click="handleUnbind"
      >
        {{ $t("pages.user.secureSetting.mfaUnbind") }}
      </ElButton>
    </div>
    <div class="item-description">
      <template v-if="hasTotp">{{ $t("pages.user.secureSetting.mfaBoundDesc") }}</template>
      <template v-else>{{ $t("pages.user.secureSetting.mfaDeviceDesc") }}</template>
    </div>

    <!-- 绑定弹窗：二维码 + 手动密钥 + 首码确认 -->
    <ElDialog
      v-model="enrollVisible"
      :title="$t('pages.user.secureSetting.mfaBindTitle')"
      width="440px"
      align-center
      destroy-on-close
    >
      <div class="enroll-body">
        <p class="enroll-tip">{{ $t("pages.user.secureSetting.mfaScanQr") }}</p>
        <img v-if="qrUri" :src="qrUri" alt="TOTP QR" class="qr-img" />
        <p class="enroll-secret">
          {{ $t("pages.user.secureSetting.mfaManualEntry") }}:
          <code>{{ secret }}</code>
        </p>
        <p class="enroll-tip">{{ $t("pages.user.secureSetting.mfaEnterCode") }}</p>
        <ElInput
          v-model="confirmCode"
          :placeholder="$t('pages.user.secureSetting.mfaCodePlaceholder')"
          maxlength="6"
          inputmode="numeric"
          class="code-input"
          @keyup.enter="handleConfirmEnroll"
        />
      </div>
      <template #footer>
        <ElButton @click="closeEnroll">{{ $t("core.login.mfaCancel") }}</ElButton>
        <ElButton
          type="primary"
          :loading="confirmEnroll.isPending.value"
          :disabled="confirmCode.length !== 6"
          @click="handleConfirmEnroll"
        >
          {{ $t("pages.user.secureSetting.mfaConfirmBind") }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElMessageBox, ElNotification } from "element-plus";

import {
  useGetMfaStatus,
  useStartEnrollMfa,
  useConfirmEnrollMfa,
  useDisableMfa,
} from "@/api/composables";
import { $t } from "@/core/i18n";

// 已绑定的 TOTP 因子列表（不含 secret）
const { data: statusData, refetch } = useGetMfaStatus();
const enrolledItems = computed(() => statusData.value?.enrolled ?? []);
const totpItems = computed(() =>
  enrolledItems.value.filter((m) => m.method === "TOTP" && m.enabled)
);
const hasTotp = computed(() => totpItems.value.length > 0);

// 绑定流程状态
const enrollVisible = ref(false);
const qrUri = ref("");
const secret = ref("");
const opId = ref("");
const confirmCode = ref("");

const startEnroll = useStartEnrollMfa({
  onSuccess: (resp: any) => {
    const totpResult = resp?.result?.totp ?? resp?.totp;
    if (totpResult) {
      qrUri.value = totpResult.qrCodeDataUri ?? "";
      secret.value = totpResult.secret ?? "";
      opId.value = resp.operationId ?? "";
      enrollVisible.value = true;
    } else {
      ElNotification.error($t("pages.user.secureSetting.mfaEnrollStartFailed"));
    }
  },
  onError: (err: Error) =>
    ElNotification.error(err.message || $t("pages.user.secureSetting.mfaEnrollStartFailed")),
});

const confirmEnroll = useConfirmEnrollMfa({
  onSuccess: (resp: any) => {
    if (resp?.success) {
      ElNotification.success($t("pages.user.secureSetting.mfaBindSuccess"));
      closeEnroll();
      refetch();
    } else {
      ElNotification.error($t("pages.user.secureSetting.mfaBindFailed"));
    }
  },
  onError: (err: Error) =>
    ElNotification.error(err.message || $t("pages.user.secureSetting.mfaBindFailed")),
});

const disableMfa = useDisableMfa({
  onSuccess: () => {
    ElNotification.success($t("pages.user.secureSetting.mfaUnbindSuccess"));
    refetch();
  },
  onError: (err: Error) =>
    ElNotification.error(err.message || $t("pages.user.secureSetting.mfaUnbindFailed")),
});

const handleStartEnroll = () => {
  startEnroll.mutate({ method: "TOTP" } as any);
};

const handleConfirmEnroll = () => {
  if (confirmCode.value.length !== 6 || !opId.value) return;
  confirmEnroll.mutate({
    method: "TOTP",
    operationId: opId.value,
    totpCode: confirmCode.value,
  } as any);
};

const closeEnroll = () => {
  enrollVisible.value = false;
  qrUri.value = "";
  secret.value = "";
  opId.value = "";
  confirmCode.value = "";
};

const handleUnbind = () => {
  const target = totpItems.value[0];
  if (!target) return;
  ElMessageBox.confirm(
    $t("pages.user.secureSetting.mfaUnbindConfirmContent"),
    $t("pages.user.secureSetting.mfaUnbindConfirmTitle"),
    {
      confirmButtonText: $t("pages.user.secureSetting.mfaUnbind"),
      cancelButtonText: $t("core.login.mfaCancel"),
      type: "warning",
    }
  ).then(() => {
    disableMfa.mutate({ credentialId: target.id, method: "TOTP" } as any);
  });
};
</script>

<style lang="scss" scoped>
.setting-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin-bottom: 8px;
}

.item-title {
  flex: 1;
  font-size: 14px;
  font-weight: 500;
}

.item-description {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.enroll-body {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.enroll-tip {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin: 4px 0;
}

.qr-img {
  width: 200px;
  height: 200px;
}

.enroll-secret {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
  text-align: center;
}

.code-input {
  max-width: 200px;
  text-align: center;
}
</style>
