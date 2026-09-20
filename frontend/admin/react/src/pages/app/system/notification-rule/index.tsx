import { useRef, useState } from 'react';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ProTable,
  ModalForm,
  ProFormText,
  ProFormTextArea,
  ProFormSelect,
  ProFormSwitch,
} from '@ant-design/pro-components';
import { App, Button, Popconfirm, Tag, Tooltip } from 'antd';
import { DeleteOutlined, EditOutlined, PlusOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type {
  notificationservicev1_Channel,
  notificationservicev1_NotificationRule as NotificationRule,
  notificationservicev1_CreateNotificationRuleRequest,
  notificationservicev1_UpdateNotificationRuleRequest,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import {
  fetchListNotificationRules,
  useCreateNotificationRule,
  useDeleteNotificationRule,
  useTestDispatchNotification,
  useUpdateNotificationRule,
} from '@/api/hooks/notification-rule';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';

/**
 * 通知路由规则管理（事件类型 → 渠道 + 派发方式）
 *
 * 这张表是投递时的唯一真相：改完这一行，下一次投递立刻按新路由走（服务端不缓存）。
 * 两个决定来自同一行，所以「渠道」与「异步」必须一起看：异步 = 请求入队即回，
 * 结论由队列回写台账；同步 = 当场拨号，接口的回话就是投递结论。
 *
 * 删掉一行 = 「这个事件不再通知」：服务端不会在下次启动补回来，但那条事件的调用
 * 会拿到「没有启用路由规则」的报错（不静默丢通知）。
 */

const CHANNEL_COLORS: Record<notificationservicev1_Channel, string> = {
  EMAIL: 'blue',
  SMS: 'green',
  WEBHOOK: 'purple',
  INTERNAL: 'cyan',
  CHANNEL_UNSPECIFIED: 'default',
};

const eventTypeOptions = (t: (key: string) => string) =>
  (
    ['PASSWORD_RESET_CODE', 'CONTACT_BIND_CODE', 'CHANNEL_TEST_EMAIL', 'INTERNAL_MESSAGE'] as const
  ).map((value) => ({ value, label: t(`eventTypeMap.${value}`) }));

const channelOptions = (t: (key: string) => string) =>
  (['EMAIL', 'WEBHOOK', 'INTERNAL'] as const).map((value) => ({
    value,
    label: t(`channelMap.${value}`),
  }));

const NotificationRulePage = () => {
  const { t } = useTranslation('notification-rule');
  const actionRef = useRef<ActionType>(null);
  const { message } = App.useApp();
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const [formOpen, setFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<NotificationRule | undefined>();
  const [testTarget, setTestTarget] = useState<NotificationRule | undefined>();

  const refresh = () => actionRef.current?.reload();

  const createMutation = useCreateNotificationRule();
  const updateMutation = useUpdateNotificationRule();
  const deleteMutation = useDeleteNotificationRule();
  const testMutation = useTestDispatchNotification();

  const handleDelete = async (record: NotificationRule) => {
    if (!record.id) return;
    try {
      await deleteMutation.mutateAsync({ id: record.id });
      message.success(t('deleteSuccess'));
      refresh();
    } catch (error: any) {
      // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
      console.error('delete notification rule failed', error);
      message.error(error.message || t('deleteFailed'));
    }
  };

  const handleSubmit = async (values: Record<string, any>) => {
    const data = {
      eventType: values.eventType,
      channel: values.channel,
      isAsync: !!values.isAsync,
      isEnabled: !!values.isEnabled,
      remark: values.remark,
    };
    try {
      if (formMode === 'create') {
        const req: notificationservicev1_CreateNotificationRuleRequest = { data };
        await createMutation.mutateAsync(req);
        message.success(t('createSuccess'));
      } else if (selected?.id) {
        const req: notificationservicev1_UpdateNotificationRuleRequest = {
          id: selected.id,
          data,
          // event_type 刻意不进掩码：改事件类型等于换一条路由，删旧建新才说得清
          updateMask: 'channel,isAsync,isEnabled,remark',
        };
        await updateMutation.mutateAsync(req);
        message.success(t('updateSuccess'));
      }
      setFormOpen(false);
      refresh();
      return true;
    } catch (error: any) {
      console.error('save notification rule failed', error);
      message.error(error.message || t('saveFailed'));
      return false;
    }
  };

  const handleTestDispatch = async (values: { target?: string }) => {
    if (!testTarget?.id) return false;
    try {
      const resp = await testMutation.mutateAsync({
        id: testTarget.id,
        target: values.target || undefined,
      });
      // 异步规则回 SENDING：这次只是「已交给队列」，说「已投递」而不说「已送达」
      message.success(
        resp.status === 'SENDING'
          ? t('testDispatchQueued', { id: resp.deliveryId })
          : t('testDispatchSent', { id: resp.deliveryId }),
      );
      setTestTarget(undefined);
      return true;
    } catch (error: any) {
      console.error('test dispatch failed', error);
      message.error(error.message || t('testDispatchFailed'));
      return false;
    }
  };

  const columns: ProColumns<NotificationRule>[] = [
    {
      title: t('eventType'),
      dataIndex: 'eventType',
      width: 150,
      valueType: 'select',
      fieldProps: { options: eventTypeOptions(t) },
      render: (_, record) =>
        record.eventType ? t(`eventTypeMap.${record.eventType}`) : '-',
    },
    {
      title: t('channel'),
      dataIndex: 'channel',
      width: 110,
      valueType: 'select',
      fieldProps: { options: channelOptions(t) },
      render: (_, record) =>
        record.channel ? (
          <Tag color={CHANNEL_COLORS[record.channel]}>{t(`channelMap.${record.channel}`)}</Tag>
        ) : (
          '-'
        ),
    },
    {
      title: t('isAsync'),
      dataIndex: 'isAsync',
      width: 110,
      tooltip: t('isAsyncHint'),
      render: (_, record) =>
        record.isAsync ? (
          <Tag color="geekblue">{t('isAsyncOn')}</Tag>
        ) : (
          <Tag>{t('isAsyncOff')}</Tag>
        ),
    },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 100,
      tooltip: t('isEnabledHint'),
      render: (_, record) =>
        record.isEnabled ? (
          <Tag color="success">{t('enabledOn')}</Tag>
        ) : (
          <Tag>{t('enabledOff')}</Tag>
        ),
    },
    {
      title: t('remark'),
      dataIndex: 'remark',
      width: 260,
      ellipsis: true,
      render: (_, record) => record.remark || '-',
    },
    {
      title: t('updatedAt'),
      dataIndex: 'updatedAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('actions'),
      valueType: 'option',
      width: 240,
      fixed: 'right',
      render: (_, record) => [
        <Button
          key="edit"
          type="link"
          size="small"
          icon={<EditOutlined />}
          onClick={() => {
            setFormMode('edit');
            setSelected(record);
            setFormOpen(true);
          }}
        >
          {t('edit')}
        </Button>,
        // 站内信规则不给测试按钮：它需要一条真实的消息本体，填个收件人 ID 测不出东西
        record.channel !== 'INTERNAL' && (
          <Tooltip key="test-tip" title={t('testDispatchHint')}>
            <Button
              type="link"
              size="small"
              icon={<ThunderboltOutlined />}
              onClick={() => setTestTarget(record)}
            >
              {t('testDispatch')}
            </Button>
          </Tooltip>
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
        <ProTable<NotificationRule>
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
              const response = await fetchListNotificationRules(query);
              return { data: response.items || [], total: response.total || 0, success: true };
            } catch (error: any) {
              console.error('list notification rules failed', error);
              message.error(error.message || t('fetchFailed'));
              return { data: [], total: 0, success: false };
            }
          }}
          rowKey="id"
          search={{ labelWidth: 'auto', defaultCollapsed: false }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
          }}
          options={{ density: true, fullScreen: true, setting: true, reload: true }}
          toolBarRender={() => [
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setFormMode('create');
                setSelected(undefined);
                setFormOpen(true);
              }}
            >
              {t('create')}
            </Button>,
          ]}
          size="middle"
          bordered
          scroll={{ y: tableScrollY, x: 1100 }}
        />
      </div>

      <ModalForm
        title={formMode === 'create' ? t('create') : t('edit')}
        width={520}
        open={formOpen}
        onOpenChange={setFormOpen}
        modalProps={{ destroyOnHidden: true, mask: { closable: false } }}
        submitTimeout={3000}
        onFinish={handleSubmit}
        initialValues={
          formMode === 'create'
            ? { channel: 'EMAIL', isAsync: true, isEnabled: true }
            : { ...selected }
        }
      >
        <ProFormSelect
          name="eventType"
          label={t('eventType')}
          options={eventTypeOptions(t)}
          disabled={formMode === 'edit'}
          rules={[{ required: true, message: t('requiredEventType') }]}
          fieldProps={{ showSearch: true }}
        />
        <ProFormSelect
          name="channel"
          label={t('channel')}
          options={channelOptions(t)}
          rules={[{ required: true, message: t('requiredChannel') }]}
        />
        <ProFormSwitch name="isAsync" label={t('isAsync')} fieldProps={{ checkedChildren: t('isAsyncOn'), unCheckedChildren: t('isAsyncOff') }} />
        <ProFormSwitch name="isEnabled" label={t('isEnabled')} />
        <ProFormTextArea name="remark" label={t('remark')} fieldProps={{ rows: 2 }} />
      </ModalForm>

      <ModalForm<{ target?: string }>
        title={t('testDispatchTitle', {
          event: testTarget?.eventType ? t(`eventTypeMap.${testTarget.eventType}`) : '',
        })}
        width={480}
        open={!!testTarget}
        onOpenChange={(open) => {
          if (!open) setTestTarget(undefined);
        }}
        modalProps={{ destroyOnHidden: true }}
        submitTimeout={10000}
        onFinish={handleTestDispatch}
      >
        <ProFormText
          name="target"
          label={t('testTarget')}
          placeholder={t('testTargetPlaceholder')}
          tooltip={t('testTargetHint')}
        />
      </ModalForm>
    </ContentContainer>
  );
};

export default NotificationRulePage;
