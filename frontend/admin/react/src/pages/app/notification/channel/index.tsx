import { useMemo, useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import {
  ModalForm,
  DrawerForm,
  ProFormText,
  ProFormDigit,
  ProFormSelect,
  ProFormTextArea,
  ProFormSwitch,
  ProFormDependency,
} from '@ant-design/pro-components';
import { Button, Tag, App } from 'antd';
import type { ProFormItemProps } from '@ant-design/pro-components';
import { EditOutlined, DeleteOutlined, PlusOutlined, SendOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type {
  notification_channelservicev1_NotificationChannel as NotificationChannel,
  notification_channelservicev1_SendTestRequest,
  notification_channelservicev1_CreateNotificationChannelRequest,
  notification_channelservicev1_UpdateNotificationChannelRequest,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import {
  fetchListNotificationChannels,
  useDeleteNotificationChannel,
  useSendTest,
  useUpdateNotificationChannel,
  useCreateNotificationChannel,
} from '@/api/hooks/notification-channel';
import { useListChannelSchemas, type ChannelSchema } from '@/api/hooks/notification-channel-schema';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';

/**
 * 渠道类型 → 文案/颜色。列必须按 record.type 渲染：一个渠道类型对应一列自己的配置
 * （SMTP 那几列对 WEBHOOK 没有意义，反之 webhookUrl 对邮件渠道为空），混着显示会读成"没配"。
 */
const CHANNEL_TYPE_LABEL_KEY: Partial<Record<NonNullable<NotificationChannel['type']>, string>> = {
  EMAIL: 'typeEmail',
  WEBHOOK: 'typeWebhook',
  SMS: 'typeSms',
};

// EXT_* 扩展渠道的展示名/分组来自后端 schema（settings 列那批），不在 i18n 里逐个维护。
// 判定"是不是扩展渠道"的唯一真理源 = schema 清单（hook 数据），组件内用 schemaByType。

const CHANNEL_TYPE_COLOR: Partial<Record<NonNullable<NotificationChannel['type']>, string>> = {
  EMAIL: 'blue',
  WEBHOOK: 'purple',
  SMS: 'cyan',
};

/** settings 读回显 JSON 容错解析：坏数据返回 undefined，动态表单按全空处理。 */
function safeParseSettings(raw: string): Record<string, unknown> | undefined {
  try {
    const parsed = JSON.parse(raw);
    return typeof parsed === 'object' && parsed !== null ? (parsed as Record<string, unknown>) : undefined;
  } catch {
    return undefined;
  }
}

/** EXT_* 扩展渠道的 Tag 统一色。 */
const EXT_TAG_COLOR = 'geekblue';

/**
 * 短信服务商 → 文案。列值就是 proto SmsProvider 成员名的同一组字符串（枚举按名字配对）。
 */
const SMS_PROVIDER_LABEL_KEY: Record<
  NonNullable<NotificationChannel['smsProvider']>,
  string
> = {
  ALIYUN: 'providerAliyun',
  TENCENT: 'providerTencent',
};

/**
 * 签名风格 → 文案。列值就是后端 webhook_style.go 里的同一组字符串（枚举按名字配对），
 * 所以这里既不换算也不加前缀。
 */
const SIGN_STYLE_LABEL_KEY: Record<
  NonNullable<NotificationChannel['webhookSignStyle']>,
  string
> = {
  CUSTOM: 'signStyleCustom',
  NONE: 'signStyleNone',
  DINGTALK: 'signStyleDingtalk',
  FEISHU: 'signStyleFeishu',
  WECOM: 'signStyleWecom',
};

// 这一列可空，读到空值时按 CUSTOM 显示——它的实际行为就是 CUSTOM。
// （本机实测：PG 加列带 DEFAULT，存量行已被回填成 'CUSTOM'，所以空值只出现在直接写 SQL 置 NULL 的行。）
// EMAIL 行没有这一列，返回 undefined 由列渲染 '-'。
const signStyleKeyOf = (record: NotificationChannel): string | undefined =>
  record.type === 'WEBHOOK'
    ? SIGN_STYLE_LABEL_KEY[record.webhookSignStyle ?? 'CUSTOM']
    : undefined;

/**
 * 载荷模板可引用的占位符清单，作为词条的 `vars` 插值值传进去，而不是把花括号写进词条：
 * 一是这些名字属于接口契约（三种语言都不该翻译），二是 i18next 会把词条里的 `{{x}}`
 * 当成插值语法吃掉，写进去只会渲染成一串空白。
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

/**
 * 通知渠道管理页面（平台级配置）
 * 两类渠道：EMAIL（SMTP 账号）与 WEBHOOK（回调地址 + 可选签名密钥）。
 */
const NotificationChannelManagement = () => {
  const { t } = useTranslation('notification-channel');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // 扩展渠道 schema：展示名/分组/动态表单的唯一数据源
  const schemasQuery = useListChannelSchemas();
  const schemas = useMemo(() => schemasQuery.data?.items ?? [], [schemasQuery.data]);
  const schemaByType = useMemo(
    () => new Map<string, ChannelSchema>(schemas.map((s) => [String(s.type), s])),
    [schemas],
  );
  const isExtType = (t?: string) => !!t && schemaByType.has(t);
  // 类型下拉按 schema 分组（EXT 部分），老三样固定在最前
  const extTypeOptions = useMemo(() => {
    const groups = new Map<string, { label: string; value: string }[]>();
    for (const s of schemas) {
      const g = String(s.group || 'EXT');
      if (!groups.has(g)) groups.set(g, []);
      groups.get(g)!.push({ label: String(s.label), value: String(s.type) });
    }
    return [...groups.entries()].map(([group, options]) => ({ group, label: t(`extGroup.${group}`), options }));
  }, [schemas, t]);

  const [drawerOpen, setDrawerOpen] = useState(false);
  // 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
  const { guardClose, handleValuesChange } = useDrawerCloseGuard(drawerOpen, () => setDrawerOpen(false));
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<NotificationChannel | undefined>();
  const [testTarget, setTestTarget] = useState<NotificationChannel | undefined>();

  const refresh = () => {
    actionRef.current?.reload();
    queryClient.invalidateQueries({ queryKey: ['listNotificationChannels'] });
  };

  const deleteMutation = useDeleteNotificationChannel({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      refresh();
    },
  });

  const createMutation = useCreateNotificationChannel();
  const updateMutation = useUpdateNotificationChannel();
  const testMutation = useSendTest();

  const handleDelete = async (record: NotificationChannel) => {
    if (!record.id) return;
    try {
      await deleteMutation.mutateAsync({ id: record.id });
    } catch (error: any) {
      message.error(error.message || t('deleteFailed'));
    }
  };

  const handleSubmit = async (values: Record<string, any>) => {
    // 三个密钥是请求级字段（不进 data）：读视图只有 hasPassword / hasWebhookSecret / hasSmsAccessSecret 布尔
    const { password, webhookSecret, smsAccessSecret, settings, ...data } = values;
    // 拆分选项合成键：SMS@ALIYUN / SMS@TENCENT → type=SMS + smsProvider
    if (String(data.type).startsWith('SMS@')) {
      data.smsProvider = String(data.type).split('@')[1];
      data.type = 'SMS';
    }
    // 扩展渠道：动态表单的 settings 对象序列化为 JSON 明文上送（sensitive 留空 = 保留已存值）
    if (isExtType(data.type)) {
      data.settings = settings ? JSON.stringify(settings) : '';
    } else {
      delete data.settings;
    }
    try {
      if (drawerMode === 'create') {
        const req: notification_channelservicev1_CreateNotificationChannelRequest = {
          data,
          password: password || undefined,
          webhookSecret: webhookSecret || undefined,
          smsAccessSecret: smsAccessSecret || undefined,
        };
        await createMutation.mutateAsync(req);
        message.success(t('createSuccess'));
      } else if (selected?.id) {
        const req: notification_channelservicev1_UpdateNotificationChannelRequest = {
          id: selected.id,
          data,
          password: password || undefined,
          webhookSecret: webhookSecret || undefined,
          smsAccessSecret: smsAccessSecret || undefined,
          updateMask: isExtType(data.type)
            ? 'name,type,settings,enabled,remark'
            : 'name,type,smtpHost,smtpPort,smtpUsername,smtpFrom,smtpFromName,smtpTls,webhookUrl,webhookSignStyle,webhookPayloadTemplate,smsProvider,smsAccessKeyId,smsSignName,smsTemplateCode,smsAppId,smsRegion,enabled,remark',
        };
        await updateMutation.mutateAsync(req);
        message.success(t('updateSuccess'));
      }
      setDrawerOpen(false);
      refresh();
      return true;
    } catch (error: any) {
      message.error(error.message || t('saveFailed'));
      return false;
    }
  };

  const handleTestSend = async (values: { recipient: string }) => {
    if (!testTarget?.id) return false;
    const req: notification_channelservicev1_SendTestRequest = {
      id: testTarget.id,
      recipient: values.recipient,
    };
    try {
      await testMutation.mutateAsync(req);
      message.success(t('testSendSuccess'));
      setTestTarget(undefined);
      return true;
    } catch (error: any) {
      message.error(error.message || t('testSendFailed'));
      return false;
    }
  };

  const columns: ProColumns<NotificationChannel>[] = [
    {
      title: t('name'),
      dataIndex: 'name',
      width: 150,
      ellipsis: true,
    },
    {
      title: t('type'),
      dataIndex: 'type',
      width: 100,
      render: (_, record) =>
        record.type ? (
          <Tag color={CHANNEL_TYPE_COLOR[record.type] ?? EXT_TAG_COLOR}>
            {schemaByType.get(record.type)?.label ?? t(CHANNEL_TYPE_LABEL_KEY[record.type] ?? record.type)}
          </Tag>
        ) : (
          '-'
        ),
    },
    {
      title: t('smtpHost'),
      dataIndex: 'smtpHost',
      width: 160,
      ellipsis: true,
      render: (_, record) => record.smtpHost || '-',
    },
    {
      title: t('smtpPort'),
      dataIndex: 'smtpPort',
      width: 90,
      render: (_, record) => record.smtpPort ?? '-',
    },
    {
      title: t('smtpFrom'),
      dataIndex: 'smtpFrom',
      width: 180,
      ellipsis: true,
      render: (_, record) => record.smtpFrom || '-',
    },
    {
      title: t('smtpTls'),
      dataIndex: 'smtpTls',
      width: 110,
      render: (_, record) => {
        const map: Record<string, string> = {
          NONE: t('tlsNone'),
          START_TLS: t('tlsStartTls'),
          SSL: t('tlsSsl'),
        };
        return map[record.smtpTls || ''] || record.smtpTls || '-';
      },
    },
    {
      title: t('webhookUrl'),
      dataIndex: 'webhookUrl',
      width: 200,
      ellipsis: true,
      render: (_, record) => record.webhookUrl || '-',
    },
    {
      title: t('webhookSignStyle'),
      dataIndex: 'webhookSignStyle',
      width: 140,
      render: (_, record) => {
        const key = signStyleKeyOf(record);
        return key ? <Tag>{t(key)}</Tag> : '-';
      },
    },
    {
      title: t('smsProvider'),
      dataIndex: 'smsProvider',
      width: 110,
      render: (_, record) => {
        if (record.type !== 'SMS') return '-';
        const key = record.smsProvider ? SMS_PROVIDER_LABEL_KEY[record.smsProvider] : undefined;
        return key ? <Tag>{t(key)}</Tag> : '-';
      },
    },
    {
      title: t('smsSignName'),
      dataIndex: 'smsSignName',
      width: 120,
      ellipsis: true,
      render: (_, record) => (record.type === 'SMS' ? record.smsSignName || '-' : '-'),
    },
    {
      title: t('hasPassword'),
      dataIndex: 'hasPassword',
      width: 100,
      render: (_, record) =>
        record.hasPassword ? (
          <Tag color="green">{t('passwordSet')}</Tag>
        ) : (
          <Tag>{t('passwordNotSet')}</Tag>
        ),
    },
    {
      title: t('hasWebhookSecret'),
      dataIndex: 'hasWebhookSecret',
      width: 100,
      render: (_, record) =>
        record.hasWebhookSecret ? (
          <Tag color="green">{t('passwordSet')}</Tag>
        ) : (
          <Tag>{t('passwordNotSet')}</Tag>
        ),
    },
    {
      title: t('hasSmsAccessSecret'),
      dataIndex: 'hasSmsAccessSecret',
      width: 100,
      render: (_, record) =>
        record.hasSmsAccessSecret ? (
          <Tag color="green">{t('passwordSet')}</Tag>
        ) : (
          <Tag>{t('passwordNotSet')}</Tag>
        ),
    },
    {
      title: t('enabled'),
      dataIndex: 'enabled',
      width: 90,
      render: (_, record) =>
        record.enabled ? <Tag color="success">{t('enabledOn')}</Tag> : <Tag>{t('enabledOff')}</Tag>,
    },
    {
      title: t('remark'),
      dataIndex: 'remark',
      width: 140,
      ellipsis: true,
      hideInTable: window.innerWidth < 1600,
      render: (_, record) => record.remark || '-',
    },
    {
      title: t('actions'),
      valueType: 'option',
      // 编辑渠道+测试发送+删除 三个图标+文字按钮实测需 284px，250 会被截断
      width: 290,
      fixed: 'right',
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'edit',
              label: t('edit'),
              icon: <EditOutlined />,
              onClick: () => {
                setDrawerMode('edit');
                setSelected(record);
                setDrawerOpen(true);
              },
            },
            {
              // 测试发送对"有收件目标语义"的渠道有意义：EMAIL 测 SMTP、SMS 发真验证码、
              // EXT_* 按 schema 发一条测试消息；WEBHOOK 的端到端验证在路由规则页。
              key: 'test',
              label: t('testSend'),
              icon: <SendOutlined />,
              hidden: !(record.type === 'EMAIL' || record.type === 'SMS' || isExtType(record.type)),
              onClick: () => setTestTarget(record),
            },
            {
              key: 'delete',
              label: t('delete'),
              icon: <DeleteOutlined />,
              danger: true,
              confirm: { title: t('deleteConfirm') },
              onClick: () => handleDelete(record),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ListTable<NotificationChannel>
          actionRef={actionRef}
          columns={columns}
          request={async (params) => {
            const query = new PaginationQuery({
              paging: { page: params.current || 1, pageSize: params.pageSize || 20 },
              formValues: Object.fromEntries(
                Object.entries(params).filter(([key]) => !['current', 'pageSize'].includes(key)),
              ),
            });
            const response = await fetchListNotificationChannels(query);
            return { data: response.items || [], total: Number(response.total ?? 0), success: true };
          }}
          rowKey="id"
          search={false}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
          }}
          options={{ density: true, fullScreen: true, setting: { settingIcon: <ColumnSlidersIcon /> }, reload: true }}
          toolBarRender={() => [
            <TableExportButton key="export" fetcher={fetchListNotificationChannels} columns={columns} filename="notification-channels" />,
            <Button
              key="create"
              color="primary"
              variant="outlined"
              icon={<PlusOutlined />}
              onClick={() => {
                setDrawerMode('create');
                setSelected(undefined);
                setDrawerOpen(true);
              }}
            >
              {t('common:button.add')}
            </Button>,
          ]}
          size="middle"
          bordered
          scroll={{ y: tableScrollY, x: 2110 }}
        />
      </div>

      {/* 新建/编辑用抽屉不用弹窗：扩展渠道的 settings 动态表单字段多（6~8 个），
          弹窗高度挤不下；抽屉纵向空间足、与 ele 端渠道抽屉同形。 */}
      <DrawerForm
        title={drawerMode === 'create' ? t('common:modal.create', { moduleName: t('title') }) : t('common:modal.update', { moduleName: t('title') })}
        open={drawerOpen}
        onOpenChange={(v) => {
          // 关闭一律走 drawerProps.onClose 守卫；这里只放行打开
          if (v) setDrawerOpen(true);
        }}
        onValuesChange={handleValuesChange}
        drawerProps={{ destroyOnHidden: true, size: 600, onClose: guardClose }}
        submitTimeout={3000}
        onFinish={handleSubmit}
        initialValues={
          drawerMode === 'create'
            ? { type: 'EMAIL', smtpTls: 'START_TLS', smtpPort: 587, webhookSignStyle: 'CUSTOM', smsProvider: 'ALIYUN', enabled: true }
            : {
                ...selected,
                password: undefined,
                webhookSecret: undefined,
                smsAccessSecret: undefined,
                // 同列表列：该列为空时按 CUSTOM 摆进表单，否则这一项是空的、改不动
                webhookSignStyle: selected?.webhookSignStyle ?? 'CUSTOM',
                // SMS 行回显成拆分选项的合成键（SMS@ALIYUN / SMS@TENCENT），提交时再拆回
                type:
                  selected?.type === 'SMS'
                    ? `SMS@${selected.smsProvider ?? 'ALIYUN'}`
                    : selected?.type,
                // settings 读回显是脱敏 JSON 串：parse 成对象供动态表单回填
                // （sensitive 字段值为空串=占位，留空提交即保留已存密文）
                settings: selected?.settings ? safeParseSettings(selected.settings) : undefined,
              }
        }
        layout="horizontal"
        labelCol={{ flex: '0 0 120px' }}
        wrapperCol={{ flex: 'auto' }}
      >
        <ProFormText
          name="name"
          label={t('name')}
          rules={[{ required: true, message: t('requiredName') }]}
        />
        <ProFormSelect
          name="type"
          label={t('type')}
          options={[
            { label: t('typeEmail'), value: 'EMAIL' },
            { label: t('typeWebhook'), value: 'WEBHOOK' },
            // 短信 / 消息 分组：阿里云/腾讯云拆成两个一级选项（合成键 SMS@provider，
            // 提交/回显时与 type=SMS + smsProvider 互转），与扩展短信渠道并列
            {
              label: t('extGroup.SMS'),
              title: t('extGroup.SMS'),
              options: [
                { label: t('smsAliyun'), value: 'SMS@ALIYUN' },
                { label: t('smsTencent'), value: 'SMS@TENCENT' },
                ...(extTypeOptions.find((g) => g.group === 'SMS')?.options ?? []),
              ],
            },
            ...extTypeOptions
              .filter((g) => g.group !== 'SMS')
              .map((g) => ({ label: g.label, title: g.label, options: g.options })),
          ]}
          disabled={drawerMode === 'edit'}
          showSearch
          rules={[{ required: true }]}
        />
        {/* 三类渠道各自的配置项分组显示：表单里同时摆几组，管理员要自己猜哪一组生效。
            编辑态 type 不可改（改类型等于换一条渠道实现），所以几组不会在同一个会话里来回切。 */}
        <ProFormDependency name={['type']}>
          {({ type }) =>
            type === 'WEBHOOK' ? (
              <>
                <ProFormText
                  name="webhookUrl"
                  label={t('webhookUrl')}
                  placeholder="https://example.com/hooks/notification"
                  tooltip={t('webhookUrlHint')}
                  rules={
                    drawerMode === 'create'
                      ? [{ required: true, message: t('requiredWebhookUrl') }]
                      : []
                  }
                />
                <ProFormSelect
                  name="webhookSignStyle"
                  label={t('webhookSignStyle')}
                  tooltip={t('webhookSignStyleHint')}
                  options={(
                    ['CUSTOM', 'NONE', 'DINGTALK', 'FEISHU', 'WECOM'] as const
                  ).map((value) => ({ value, label: t(SIGN_STYLE_LABEL_KEY[value]) }))}
                  rules={[{ required: true, message: t('requiredSignStyle') }]}
                />
                {/* 留空 = 用该风格的内置默认正文（内置形状由后端 webhook_style.go 持有，
                    这里不复制一份，否则两处会漂）。 */}
                <ProFormTextArea
                  name="webhookPayloadTemplate"
                  label={t('webhookPayloadTemplate')}
                  tooltip={t('webhookPayloadTemplateHint', { vars: WEBHOOK_TEMPLATE_VARS })}
                  placeholder='{"msgtype":"text","text":{"content":"{{title}}"}}'
                  fieldProps={{ rows: 3, style: { fontFamily: 'var(--font-mono, monospace)' } }}
                />
                <ProFormText.Password
                  name="webhookSecret"
                  label={t('webhookSecret')}
                  placeholder={
                    drawerMode === 'edit' ? t('webhookSecretKeepHint') : t('webhookSecretPlaceholder')
                  }
                  tooltip={t('webhookSecretHint')}
                />
              </>
            ) : isExtType(type) ? (
              <SettingsFormFields schema={schemaByType.get(String(type))} />
            ) : String(type).startsWith('SMS') ? (
              <>
                <ProFormSelect
                  name="smsProvider"
                  label={t('smsProvider')}
                  tooltip={t('smsProviderHint')}
                  options={[
                    { label: t('providerAliyun'), value: 'ALIYUN' },
                    { label: t('providerTencent'), value: 'TENCENT' },
                  ]}
                  rules={[{ required: true, message: t('requiredSmsProvider') }]}
                />
                <ProFormText
                  name="smsAccessKeyId"
                  label={t('smsAccessKeyId')}
                  rules={[{ required: true, message: t('requiredSmsAccessKeyId') }]}
                />
                <ProFormText.Password
                  name="smsAccessSecret"
                  label={t('smsAccessSecret')}
                  placeholder={
                    drawerMode === 'edit' ? t('smsAccessSecretKeepHint') : t('smsAccessSecretPlaceholder')
                  }
                  tooltip={t('smsAccessSecretHint')}
                  rules={
                    drawerMode === 'create' ? [{ required: true, message: t('requiredSmsAccessSecret') }] : []
                  }
                />
                <ProFormText
                  name="smsSignName"
                  label={t('smsSignName')}
                  tooltip={t('smsSignNameHint')}
                  rules={[{ required: true, message: t('requiredSmsSignName') }]}
                />
                <ProFormText
                  name="smsTemplateCode"
                  label={t('smsTemplateCode')}
                  tooltip={t('smsTemplateCodeHint')}
                  placeholder={t('smsTemplateCodePlaceholder')}
                  rules={[{ required: true, message: t('requiredSmsTemplateCode') }]}
                />
                {/* 腾讯云两项必填、阿里云留空：必留必填差在 tooltip 里说清，
                    不按服务商切 required——否则选错服务商时表单先于后端把人拦在模糊的 red 星号上。 */}
                <ProFormText
                  name="smsAppId"
                  label={t('smsAppId')}
                  tooltip={t('smsAppIdHint')}
                  placeholder="1400000000"
                />
                <ProFormText
                  name="smsRegion"
                  label={t('smsRegion')}
                  tooltip={t('smsRegionHint')}
                  placeholder="ap-guangzhou"
                />
              </>
            ) : (
              <>
                <ProFormText name="smtpHost" label={t('smtpHost')} placeholder="smtp.example.com" />
                <ProFormDigit
                  name="smtpPort"
                  label={t('smtpPort')}
                  min={1}
                  max={65535}
                  fieldProps={{ precision: 0 }}
                />
                <ProFormText name="smtpUsername" label={t('smtpUsername')} />
                <ProFormText.Password
                  name="password"
                  label={t('password')}
                  placeholder={drawerMode === 'edit' ? t('passwordKeepHint') : t('passwordPlaceholder')}
                  rules={
                    drawerMode === 'create' ? [{ required: true, message: t('requiredPassword') }] : []
                  }
                />
                <ProFormText name="smtpFrom" label={t('smtpFrom')} placeholder="noreply@example.com" />
                <ProFormText name="smtpFromName" label={t('smtpFromName')} placeholder="GoWind" />
                <ProFormSelect
                  name="smtpTls"
                  label={t('smtpTls')}
                  options={[
                    { label: t('tlsNone'), value: 'NONE' },
                    { label: t('tlsStartTls'), value: 'START_TLS' },
                    { label: t('tlsSsl'), value: 'SSL' },
                  ]}
                />
              </>
            )
          }
        </ProFormDependency>
        <ProFormSwitch name="enabled" label={t('enabled')} />
        <ProFormTextArea
          name="remark"
          label={t('remark')}
          fieldProps={{ rows: 2 }}
        />
      </DrawerForm>

      <ModalForm<{ recipient: string }>
        title={
          testTarget?.type === 'SMS'
            ? t('testSendTitleSms', { name: testTarget?.name || '' })
            : t('testSendTitle', { name: testTarget?.name || '' })
        }
        width={440}
        open={!!testTarget}
        onOpenChange={(open) => {
          if (!open) setTestTarget(undefined);
        }}
        modalProps={{ destroyOnHidden: true }}
        submitTimeout={5000}
        onFinish={handleTestSend}
      >
        <ProFormText
          name="recipient"
          label={testTarget?.type === 'SMS' ? t('testRecipientPhone') : t('testRecipient')}
          rules={[
            {
              required: true,
              message:
                testTarget?.type === 'SMS'
                  ? t('requiredRecipientPhone')
                  : t('requiredRecipient'),
            },
            // 校验跟随渠道类型：SMS 收手机号、EMAIL 收邮箱；EXT_* 的目标语义按渠道各不相同
            // （chat_id/topic/设备 Key…），只做非空不做格式——格式错了由服务商报错进台账。
            ...(testTarget?.type === 'SMS'
              ? [{ pattern: /^\+?\d[\d\s-]{4,19}$/, message: t('invalidPhone') }]
              : testTarget?.type === 'EMAIL'
                ? [{ type: 'email' as const, message: t('invalidEmail') }]
                : []),
          ]}
          placeholder={testTarget?.type === 'SMS' ? '13800138000' : 'you@example.com'}
        />
      </ModalForm>
    </ContentContainer>
  );
};

export default NotificationChannelManagement;


/** schema 字段类型 → 动态表单（settings 命名空间：提交时 values.settings = {name: value}）。 */
function SettingsFormFields({ schema }: { schema: ChannelSchema | undefined }) {
  const { t } = useTranslation('notification-channel');
  if (!schema) return null;
  return (
    <>
      {schema.targetLabel && (
        <div style={{ color: 'var(--ant-color-text-tertiary, #aaa)', fontSize: 12, marginBottom: 8 }}>
          {t('ext.targetLabel', { label: schema.targetLabel })}
        </div>
      )}
      {(schema.fields ?? []).map((f) => {
        const name = ['settings', String(f.name)];
        const label = String(f.label ?? f.name);
        const rules = f.required ? [{ required: true, message: t('ext.required', { label }) }] : [];
        const common = { name, label, rules, tooltip: f.hint || undefined, extra: undefined };
        switch (f.type) {
          case 'textarea':
            return (
              <ProFormTextArea
                key={f.name}
                {...common}
                fieldProps={{ rows: 3, placeholder: f.placeholder }}
              />
            );
          case 'number':
            return (
              <ProFormDigit
                key={f.name}
                {...common}
                fieldProps={{ precision: 0, placeholder: f.placeholder }}
              />
            );
          case 'password':
            return (
              <ProFormText.Password
                key={f.name}
                {...(common as ProFormItemProps)}
                placeholder={f.placeholder || t('ext.secretKeepHint')}
              />
            );
          case 'select':
            return (
              <ProFormSelect
                key={f.name}
                {...common}
                options={(f.options ?? []).map((o) => ({ label: o === '' ? t('ext.optionDefault') : o, value: o }))}
                fieldProps={{ allowClear: true }}
              />
            );
          default:
            return <ProFormText key={f.name} {...common} placeholder={f.placeholder} />;
        }
      })}
    </>
  );
}
