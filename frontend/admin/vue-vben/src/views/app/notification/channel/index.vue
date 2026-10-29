<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { computed, h, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { LucideFilePenLine, LucideSend, LucideTrash2 } from '@vben/icons';
import { $t } from '@vben/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';

import { message, notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListNotificationChannels,
  useCreateNotificationChannel,
  useDeleteNotificationChannel,
  useSendTest,
  useUpdateNotificationChannel,
} from '#/api';
import {
  EXT_GROUP_LABELS,
  fetchListChannelSchemas,
  parseSettingsSafe,
  type NotificationChannelSchema,
} from '#/api/composables/notification-channel-schema';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';
import type {
  notification_channelservicev1_NotificationChannel as NotificationChannel,
} from '#/api/generated/admin/service/v1';

const { mutateAsync: createChannel } = useCreateNotificationChannel();
const { mutateAsync: updateChannel } = useUpdateNotificationChannel();
const { mutateAsync: deleteChannel } = useDeleteNotificationChannel();
const { mutateAsync: sendTest } = useSendTest();

// ---- 扩展渠道 schema（类型下拉分组 + 动态表单的数据源）----
const extSchemas = ref<NotificationChannelSchema[]>([]);
fetchListChannelSchemas().then((items) => {
  extSchemas.value = items;
});

function isExtType(type?: string): boolean {
  return !!type && extSchemas.value.some((s) => s.type === type);
}

function extSchemaOf(type?: string): NotificationChannelSchema | undefined {
  return isExtType(type) ? extSchemas.value.find((s) => s.type === type) : undefined;
}

/** 类型下拉分组：SMS 组首插拆分项（阿里云/腾讯云合成键），其余组按 schema。 */
const extTypeGroups = computed(() => {
  const acc: {
    group: string;
    label: string;
    options: { label: string; value: string }[];
  }[] = [];
  for (const s of extSchemas.value) {
    const group = s.group ?? 'EXT';
    let node = acc.find((g) => g.group === group);
    if (!node) {
      node = { group, label: EXT_GROUP_LABELS[group] ?? group, options: [] };
      acc.push(node);
    }
    node.options.push({ label: s.label ?? s.type ?? '', value: s.type ?? '' });
  }
  const smsGroup = acc.find((g) => g.group === 'SMS');
  if (smsGroup) {
    smsGroup.options.unshift(
      {
        label: $t('page.notificationChannel.smsAliyun'),
        value: 'SMS@ALIYUN',
      },
      {
        label: $t('page.notificationChannel.smsTencent'),
        value: 'SMS@TENCENT',
      },
    );
  }
  return acc;
});

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
      title: $t('page.notificationChannel.smsProvider'),
      field: 'smsProvider',
      width: 110,
      slots: { default: 'smsProvider' },
    },
    {
      title: $t('page.notificationChannel.smsSignName'),
      field: 'smsSignName',
      width: 110,
      showOverflow: 'tooltip',
      formatter: ({ row }) => (row.type === 'SMS' ? row.smsSignName || '-' : '-'),
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
      title: $t('page.notificationChannel.hasSmsAccessSecret'),
      field: 'hasSmsAccessSecret',
      width: 110,
      slots: { default: 'hasSmsAccessSecret' },
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
    SMS: $t('page.notificationChannel.typeSms'),
  };
  // EXT_* 扩展渠道的展示名来自 schema（i18n 不逐渠道维护）
  const ext = extSchemaOf(type);
  if (ext) return ext.label ?? type ?? '-';
  return (type && map[type]) || type || '-';
}

function typeColor(type?: string): string {
  const map: Record<string, string> = {
    EMAIL: 'blue',
    WEBHOOK: 'purple',
    SMS: 'cyan',
  };
  if (isExtType(type)) return 'geekblue';
  return (type && map[type]) || 'default';
}

