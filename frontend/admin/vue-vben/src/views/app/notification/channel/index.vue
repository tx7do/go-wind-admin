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
      // 只有 WEBHOOK 行会填这一列：邮件渠道这里是 '-'，不是"没配好"
      title: $t('page.notificationChannel.webhookUrl'),
      field: 'webhookUrl',
      minWidth: 200,
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.notificationChannel.webhookSignStyle'),
      field: 'webhookSignStyle',
      width: 130,
      slots: { default: 'signStyle' },
    },
    {
      title: $t('page.notificationChannel.hasPassword'),
      field: 'hasPassword',
      width: 100,
      slots: { default: 'hasPassword' },
    },
    {
      title: $t('page.notificationChannel.hasWebhookSecret'),
      field: 'hasWebhookSecret',
      width: 110,
      slots: { default: 'hasWebhookSecret' },
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

// 渠道类型按枚举查表渲染：一个类型对应自己那一组配置（SMTP 那几列对 WEBHOOK 没有意义，
// 反之 webhookUrl 对邮件渠道为空），混着显示会被读成"没配"。
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

// 出站风格：值与后端 webhook_style.go 里的同一组字符串逐字相同（枚举按名字配对）。
// 这一列可空，读到空值时按 CUSTOM 显示——它的实际行为就是 CUSTOM，写"自有方案"比 "-" 更像事实；
//（本机实测：PG 加列带 DEFAULT，存量行已被回填成 'CUSTOM'，空值只剩直接写 SQL 置 NULL 的行。）
// EMAIL 行没有这一列，才显示 "-"。
function signStyleLabel(row: NotificationChannel): string {
  if (row.type !== 'WEBHOOK') return '-';
  const map: Record<string, string> = {
    CUSTOM: $t('page.notificationChannel.signStyleCustom'),
    NONE: $t('page.notificationChannel.signStyleNone'),
    DINGTALK: $t('page.notificationChannel.signStyleDingtalk'),
    FEISHU: $t('page.notificationChannel.signStyleFeishu'),
    WECOM: $t('page.notificationChannel.signStyleWecom'),
  };
  return map[row.webhookSignStyle || 'CUSTOM'] || '-';
}

/**
 * 载荷模板的可引用变量清单。不写进词条：这些名字是接口契约的一部分（三种语言都不译），
 * 而且 vue-i18n 会把消息里的花括号当插值语法解析。
 */
const WEBHOOK_TEMPLATE_VARS = [
  'title',
  'content',
  'event_type',
  'timestamp',
  'sign',
  'nonce',
  'recipient_user_id',
  'related_id',
  'delivered_at',
]
  .map((name) => `{{${name}}}`)
  .join(' ');

// 示例取自钉钉那一档的内置形状：留空时后端就发这个形状，示例只是提示"可以改写"。
const WEBHOOK_TEMPLATE_EXAMPLE = '{"msgtype":"text","text":{"content":"{{title}}"}}';

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
  webhookUrl: '',
  webhookSignStyle: 'CUSTOM',
  webhookPayloadTemplate: '',
  webhookSecret: '',
  enabled: true,
  remark: '',
});

function openCreate() {
  editMode.value = 'create';
  editingId.value = undefined;
  Object.assign(form, {
    name: '', type: 'EMAIL', smtpHost: '', smtpPort: 587, smtpUsername: '',
    password: '', smtpFrom: '', smtpTls: 'START_TLS',
    webhookUrl: '', webhookSignStyle: 'CUSTOM', webhookPayloadTemplate: '',
    webhookSecret: '',
    enabled: true, remark: '',
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
    // 两处密钥都不回显（读视图只有 hasPassword / hasWebhookSecret 两个布尔）：留空 = 不改
    password: '',
    smtpFrom: row.smtpFrom || '',
    smtpTls: row.smtpTls || 'START_TLS',
    webhookUrl: row.webhookUrl || '',
    webhookSignStyle: row.webhookSignStyle || 'CUSTOM',
    webhookPayloadTemplate: row.webhookPayloadTemplate || '',
    webhookSecret: '',
    enabled: !!row.enabled,
    remark: row.remark || '',
  });
  editOpen.value = true;
}

