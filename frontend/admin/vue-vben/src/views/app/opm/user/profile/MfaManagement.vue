<script lang="ts" setup>
import { computed, ref } from 'vue';

import { Button, Image, Input, Modal, notification } from 'ant-design-vue';

import { $t } from '@vben/locales';

import {
  useGetMfaStatus,
  useStartEnrollMfa,
  useConfirmEnrollMfa,
  useDisableMfa,
} from '#/api/composables';

// 已绑定的 TOTP 因子列表（不含 secret）
const { data: statusData, refetch } = useGetMfaStatus();
const enrolledItems = computed(() => statusData.value?.enrolled ?? []);
const totpItems = computed(() =>
  enrolledItems.value.filter((m) => m.method === 'TOTP' && m.enabled),
);
const hasTotp = computed(() => totpItems.value.length > 0);

// 绑定流程状态
const enrollVisible = ref(false);
const qrUri = ref('');
const secret = ref('');
const opId = ref('');
const confirmCode = ref('');

const startEnroll = useStartEnrollMfa({
  onSuccess: (resp: any) => {
    const totpResult = resp?.result?.totp ?? resp?.totp;
    if (totpResult) {
      qrUri.value = totpResult.qrCodeDataUri ?? '';
      secret.value = totpResult.secret ?? '';
      opId.value = resp.operationId ?? '';
      enrollVisible.value = true;
    } else {
      notification.error({ message: $t('authentication.mfaEnrollStartFailed') });
    }
  },
  onError: (err: Error) =>
    notification.error({
      message: err.message || $t('authentication.mfaEnrollStartFailed'),
    }),
});

const confirmEnroll = useConfirmEnrollMfa({
  onSuccess: (resp: any) => {
    if (resp?.success) {
      notification.success({ message: $t('authentication.mfaBindSuccess') });
      closeEnroll();
      refetch();
    } else {
      notification.error({ message: $t('authentication.mfaBindFailed') });
    }
  },
  onError: (err: Error) =>
    notification.error({ message: err.message || $t('authentication.mfaBindFailed') }),
});

const disableMfa = useDisableMfa({
  onSuccess: () => {
    notification.success({ message: $t('authentication.mfaUnbindSuccess') });
    refetch();
  },
  onError: (err: Error) =>
    notification.error({ message: err.message || $t('authentication.mfaUnbindFailed') }),
});

const handleStartEnroll = () => {
  startEnroll.mutate({ method: 'TOTP' } as any);
};

const handleConfirmEnroll = () => {
  if (confirmCode.value.length !== 6 || !opId.value) return;
  confirmEnroll.mutate({
    method: 'TOTP',
    operationId: opId.value,
    totpCode: confirmCode.value,
  } as any);
};

const closeEnroll = () => {
  enrollVisible.value = false;
  qrUri.value = '';
  secret.value = '';
  opId.value = '';
  confirmCode.value = '';
};

const handleUnbind = () => {
  const target = totpItems.value[0];
  if (!target) return;
  Modal.confirm({
    title: $t('authentication.mfaUnbindConfirmTitle'),
    content: $t('authentication.mfaUnbindConfirmContent'),
    okText: $t('authentication.mfaUnbind'),
    okType: 'danger',
    cancelText: $t('authentication.mfaCancel'),
    onOk: () => {
      disableMfa.mutate({ credentialId: target.id, method: 'TOTP' } as any);
    },
  });
};
</script>

<template>
  <div class="mfa-management">
    <div class="mfa-row">
      <div class="mfa-info">
        <div class="mfa-title">{{ $t('authentication.mfaDeviceTitle') }}</div>
        <div class="mfa-desc">
          {{
            hasTotp
              ? $t('authentication.mfaBoundDesc')
              : $t('authentication.mfaDeviceDesc')
          }}
        </div>
      </div>
      <div class="mfa-actions">
        <Button
          v-if="!hasTotp"
          type="primary"
          size="small"
          :loading="startEnroll.isPending.value"
          @click="handleStartEnroll"
        >
          {{ $t('authentication.mfaStartBind') }}
        </Button>
        <Button
          v-else
          danger
          size="small"
          :loading="disableMfa.isPending.value"
          @click="handleUnbind"
        >
          {{ $t('authentication.mfaUnbind') }}
        </Button>
      </div>
    </div>

    <Modal
      v-model:open="enrollVisible"
      :title="$t('authentication.mfaBindTitle')"
      :footer="null"
      width="440px"
      destroy-on-close
    >
      <div class="enroll-body">
        <p class="enroll-tip">{{ $t('authentication.mfaScanQr') }}</p>
        <Image v-if="qrUri" :src="qrUri" :width="200" :height="200" />
        <p class="enroll-secret">
          {{ $t('authentication.mfaManualEntry') }}:
          <code>{{ secret }}</code>
        </p>
        <p class="enroll-tip">{{ $t('authentication.mfaEnterCode') }}</p>
        <Input
          v-model:value="confirmCode"
          :placeholder="$t('authentication.mfaCodePlaceholder')"
          :maxlength="6"
          class="code-input"
          @keyup.enter="handleConfirmEnroll"
        />
        <div class="enroll-footer">
          <Button @click="closeEnroll">{{ $t('authentication.mfaCancel') }}</Button>
          <Button
            type="primary"
            :loading="confirmEnroll.isPending.value"
            :disabled="confirmCode.length !== 6"
            @click="handleConfirmEnroll"
          >
            {{ $t('authentication.mfaConfirmBind') }}
          </Button>
        </div>
      </div>
    </Modal>
  </div>
</template>

<style lang="less" scoped>
.mfa-management {
  padding: 16px 0;
}

.mfa-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.mfa-info {
  flex: 1;
  min-width: 0;
}

.mfa-title {
  font-size: 14px;
  font-weight: 500;
}

.mfa-desc {
  margin-top: 4px;
  font-size: 13px;
  color: var(--muted-foreground);
  line-height: 1.5;
}

.enroll-body {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.enroll-tip {
  margin: 4px 0;
  font-size: 13px;
  color: var(--muted-foreground);
}

.enroll-secret {
  font-size: 12px;
  color: var(--muted-foreground);
  word-break: break-all;
  text-align: center;
}

.code-input {
  max-width: 200px;
  text-align: center;
}

.enroll-footer {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}
</style>
