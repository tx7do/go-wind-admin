import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import {
  ProTable,
  ModalForm,
  ProFormText,
  ProFormDigit,
  ProFormSelect,
  ProFormTextArea,
  ProFormSwitch,
  ProFormDependency,
} from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined, SendOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type {
  notification_channelservicev1_NotificationChannel as NotificationChannel,
  notification_channelservicev1_SendTestEmailRequest,
  notification_channelservicev1_CreateNotificationChannelRequest,
  notification_channelservicev1_UpdateNotificationChannelRequest,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import {
  fetchListNotificationChannels,
  useDeleteNotificationChannel,
  useSendTestEmail,
  useUpdateNotificationChannel,
  useCreateNotificationChannel,
} from '@/api/hooks/notification-channel';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';

/**
 * 渠道类型 → 文案/颜色。列必须按 record.type 渲染：一个渠道类型对应一列自己的配置
 * （SMTP 那几列对 WEBHOOK 没有意义，反之 webhookUrl 对邮件渠道为空），混着显示会读成"没配"。
 */
const CHANNEL_TYPE_LABEL_KEY: Record<
  NonNullable<NotificationChannel['type']>,
  string
> = {
  EMAIL: 'typeEmail',
  WEBHOOK: 'typeWebhook',
};

const CHANNEL_TYPE_COLOR: Record<NonNullable<NotificationChannel['type']>, string> = {
  EMAIL: 'blue',
  WEBHOOK: 'purple',
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

  const [drawerOpen, setDrawerOpen] = useState(false);
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
  const testMutation = useSendTestEmail();

  const handleDelete = async (record: NotificationChannel) => {
    if (!record.id) return;
    try {
      await deleteMutation.mutateAsync({ id: record.id });
    } catch (error: any) {
      message.error(error.message || t('deleteFailed'));
    }
  };

  const handleSubmit = async (values: Record<string, any>) => {
    // 两处密钥是请求级字段（不进 data）：读视图只有 hasPassword / hasWebhookSecret 两个布尔
    const { password, webhookSecret, ...data } = values;
    try {
      if (drawerMode === 'create') {
        const req: notification_channelservicev1_CreateNotificationChannelRequest = {
          data,
          password: password || undefined,
          webhookSecret: webhookSecret || undefined,
        };
        await createMutation.mutateAsync(req);
        message.success(t('createSuccess'));
      } else if (selected?.id) {
        const req: notification_channelservicev1_UpdateNotificationChannelRequest = {
          id: selected.id,
          data,
          password: password || undefined,
          webhookSecret: webhookSecret || undefined,
          updateMask:
            'name,type,smtpHost,smtpPort,smtpUsername,smtpFrom,smtpTls,webhookUrl,webhookSignStyle,webhookPayloadTemplate,enabled,remark',
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
    const req: notification_channelservicev1_SendTestEmailRequest = {
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
          <Tag color={CHANNEL_TYPE_COLOR[record.type]}>
            {t(CHANNEL_TYPE_LABEL_KEY[record.type])}
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
      width: 220,
      fixed: 'right',
      render: (_, record) => [
        <Button
          key="edit"
          type="link"
          size="small"
          icon={<EditOutlined />}
          onClick={() => {
            setDrawerMode('edit');
            setSelected(record);
            setDrawerOpen(true);
          }}
        >
          {t('edit')}
        </Button>,
        // 测试发送只对 EMAIL 有意义：它测的就是这一个 SMTP 账号能否握手发信。
        // WEBHOOK 的"当场试一次"在路由规则页（那里测的是事件 → 渠道 → 台账整条链）。
        record.type === 'EMAIL' && (
          <Button key="test" type="link" size="small" icon={<SendOutlined />} onClick={() => setTestTarget(record)}>
            {t('testSend')}
          </Button>
        ),
        <Popconfirm key="delete" title={t('deleteConfirm')} onConfirm={() => handleDelete(record)}>
          <Button danger type="link" size="small" icon={<DeleteOutlined />}>
            {t('delete')}
          </Button>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ProTable<NotificationChannel>
          actionRef={actionRef}
          columns={columns}
          request={async (params) => {
            try {
              const query = new PaginationQuery({
                paging: { page: params.current || 1, pageSize: params.pageSize || 20 },
                formValues: Object.fromEntries(
                  Object.entries(params).filter(([key]) => !['current', 'pageSize'].includes(key)),
                ),
              });
              const response = await fetchListNotificationChannels(query);
              return { data: response.items || [], total: response.total || 0, success: true };
            } catch (error: any) {
              message.error(error.message || t('fetchFailed'));
              return { data: [], total: 0, success: false };
            }
          }}
          rowKey="id"
          search={false}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
          }}
          options={{ density: true, fullScreen: true, setting: true, reload: true }}
          toolBarRender={() => [
            <TableExportButton key="export" fetcher={fetchListNotificationChannels} columns={columns} filename="notification-channels" />,
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setDrawerMode('create');
                setSelected(undefined);
                setDrawerOpen(true);
              }}
            >
              {t('create')}
            </Button>,
          ]}
          size="middle"
          bordered
          scroll={{ y: tableScrollY, x: 1780 }}
        />
      </div>

      <ModalForm
        title={drawerMode === 'create' ? t('create') : t('edit')}
        width={560}
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        modalProps={{ destroyOnHidden: true, mask: { closable: false } }}
        submitTimeout={3000}
        onFinish={handleSubmit}
        initialValues={
          drawerMode === 'create'
            ? { type: 'EMAIL', smtpTls: 'START_TLS', smtpPort: 587, webhookSignStyle: 'CUSTOM', enabled: true }
            : {
                ...selected,
                password: undefined,
                webhookSecret: undefined,
                // 同列表列：该列为空时按 CUSTOM 摆进表单，否则这一项是空的、改不动
                webhookSignStyle: selected?.webhookSignStyle ?? 'CUSTOM',
              }
        }
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
          ]}
          disabled={drawerMode === 'edit'}
          rules={[{ required: true }]}
        />
        {/* 两类渠道各自的配置项分组显示：表单里同时摆两组，管理员要自己猜哪一组生效。
            编辑态 type 不可改（改类型等于换一条渠道实现），所以两组不会在同一个会话里来回切。 */}
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
        <ProFormTextArea name="remark" label={t('remark')} fieldProps={{ rows: 2 }} />
      </ModalForm>

      <ModalForm<{ recipient: string }>
        title={t('testSendTitle', { name: testTarget?.name || '' })}
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
          label={t('testRecipient')}
          rules={[
            { required: true, message: t('requiredRecipient') },
            { type: 'email', message: t('invalidEmail') },
          ]}
          placeholder="you@example.com"
        />
      </ModalForm>
    </ContentContainer>
  );
};

export default NotificationChannelManagement;