/**
 * 只提交本类型那一组配置——等价于 react 端「另一组的表单项压根没挂载」。
 * 把两组一起提交会把对端那一列写空：编辑一条 WEBHOOK 渠道不该顺手清掉它曾经配过的 SMTP。
 */
function buildChannelData(): Record<string, any> {
  const data: Record<string, any> = {
    name: form.name,
    type: form.type,
    enabled: form.enabled,
    remark: form.remark,
  };
  if (form.type === 'WEBHOOK') {
    data.webhookUrl = form.webhookUrl;
    data.webhookSignStyle = form.webhookSignStyle;
    data.webhookPayloadTemplate = form.webhookPayloadTemplate;
  } else {
    data.smtpHost = form.smtpHost;
    data.smtpPort = form.smtpPort;
    data.smtpUsername = form.smtpUsername;
    data.smtpFrom = form.smtpFrom;
    data.smtpTls = form.smtpTls;
  }
  return data;
}

async function handleSave() {
  if (!form.name) {
    message.error($t('page.notificationChannel.requiredName'));
    return;
  }
  if (editMode.value === 'create') {
    if (form.type === 'WEBHOOK' && !form.webhookUrl) {
      message.error($t('page.notificationChannel.requiredWebhookUrl'));
      return;
    }
    if (form.type !== 'WEBHOOK' && !form.password) {
      message.error($t('page.notificationChannel.requiredPassword'));
      return;
    }
  }
  editSaving.value = true;
  try {
    // 两处密钥是**请求级**字段（与 data 平级，不在 data 里）：读 DTO 只有两个布尔，
    // 留空表示不改已存值
    const data = buildChannelData();
    const password = form.password || undefined;
    const webhookSecret = form.webhookSecret || undefined;
    if (editMode.value === 'create') {
      await createChannel({ data, password, webhookSecret });
      message.success($t('page.notificationChannel.createSuccess'));
    } else if (editingId.value) {
      await updateChannel({
        id: editingId.value,
        data,
        password,
        webhookSecret,
        updateMask:
          'name,type,smtpHost,smtpPort,smtpUsername,smtpFrom,smtpTls,webhookUrl,webhookSignStyle,webhookPayloadTemplate,enabled,remark',
      });
      message.success($t('page.notificationChannel.updateSuccess'));
    }
    editOpen.value = false;
    await gridApi.reload();
  } catch (error: any) {
    // 不吞错：原始错误对象进控制台，翻译文案只给人看
    console.error('[notification-channel] save failed', error);
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
    console.error('[notification-channel] delete failed', error);
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
    console.error('[notification-channel] toggle enabled failed', error);
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
  if (!testTarget.value?.id) return;
  if (!testRecipient.value) {
    message.error($t('page.notificationChannel.requiredRecipient'));
    return;
  }
  testSaving.value = true;
  try {
    await sendTest({ id: testTarget.value.id, recipient: testRecipient.value });
    notification.success({ message: $t('page.notificationChannel.testSendSuccess') });
    testOpen.value = false;
  } catch (error: any) {
    console.error('[notification-channel] test send failed', error);
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
      <template #signStyle="{ row }">
        <a-tag v-if="row.type === 'WEBHOOK'">{{ signStyleLabel(row) }}</a-tag>
        <template v-else>-</template>
      </template>
      <template #hasPassword="{ row }">
        <a-tag v-if="row.hasPassword" color="success">
          {{ $t('page.notificationChannel.passwordSet') }}
        </a-tag>
        <a-tag v-else>{{ $t('page.notificationChannel.passwordNotSet') }}</a-tag>
      </template>
      <template #hasWebhookSecret="{ row }">
        <a-tag v-if="row.hasWebhookSecret" color="success">
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
        <!-- 测试发送只对 EMAIL 行有意义：它测的就是这一个 SMTP 账号能否握手发信。
             WEBHOOK 的"当场试一次"在路由规则页（那里测的是事件 → 渠道 → 台账整条链）。 -->
        <a-button
          v-if="row.type === 'EMAIL'"
          type="link"
          size="small"
          @click.stop="openTestSend(row)"
        >
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
        <a-form-item :label="$t('page.notificationChannel.type')" required>
          <!-- 编辑态禁改类型：改类型等于换一条渠道实现，与规则页禁改 eventType 同理 -->
          <a-select v-model:value="form.type" :disabled="editMode === 'edit'">
            <a-select-option value="EMAIL">
              {{ $t('page.notificationChannel.typeEmail') }}
            </a-select-option>
            <a-select-option value="WEBHOOK">
              {{ $t('page.notificationChannel.typeWebhook') }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <!-- 两类渠道各自的配置项分组显示：表单里同时摆两组，管理员要自己猜哪一组生效。
             编辑态 type 不可改，所以两组不会在同一个会话里来回切。 -->
        <template v-if="form.type === 'WEBHOOK'">
          <a-form-item
            :label="$t('page.notificationChannel.webhookUrl')"
            :help="$t('page.notificationChannel.webhookUrlHint')"
            :required="editMode === 'create'"
          >
            <a-input
              v-model:value="form.webhookUrl"
              placeholder="https://example.com/hooks/notification"
            />
          </a-form-item>
          <a-form-item
            :label="$t('page.notificationChannel.webhookSignStyle')"
            :help="$t('page.notificationChannel.webhookSignStyleHint')"
          >
            <a-select v-model:value="form.webhookSignStyle">
              <a-select-option value="CUSTOM">
                {{ $t('page.notificationChannel.signStyleCustom') }}
              </a-select-option>
              <a-select-option value="NONE">
                {{ $t('page.notificationChannel.signStyleNone') }}
              </a-select-option>
              <a-select-option value="DINGTALK">
                {{ $t('page.notificationChannel.signStyleDingtalk') }}
              </a-select-option>
              <a-select-option value="FEISHU">
                {{ $t('page.notificationChannel.signStyleFeishu') }}
              </a-select-option>
              <a-select-option value="WECOM">
                {{ $t('page.notificationChannel.signStyleWecom') }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <!-- 留空 = 用该风格的内置默认正文（内置形状由后端 webhook_style.go 持有，
               这里不复制一份，否则两处会漂）。 -->
          <a-form-item
            :label="$t('page.notificationChannel.webhookPayloadTemplate')"
            :help="$t('page.notificationChannel.webhookPayloadTemplateHint', { vars: WEBHOOK_TEMPLATE_VARS })"
          >
            <a-textarea
              v-model:value="form.webhookPayloadTemplate"
              :rows="3"
              :placeholder="WEBHOOK_TEMPLATE_EXAMPLE"
              class="font-mono"
            />
          </a-form-item>
          <a-form-item
            :label="$t('page.notificationChannel.webhookSecret')"
            :help="$t('page.notificationChannel.webhookSecretHint')"
          >
            <a-input-password
              v-model:value="form.webhookSecret"
              :placeholder="
                editMode === 'edit'
                  ? $t('page.notificationChannel.webhookSecretKeepHint')
                  : $t('page.notificationChannel.webhookSecretPlaceholder')
              "
            />
          </a-form-item>
        </template>
        <template v-else>
          <a-form-item :label="$t('page.notificationChannel.smtpHost')">
            <a-input v-model:value="form.smtpHost" placeholder="smtp.example.com" />
          </a-form-item>
          <a-form-item :label="$t('page.notificationChannel.smtpPort')">
            <a-input-number v-model:value="form.smtpPort" :min="1" :max="65535" class="w-full" />
          </a-form-item>
          <a-form-item :label="$t('page.notificationChannel.smtpUsername')">
            <a-input v-model:value="form.smtpUsername" />
          </a-form-item>
          <a-form-item
            :label="$t('page.notificationChannel.password')"
            :required="editMode === 'create'"
          >
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
        </template>
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
