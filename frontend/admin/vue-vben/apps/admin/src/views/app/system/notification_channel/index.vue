<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message, notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListNotificationChannels,
  useCreateNotificationChannel,
  useDeleteNotificationChannel,
  useSendTestEmail,
  useUpdateNotificationChannel,
} from '#/api';
import TableExportButton from '#/components/TableExportButton.vue';
import type {
  notification_channelservicev1_NotificationChannel as NotificationChannel,
} from '#/api/generated/admin/service/v1';

const { mutateAsync: createChannel } = useCreateNotificationChannel();
const { mutateAsync: updateChannel } = useUpdateNotificationChannel();
const { mutateAsync: deleteChannel } = useDeleteNotificationChannel();
const { mutateAsync: sendTest } = useSendTestEmail();

const formOptions = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.notificationChannel.name'),
      componentProps: { allowClear: true },
    },
  ],
};

const gridOptions: VxeGridProps<NotificationChannel> = {
  toolbarConfig: {
    custom: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  pagerConfig: {},
  rowConfig: { isHover: true, keyField: 'id' },
  stripe: true,
  proxyConfig: {
    ajax: {
      query: async ({ page }) => {
        return await fetchListNotificationChannels({
          page: page.currentPage,
          pageSize: page.pageSize,
        });
      },
    },
  },
  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.notificationChannel.name'), field: 'name', minWidth: 140 },
    {
      title: $t('page.notificationChannel.type'),
      field: 'type',
      width: 110,
      slots: { default: 'type' },
    },
    { title: $t('page.notificationChannel.smtpHost'), field: 'smtpHost', minWidth: 150 },
    { title: $t('page.notificationChannel.smtpPort'), field: 'smtpPort', width: 80 },
    { title: $t('page.notificationChannel.smtpFrom'), field: 'smtpFrom', minWidth: 170 },
    {
      title: $t('page.notificationChannel.smtpTls'),
      field: 'smtpTls',
      width: 110,
      slots: { default: 'tls' },
    },
    {
      title: $t('page.notificationChannel.hasPassword'),
      field: 'hasPassword',
      width: 100,
      slots: { default: 'hasPassword' },
    },
    {
      title: $t('page.notificationChannel.enabled'),
      field: 'enabled',
      width: 90,
      slots: { default: 'enabled' },
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 230,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListNotificationChannels({ page, pageSize });

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

function tlsLabel(mode?: string): string {
  const map: Record<string, string> = {
    NONE: $t('page.notificationChannel.tlsNone'),
    START_TLS: $t('page.notificationChannel.tlsStartTls'),
    SSL: $t('page.notificationChannel.tlsSsl'),
  };
  return (mode && map[mode]) || mode || '-';
}

// 渠道类型同样按枚举查表：WEBHOOK 还没有发送实现（P2 才落地），建渠道的表单因此只给
// EMAIL 选项，但列不能写死成"邮件 (SMTP)"——已有行是 WEBHOOK 时显示就是错的。
function typeLabel(type?: string): string {
  const map: Record<string, string> = {
    EMAIL: $t('page.notificationChannel.typeEmail'),
    WEBHOOK: $t('page.notificationChannel.typeWebhook'),
  };
  return (type && map[type]) || type || '-';
}

function typeColor(type?: string): string {
  const map: Record<string, string> = {
    EMAIL: 'blue',
    WEBHOOK: 'purple',
  };
  return (type && map[type]) || 'default';
}

// ============ 创建/编辑 ============
const editOpen = ref(false);
const editMode = ref<'create' | 'edit'>('create');
const editSaving = ref(false);
const editingId = ref<number>();
const form = reactive({
  name: '',
  type: 'EMAIL',
  smtpHost: '',
  smtpPort: 587,
  smtpUsername: '',
  password: '',
  smtpFrom: '',
  smtpTls: 'START_TLS',
  enabled: true,
  remark: '',
});

function openCreate() {
  editMode.value = 'create';
  editingId.value = undefined;
  Object.assign(form, {
    name: '', type: 'EMAIL', smtpHost: '', smtpPort: 587, smtpUsername: '',
    password: '', smtpFrom: '', smtpTls: 'START_TLS', enabled: true, remark: '',
  });
  editOpen.value = true;
}

function openEdit(row: NotificationChannel) {
  editMode.value = 'edit';
  editingId.value = row.id;
  Object.assign(form, {
    name: row.name || '',
    type: row.type || 'EMAIL',
    smtpHost: row.smtpHost || '',
    smtpPort: row.smtpPort ?? 587,
    smtpUsername: row.smtpUsername || '',
    password: '',
    smtpFrom: row.smtpFrom || '',
    smtpTls: row.smtpTls || 'START_TLS',
    enabled: !!row.enabled,
    remark: row.remark || '',
  });
  editOpen.value = true;
}

async function handleSave() {
  if (!form.name) {
    message.error($t('page.notificationChannel.requiredName'));
    return;
  }
  editSaving.value = true;
  try {
    const { password, ...data } = form as Record<string, any>;
    if (editMode.value === 'create') {
      if (!password) {
        message.error($t('page.notificationChannel.requiredPassword'));
        return;
      }
      await createChannel({ data, password });
      message.success($t('page.notificationChannel.createSuccess'));
    } else if (editingId.value) {
      await updateChannel({
        id: editingId.value,
        data,
        password: password || undefined,
        updateMask:
          'name,smtpHost,smtpPort,smtpUsername,smtpFrom,smtpTls,enabled,remark',
      });
      message.success($t('page.notificationChannel.updateSuccess'));
    }
    editOpen.value = false;
    await gridApi.reload();
  } catch (error: any) {
    message.error(error?.message || $t('page.notificationChannel.saveFailed'));
  } finally {
    editSaving.value = false;
  }
}

// ============ 删除 / 启停 / 测试发送 ============
async function handleDelete(row: NotificationChannel) {
  try {
    await deleteChannel({ id: row.id });
    notification.success({ message: $t('ui.notification.delete_success') });
    await gridApi.reload();
  } catch (error: any) {
    notification.error({ message: error?.message || $t('ui.notification.delete_failed') });
  }
}

async function handleToggleEnabled(row: NotificationChannel) {
  const next = !row.enabled;
  try {
    await updateChannel({
      id: row.id,
      data: { name: row.name, type: row.type, enabled: next },
      updateMask: 'enabled',
    });
    notification.success({ message: $t('ui.notification.update_success') });
    await gridApi.reload();
  } catch (error: any) {
    notification.error({ message: error?.message || $t('ui.notification.update_failed') });
  }
}

const testOpen = ref(false);
const testSaving = ref(false);
const testTarget = ref<NotificationChannel>();
const testRecipient = ref('');

function openTestSend(row: NotificationChannel) {
  testTarget.value = row;
  testRecipient.value = '';
  testOpen.value = true;
}

async function handleTestSend() {
  if (!testTarget.value?.id || !testRecipient.value) return;
  testSaving.value = true;
  try {
    await sendTest({ id: testTarget.value.id, recipient: testRecipient.value });
    notification.success({ message: $t('page.notificationChannel.testSendSuccess') });
    testOpen.value = false;
  } catch (error: any) {
    notification.error({
      message: error?.message || $t('page.notificationChannel.testSendFailed'),
    });
  } finally {
    testSaving.value = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.notificationChannel.moduleName')">
      <template #toolbar-tools>
        <a-button type="primary" class="mr-2" @click="openCreate">
          {{ $t('page.notificationChannel.create') }}
        </a-button>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="notification-channels" />
      </template>
      <template #type="{ row }">
        <a-tag v-if="row.type" :color="typeColor(row.type)">{{ typeLabel(row.type) }}</a-tag>
        <template v-else>-</template>
      </template>
      <template #tls="{ row }">{{ tlsLabel(row.smtpTls) }}</template>
      <template #hasPassword="{ row }">
        <a-tag v-if="row.hasPassword" color="success">
          {{ $t('page.notificationChannel.passwordSet') }}
        </a-tag>
        <a-tag v-else>{{ $t('page.notificationChannel.passwordNotSet') }}</a-tag>
      </template>
      <template #enabled="{ row }">
        <a-switch
          :checked="row.enabled"
          :checked-children="$t('page.notificationChannel.enabledOn')"
          @click="handleToggleEnabled(row)"
        />
      </template>
      <template #action="{ row }">
        <a-button type="link" size="small" @click.stop="openEdit(row)">
          {{ $t('common.edit') }}
        </a-button>
        <a-button type="link" size="small" @click.stop="openTestSend(row)">
          {{ $t('page.notificationChannel.testSend') }}
        </a-button>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="$t('page.notificationChannel.deleteConfirm')"
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" size="small">
            {{ $t('common.delete') }}
          </a-button>
        </a-popconfirm>
      </template>
    </Grid>

    <!-- 创建/编辑 -->
    <a-modal
      v-model:open="editOpen"
      :title="editMode === 'create' ? $t('page.notificationChannel.create') : $t('page.notificationChannel.edit')"
      :confirm-loading="editSaving"
      destroy-on-close
      @ok="handleSave"
    >
      <a-form layout="vertical" class="pt-2">
        <a-form-item :label="$t('page.notificationChannel.name')" required>
          <a-input v-model:value="form.name" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.type')">
          <a-select v-model:value="form.type" :disabled="editMode === 'edit'">
            <a-select-option value="EMAIL">
              {{ $t('page.notificationChannel.typeEmail') }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.smtpHost')">
          <a-input v-model:value="form.smtpHost" placeholder="smtp.example.com" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.smtpPort')">
          <a-input-number v-model:value="form.smtpPort" :min="1" :max="65535" class="w-full" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.smtpUsername')">
          <a-input v-model:value="form.smtpUsername" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.password')">
          <a-input-password
            v-model:value="form.password"
            :placeholder="
              editMode === 'edit'
                ? $t('page.notificationChannel.passwordKeepHint')
                : $t('page.notificationChannel.passwordPlaceholder')
            "
          />
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.smtpFrom')">
          <a-input v-model:value="form.smtpFrom" placeholder="noreply@example.com" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.smtpTls')">
          <a-select v-model:value="form.smtpTls">
            <a-select-option value="NONE">
              {{ $t('page.notificationChannel.tlsNone') }}
            </a-select-option>
            <a-select-option value="START_TLS">
              {{ $t('page.notificationChannel.tlsStartTls') }}
            </a-select-option>
            <a-select-option value="SSL">
              {{ $t('page.notificationChannel.tlsSsl') }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.enabled')">
          <a-switch v-model:checked="form.enabled" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.remark')">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 测试发送 -->
    <a-modal
      v-model:open="testOpen"
      :title="
        $t('page.notificationChannel.testSendTitle', { name: testTarget?.name || '' })
      "
      :confirm-loading="testSaving"
      @ok="handleTestSend"
    >
      <a-input
        v-model:value="testRecipient"
        class="mt-4"
        :placeholder="$t('page.notificationChannel.testRecipient')"
      />
    </a-modal>
  </Page>
</template>
