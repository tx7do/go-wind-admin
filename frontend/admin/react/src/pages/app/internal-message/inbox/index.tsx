import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, App } from 'antd';
import { DeleteOutlined, CheckOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { internal_messageservicev1_InternalMessageRecipient as InboxItem } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { useAuthStore } from '@/stores';
import {
  fetchListUserInbox,
  useMarkNotificationAsRead,
  useDeleteNotificationFromInbox,
} from '@/api/hooks/internal-message';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { InboxDetailDrawer } from './components/InboxDetailDrawer';
import { getRecipientStatusMap, getRecipientStatusOptions } from './constants';

/**
 * 收件箱页面
 */
const InboxList = () => {
  const { t } = useTranslation('inbox');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const userId = useAuthStore((s) => s.userInfo?.id);

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const statusMap = getRecipientStatusMap(t);

  // 全文阅读抽屉
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailRecord, setDetailRecord] = useState<InboxItem | null>(null);

  const refreshInbox = () => {
    actionRef.current?.reload();
    queryClient.invalidateQueries({ queryKey: ['listUserInbox'] });
    // 顶栏通知徽标（inboxPreview 查询）同步刷新
    queryClient.invalidateQueries({ queryKey: ['inboxPreview'] });
  };

  // 标记已读
  const markReadMutation = useMarkNotificationAsRead({
    onSuccess: () => {
      message.success(t('markReadSuccess'));
      refreshInbox();
    },
    onError: (error: Error) => {
      message.error(error.message || t('markReadFailed'));
    },
  });

  // 打开全文抽屉时的静默已读（不弹 toast，列表与徽标仍刷新）
  const silentMarkReadMutation = useMarkNotificationAsRead({
    onSuccess: refreshInbox,
    onError: () => {},
  });

  // 一键全部已读（后端约定：recipientIds 为空 = 该用户全部未读）
  const markAllReadMutation = useMarkNotificationAsRead({
    onSuccess: () => {
      message.success(t('markAllReadSuccess'));
      refreshInbox();
    },
    onError: (error: Error) => {
      message.error(error.message || t('markAllReadFailed'));
    },
  });

  // 删除
  const deleteMutation = useDeleteNotificationFromInbox({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      refreshInbox();
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  const openDetail = (record: InboxItem) => {
    setDetailRecord(record);
    setDetailOpen(true);
    // 打开即已读：未读（RECEIVED）消息在阅读时静默标记
    if (record.status === 'RECEIVED' && record.id) {
      silentMarkReadMutation.mutate({
        userId: record.recipientUserId,
        recipientIds: [record.id],
      });
    }
  };

  // 列配置
  const columns: ProColumns<InboxItem>[] = [
    {
      title: t('title'),
      dataIndex: 'title',
      ellipsis: true,
      render: (_, record) => (
        <a onClick={() => openDetail(record)} style={{ cursor: 'pointer' }}>
          {record.title || '-'}
        </a>
      ),
    },
    {
      title: t('status'),
      dataIndex: 'status',
      valueType: 'select',
      fieldProps: {
        options: getRecipientStatusOptions(t),
      },
      width: 100,
      render: (_, record) => {
        const status = record.status as keyof typeof statusMap;
        const config = statusMap[status] || { text: status, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('readAt'),
      dataIndex: 'readAt',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 90,
      render: (_, record) => [
        record.status !== 'READ' && (
          <a
            key="read"
            onClick={() => {
              if (record.id) {
                markReadMutation.mutate({
                  userId: record.recipientUserId,
                  recipientIds: [record.id],
                });
              }
            }}
          >
            <CheckOutlined />
          </a>
        ),
        <Popconfirm
          key="delete"
          title={t('deleteConfirmTitle')}
          description={t('deleteConfirmDesc', { moduleName: t('moduleName') })}
          onConfirm={() => {
            if (record.id) {
              deleteMutation.mutate({
                userId: record.recipientUserId,
                recipientIds: [record.id],
              });
            }
          }}
          okText={t('common:button.ok')}
          cancelText={t('common:button.cancel')}
        >
          <a style={{ color: 'var(--ant-color-error)' }}>
            <DeleteOutlined />
          </a>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ProTable<InboxItem>
          actionRef={actionRef}
          columns={columns}
          request={async (params, _sorter, _filter) => {
            try {
              const formValues: Record<string, any> = {};
              Object.entries(params).forEach(([key, value]) => {
                if (!['current', 'pageSize'].includes(key) && value !== undefined) {
                  formValues[key] = value;
                }
              });
              // 收件箱只看自己的：不传 recipient_user_id 会按租户过滤，列出其他用户的收件记录
              formValues.recipient_user_id = String(userId);

              const query = new PaginationQuery({
                formValues,
              });

              const response = await fetchListUserInbox(query);

              return {
                data: response.items || [],
                total: response.total || 0,
                success: true,
              };
            } catch (error: any) {
              message.error(error.message || t('fetchFailed'));
              return { data: [], total: 0, success: false };
            }
          }}
          rowKey="id"
          search={{
            labelWidth: 'auto',
            defaultCollapsed: false,
          }}
          pagination={false}
          options={{
            density: true,
            fullScreen: true,
            setting: true,
            reload: true,
          }}
          toolBarRender={() => [
            <Button
              key="mark-all-read"
              icon={<CheckOutlined />}
              loading={markAllReadMutation.isPending}
              onClick={() => markAllReadMutation.mutate({ userId, recipientIds: [] })}
            >
              {t('markAllRead')}
            </Button>,
          ]}
          size="middle"
          bordered
          cardBordered={false}
          scroll={{ y: tableScrollY }}
        />
      </div>

      <InboxDetailDrawer
        open={detailOpen}
        record={detailRecord}
        onClose={() => setDetailOpen(false)}
      />
    </ContentContainer>
  );
};

export default InboxList;