// 短信服务商：值与 proto SmsProvider 成员名的同一组字符串（枚举按名字配对）。
function smsProviderLabel(row: NotificationChannel): string {
  if (row.type !== 'SMS') return '-';
  const map: Record<string, string> = {
    ALIYUN: $t('page.notificationChannel.providerAliyun'),
    TENCENT: $t('page.notificationChannel.providerTencent'),
  };
  return (row.smsProvider && map[row.smsProvider]) || row.smsProvider || '-';
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
  smtpFromName: '',
  smtpTls: 'START_TLS',
  webhookUrl: '',
  webhookSignStyle: 'CUSTOM',
  webhookPayloadTemplate: '',
  webhookSecret: '',
  smsProvider: 'ALIYUN',
  smsAccessKeyId: '',
  smsAccessSecret: '',
  smsSignName: '',
  smsTemplateCode: '',
  smsAppId: '',
  smsRegion: '',
  settings: {} as Record<string, any>,
  enabled: true,
  remark: '',
});

function openCreate() {
  editMode.value = 'create';
  editingId.value = undefined;
  Object.assign(form, {
    name: '', type: 'EMAIL', smtpHost: '', smtpPort: 587, smtpUsername: '',
    password: '', smtpFrom: '', smtpFromName: '', smtpTls: 'START_TLS',
    webhookUrl: '', webhookSignStyle: 'CUSTOM', webhookPayloadTemplate: '',
    webhookSecret: '',
    smsProvider: 'ALIYUN', smsAccessKeyId: '', smsAccessSecret: '', smsSignName: '',
    smsTemplateCode: '', smsAppId: '', smsRegion: '',
    settings: {},
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
    // 三处密钥都不回显（读视图只有 hasPassword / hasWebhookSecret / hasSmsAccessSecret 三个布尔）：留空 = 不改
    password: '',
    smtpFrom: row.smtpFrom || '',
    smtpFromName: row.smtpFromName || '',
    smtpTls: row.smtpTls || 'START_TLS',
    webhookUrl: row.webhookUrl || '',
    webhookSignStyle: row.webhookSignStyle || 'CUSTOM',
    webhookPayloadTemplate: row.webhookPayloadTemplate || '',
    webhookSecret: '',
    smsProvider: row.smsProvider || 'ALIYUN',
    smsAccessKeyId: row.smsAccessKeyId || '',
    smsSignName: row.smsSignName || '',
    smsTemplateCode: row.smsTemplateCode || '',
    smsAppId: row.smsAppId || '',
    smsRegion: row.smsRegion || '',
    // settings 读回显是脱敏 JSON 串：parse 成对象供动态表单回填（sensitive 空串占位）
    settings: parseSettingsSafe(row.settings),
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
  } else if (isExtType(form.type)) {
    data.settings = JSON.stringify(form.settings);
  } else if (form.type === 'SMS') {
    data.smsProvider = form.smsProvider;
    data.smsAccessKeyId = form.smsAccessKeyId;
    data.smsSignName = form.smsSignName;
    data.smsTemplateCode = form.smsTemplateCode;
    data.smsAppId = form.smsAppId;
    data.smsRegion = form.smsRegion;
  } else {
    data.smtpHost = form.smtpHost;
    data.smtpPort = form.smtpPort;
    data.smtpUsername = form.smtpUsername;
    data.smtpFrom = form.smtpFrom;
    data.smtpTls = form.smtpTls;
  }
  return data;
}

// SMS 各服务商的必填项（创建时校验；编辑留空=不改密钥所以 Secret 只在创建时必填）。
// 腾讯云多要 smsAppId / smsRegion：必留必填差在后端报错里点名，这里拦创建时的硬缺口。
function smsMissingField(): string | undefined {
  if (!form.smsProvider) return 'requiredSmsProvider';
  if (!form.smsAccessKeyId) return 'requiredSmsAccessKeyId';
  if (!form.smsSignName) return 'requiredSmsSignName';
  if (!form.smsTemplateCode) return 'requiredSmsTemplateCode';
  if (form.smsProvider === 'TENCENT') {
    if (!form.smsAppId) return 'smsAppIdHint';
    if (!form.smsRegion) return 'smsRegionHint';
  }
  return undefined;
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
    if (form.type === 'EMAIL' && !form.password) {
      message.error($t('page.notificationChannel.requiredPassword'));
      return;
    }
    if (form.type === 'SMS') {
      if (!form.smsAccessSecret) {
        message.error($t('page.notificationChannel.requiredSmsAccessSecret'));
        return;
      }
      const missing = smsMissingField();
      if (missing) {
        message.error($t(`page.notificationChannel.${missing}`));
        return;
      }
    }
    // EXT：schema required 前端先拦一遍（后端保存期同规则兜底）
    if (isExtType(form.type) && editMode.value === 'create') {
      for (const field of extSchemaOf(form.type)?.fields ?? []) {
        if (field.required && !form.settings[field.name!]) {
          message.error(`${field.label ?? field.name} is required`);
          return;
        }
      }
    }
  }
  editSaving.value = true;
  try {
    // 三处密钥是**请求级**字段（与 data 平级，不在 data 里）：读 DTO 只有三个布尔，
    // 留空表示不改已存值
    const data = buildChannelData();
    const password = form.password || undefined;
    const webhookSecret = form.webhookSecret || undefined;
    const smsAccessSecret = form.smsAccessSecret || undefined;
    if (editMode.value === 'create') {
      await createChannel({ data, password, webhookSecret, smsAccessSecret });
      message.success($t('page.notificationChannel.createSuccess'));
    } else if (editingId.value) {
      await updateChannel({
        id: editingId.value,
        data,
        password,
        webhookSecret,
        smsAccessSecret,
        updateMask: isExtType(form.type)
          ? 'name,type,settings,enabled,remark'
          : 'name,type,smtpHost,smtpPort,smtpUsername,smtpFrom,smtpFromName,smtpTls,webhookUrl,webhookSignStyle,webhookPayloadTemplate,smsProvider,smsAccessKeyId,smsSignName,smsTemplateCode,smsAppId,smsRegion,enabled,remark',
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

/* 操作列描述符：编辑 + 测试发送（仅 EMAIL/SMS）+ 删除，三钮全行内。
   测试发送只对有"收件目标"的渠道有意义：EMAIL 测 SMTP 握手发信、SMS 发一条真验证码。
   WEBHOOK 的"当场试一次"在路由规则页（那里测的是事件 → 渠道 → 台账整条链）。 */
function buildRowActions(row: NotificationChannel): TableAction[] {
  return [
    {
      key: 'edit',
      label: $t('common.edit'),
      icon: h(LucideFilePenLine),
      onClick: openEdit,
    },
    {
      key: 'test-send',
      label: $t('page.notificationChannel.testSend'),
      icon: h(LucideSend),
      hidden: !(row.type === 'EMAIL' || row.type === 'SMS'),
      onClick: openTestSend,
    },
    {
      key: 'delete',
      label: $t('common.delete'),
      icon: h(LucideTrash2),
      danger: true,
      confirm: { title: $t('page.notificationChannel.deleteConfirm') },
      onClick: handleDelete,
    },
  ];
}

async function handleTestSend() {
  if (!testTarget.value?.id) return;
  if (!testRecipient.value) {
    message.error(
      testTarget.value.type === 'SMS'
        ? $t('page.notificationChannel.requiredRecipientPhone')
        : $t('page.notificationChannel.requiredRecipient'),
    );
    return;
  }
  if (testTarget.value.type === 'SMS' && !/^\+?\d[\d\s-]{4,19}$/.test(testRecipient.value)) {
    message.error($t('page.notificationChannel.invalidPhone'));
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
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="notification-channels" />
        <TableAddButton @click="openCreate" />
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
      <template #smsProvider="{ row }">
        <a-tag v-if="row.type === 'SMS'">{{ smsProviderLabel(row) }}</a-tag>
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
      <template #hasSmsAccessSecret="{ row }">
        <a-tag v-if="row.hasSmsAccessSecret" color="success">
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
        <TableActions :row="row" :actions="buildRowActions(row)" />
      </template>
    </Grid>

    <!-- 创建/编辑 -->
    <a-drawer
      v-model:open="editOpen"
      :title="editMode === 'create' ? $t('page.notificationChannel.create') : $t('page.notificationChannel.edit')"
      destroy-on-close
      :width="600"
    >
      <!-- 标签与输入同行（react 端抽屉规范同款）：flex 定宽标签 + grow 输入列，
           140px 容下「SMTP 密码/授权码」「访问密钥 Secret」等长标签（与 ele 端同值）。
           输入列 flex 必须是 '1'（basis 0）而非 'auto'：auto 的 basis 取内容 max-content，
           带 :help 长文案的项会把行撑到 wrap、标签翻到输入上方 -->
      <a-form
        layout="horizontal"
        :label-col="{ flex: '0 0 140px' }"
        :wrapper-col="{ flex: '1' }"
        class="pt-2"
      >
        <a-form-item :label="$t('page.notificationChannel.name')" required>
          <a-input v-model:value="form.name" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationChannel.type')" required>
          <!-- 编辑态禁改类型：改类型等于换一条渠道实现，与规则页禁改 eventType 同理 -->
          <a-select
            v-model:value="form.type"
            :disabled="editMode === 'edit'"
            show-search
          >
            <a-select-option value="EMAIL">
              {{ $t('page.notificationChannel.typeEmail') }}
            </a-select-option>
            <a-select-option value="WEBHOOK">
              {{ $t('page.notificationChannel.typeWebhook') }}
            </a-select-option>
            <a-select-opt-group
              v-for="group in extTypeGroups"
              :key="group.group"
              :label="group.label"
            >
              <a-select-option
                v-for="opt in group.options"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </a-select-option>
            </a-select-opt-group>
          </a-select>
        </a-form-item>
        <!-- 三类渠道各自的配置项分组显示：表单里同时摆几组，管理员要自己猜哪一组生效。
             编辑态 type 不可改，所以几组不会在同一个会话里来回切。 -->
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
        <template v-else-if="isExtType(form.type)">
          <div
            v-if="extSchemaOf(form.type)?.targetLabel"
            class="mb-2 text-xs text-muted-foreground"
          >
            {{ $t('page.notificationChannel.ext.targetLabel', { label: extSchemaOf(form.type)?.targetLabel }) }}
          </div>
          <a-form-item
            v-for="field in extSchemaOf(form.type)?.fields ?? []"
            :key="field.name"
            :label="field.label ?? field.name"
            :required="field.required"
          >
            <a-input-number
              v-if="field.type === 'number'"
              v-model:value="form.settings[field.name!]"
              :min="0"
              class="w-full"
            />
            <a-select
              v-else-if="field.type === 'select'"
              v-model:value="form.settings[field.name!]"
              allow-clear
            >
              <a-select-option
                v-for="opt in field.options ?? []"
                :key="opt"
                :value="opt"
              >
                {{ opt || $t('page.notificationChannel.ext.optionDefault') }}
              </a-select-option>
            </a-select>
            <a-textarea
              v-else-if="field.type === 'textarea'"
              v-model:value="form.settings[field.name!]"
              :rows="3"
              :placeholder="field.placeholder"
            />
            <a-input-password
              v-else-if="field.type === 'password' || field.sensitive"
              v-model:value="form.settings[field.name!]"
              :placeholder="
                editMode === 'edit'
                  ? $t('page.notificationChannel.ext.secretKeepHint')
                  : field.placeholder
              "
            />
            <a-input v-else v-model:value="form.settings[field.name!]" :placeholder="field.placeholder" />
            <div v-if="field.hint" class="text-xs text-muted-foreground">{{ field.hint }}</div>
          </a-form-item>
        </template>
        <template v-else-if="form.type === 'SMS'">
          <a-form-item
            :label="$t('page.notificationChannel.smsProvider')"
            :help="$t('page.notificationChannel.smsProviderHint')"
            required
          >
            <a-select v-model:value="form.smsProvider">
              <a-select-option value="ALIYUN">
                {{ $t('page.notificationChannel.providerAliyun') }}
              </a-select-option>
              <a-select-option value="TENCENT">
                {{ $t('page.notificationChannel.providerTencent') }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item
            :label="$t('page.notificationChannel.smsAccessKeyId')"
            required
          >
            <a-input v-model:value="form.smsAccessKeyId" />
          </a-form-item>
          <a-form-item
            :label="$t('page.notificationChannel.smsAccessSecret')"
            :help="$t('page.notificationChannel.smsAccessSecretHint')"
            :required="editMode === 'create'"
          >
            <a-input-password
              v-model:value="form.smsAccessSecret"
              :placeholder="
                editMode === 'edit'
                  ? $t('page.notificationChannel.smsAccessSecretKeepHint')
                  : $t('page.notificationChannel.smsAccessSecretPlaceholder')
              "
            />
          </a-form-item>
          <a-form-item
            :label="$t('page.notificationChannel.smsSignName')"
            :help="$t('page.notificationChannel.smsSignNameHint')"
            required
          >
            <a-input v-model:value="form.smsSignName" />
          </a-form-item>
          <a-form-item
            :label="$t('page.notificationChannel.smsTemplateCode')"
            :help="$t('page.notificationChannel.smsTemplateCodeHint')"
            required
          >
            <a-input
              v-model:value="form.smsTemplateCode"
              :placeholder="$t('page.notificationChannel.smsTemplateCodePlaceholder')"
            />
          </a-form-item>
          <!-- 腾讯云两项必填、阿里云留空：必留必填差在 help 里说清，
               校验在保存时按服务商拦（见 smsMissingField），这里不摆红星。 -->
          <a-form-item
            :label="$t('page.notificationChannel.smsAppId')"
            :help="$t('page.notificationChannel.smsAppIdHint')"
          >
            <a-input v-model:value="form.smsAppId" placeholder="1400000000" />
          </a-form-item>
          <a-form-item
            :label="$t('page.notificationChannel.smsRegion')"
            :help="$t('page.notificationChannel.smsRegionHint')"
          >
            <a-input v-model:value="form.smsRegion" placeholder="ap-guangzhou" />
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
          <a-form-item :label="$t('page.notificationChannel.smtpFromName')">
            <a-input v-model:value="form.smtpFromName" placeholder="GoWind" />
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
          <template #footer>
        <a-space style="display: flex; justify-content: flex-end">
          <a-button @click="editOpen = false">{{ $t('ui.button.cancel') }}</a-button>
          <a-button type="primary" :loading="editSaving" @click="handleSave">
            {{ $t('ui.button.ok') }}
          </a-button>
        </a-space>
      </template>
</a-drawer>

    <!-- 测试发送 -->
    <a-modal
      v-model:open="testOpen"
      :title="
        testTarget?.type === 'SMS'
          ? $t('page.notificationChannel.testSendTitleSms', { name: testTarget?.name || '' })
          : $t('page.notificationChannel.testSendTitle', { name: testTarget?.name || '' })
      "
      :confirm-loading="testSaving"
      @ok="handleTestSend"
    >
      <a-input
        v-model:value="testRecipient"
        class="mt-4"
        :placeholder="
          testTarget?.type === 'SMS'
            ? $t('page.notificationChannel.testRecipientPhone')
            : $t('page.notificationChannel.testRecipient')
        "
      />
    </a-modal>
  </Page>
</template>
