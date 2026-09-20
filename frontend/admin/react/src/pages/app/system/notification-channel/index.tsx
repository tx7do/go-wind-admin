import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import { ProTable, ModalForm, ProFormText, ProFormDigit, ProFormSelect, ProFormTextArea, ProFormSwitch } from '@ant-design/pro-components';
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
 * 渠道类型 → 文案/颜色。列必须按 record.type 渲染：WEBHOOK 目前没有发送实现（P2 才落地），
 * 建渠道的表单也因此只给 EMAIL 选项，但行数据里有 WEBHOOK 时不能跟着显示"邮件 (SMTP)"。
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
 * 通知渠道管理页面（平台级配置）
 * 一期实现 EMAIL（SMTP）渠道：CRUD + 测试发送。
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
    const { password, ...data } = values;
    try {
      if (drawerMode === 'create') {
        const req: notification_channelservicev1_CreateNotificationChannelRequest = {
          data,
          password: password || undefined,
        };
        await createMutation.mutateAsync(req);
        message.success(t('createSuccess'));
      } else if (selected?.id) {
        const req: notification_channelservicev1_UpdateNotificationChannelRequest = {
          id: selected.id,
          data,
          password: password || undefined,
          updateMask:
            'name,type,smtpHost,smtpPort,smtpUsername,smtpFrom,smtpTls,enabled,remark',
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
        <Button
          key="test"
          type="link"
          size="small"
          icon={<SendOutlined />}
          onClick={() => setTestTarget(record)}
        >
          {t('testSend')}
        </Button>,
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
          scroll={{ y: tableScrollY, x: 1200 }}
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
            ? { type: 'EMAIL', smtpTls: 'START_TLS', smtpPort: 587, enabled: true }
            : { ...selected, password: undefined }
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
          options={[{ label: t('typeEmail'), value: 'EMAIL' }]}
          disabled={drawerMode === 'edit'}
          rules={[{ required: true }]}
        />
        <ProFormText name="smtpHost" label={t('smtpHost')} placeholder="smtp.example.com" />
        <ProFormDigit name="smtpPort" label={t('smtpPort')} min={1} max={65535} fieldProps={{ precision: 0 }} />
        <ProFormText name="smtpUsername" label={t('smtpUsername')} />
        <ProFormText.Password
          name="password"
          label={t('password')}
          placeholder={drawerMode === 'edit' ? t('passwordKeepHint') : t('passwordPlaceholder')}
          rules={drawerMode === 'create' ? [{ required: true, message: t('requiredPassword') }] : []}
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
