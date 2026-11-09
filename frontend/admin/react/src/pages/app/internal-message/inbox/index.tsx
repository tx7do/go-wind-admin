import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import dayjs, { type Dayjs } from 'dayjs';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
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
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

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
      // 字段级宽度 = span(8) × colSize：标题 8 / 状态 5 / 接收时间 11
      colSize: 1,
      // 搜索输入框撑满列宽（三等分中的一份）
      fieldProps: {
        style: { width: '100%' },
      },
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
      colSize: 0.625,
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
      // 创建时间区间搜索：只进搜索表单不进表格；transform 映射为 go-crud 的时间范围
      // 操作符（__gte/__lte 在 PaginationQuery 的操作符后缀表里，不会被误加 __contains）。
      // span 11/24：showTime 双输入 RangePicker 内容固有宽约 390px，span 8 的列
      // （侧栏展开 1650px 视口约 420px）放不下「标签+控件」，整列会塌陷
      title: t('createdAt'),
      dataIndex: 'createdAt',
      valueType: 'dateTimeRange',
      colSize: 1.375,
      hideInTable: true,
      search: {
        transform: (values?: Array<string | Dayjs>) => {
          const fmt = (v?: string | Dayjs) =>
            typeof v === 'string' ? v : v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : undefined;
          return {
            created_at__gte: fmt(values?.[0]),
            created_at__lte: fmt(values?.[1]),
          };
        },
      },
      fieldProps: {
        showTime: true,
        // 快捷区间（对齐 ele/vben）：今天/昨天/本周/上周/本月/上月
        presets: [
          {
            label: t('common:dateRange.today'),
            value: [dayjs().startOf('day'), dayjs().endOf('day')],
          },
          {
            label: t('common:dateRange.yesterday'),
            value: [dayjs().subtract(1, 'day').startOf('day'), dayjs().subtract(1, 'day').endOf('day')],
          },
          {
            label: t('common:dateRange.thisWeek'),
            value: [dayjs().startOf('week'), dayjs().endOf('week')],
          },
          {
            label: t('common:dateRange.lastWeek'),
            value: [dayjs().subtract(1, 'week').startOf('week'), dayjs().subtract(1, 'week').endOf('week')],
          },
          {
            label: t('common:dateRange.thisMonth'),
            value: [dayjs().startOf('month'), dayjs().endOf('month')],
          },
          {
            label: t('common:dateRange.lastMonth'),
            value: [dayjs().subtract(1, 'month').startOf('month'), dayjs().subtract(1, 'month').endOf('month')],
          },
        ],
      },
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 180,
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'read',
              label: t('markRead'),
              icon: <CheckOutlined />,
              hidden: record.status === 'READ',
              onClick: () => {
                if (record.id) {
                  markReadMutation.mutate({
                    userId: record.recipientUserId,
                    recipientIds: [record.id],
                  });
                }
              },
            },
            {
              key: 'delete',
              label: t('common:button.delete'),
              icon: <DeleteOutlined />,
              danger: true,
              confirm: {
                title: t('deleteConfirmTitle'),
                description: t('deleteConfirmDesc', { moduleName: t('moduleName') }),
              },
              onClick: () => {
                if (record.id) {
                  deleteMutation.mutate({
                    userId: record.recipientUserId,
                    recipientIds: [record.id],
                  });
                }
              },
            },
          ]}
        />
      ),
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ListTable<InboxItem>
          actionRef={actionRef}
          columns={columns}
          params={{ userId }}
          request={async (params, _sorter, _filter) => {
            // 收件箱只看自己的：不传 recipient_user_id 会按租户过滤，列出其他用户的收件记录。
            // userId 未就绪（静默恢复期 profile 拉取失败等半残会话）时绝不能发请求——
            // String(undefined) 会把字面量 "undefined" 当用户 ID 发出去，后端解析直接 500；
            // 先回空数据，等 userId 就位后由上方 params 变更自动触发重查。
            if (!userId) {
              return { data: [], total: 0, success: true };
            }
            const formValues: Record<string, any> = {
              recipient_user_id: String(userId),
            };
            Object.entries(params).forEach(([key, value]) => {
              // userId 是"就绪后自动重查"的触发参数，不是搜索字段，不能进查询条件
              if (!['current', 'pageSize', 'userId'].includes(key) && value !== undefined) {
                formValues[key] = value;
              }
            });

            const query = new PaginationQuery({
              formValues,
            });

            const response = await fetchListUserInbox(query);

            return {
              data: response.items || [],
              total: Number(response.total ?? 0),
              success: true,
            };
          }}
          rowKey="id"
          search={{ optionRender: searchOptionRender,
            labelWidth: 'auto',
            defaultCollapsed: false,
            // 全局 span 8，各字段再乘自己的 colSize（标题 1 / 状态 0.625 / 接收时间
            // 1.375）= 8/5/11 共 24 一行放下：状态列最窄（标签短、下拉瘦），
            // 让宽给内容固有宽约 390px 的双输入区间控件
            span: 8,
          }}
          pagination={false}
          options={{
            density: true,
            fullScreen: true,
            setting: { settingIcon: <ColumnSlidersIcon /> },
            reload: true,
          }}
          toolBarRender={() => [
            // 导出走 TableExportButton 自己的聚合查询，不经过表格 request 闭包，
            // recipient_user_id 必须在这里再注一次，否则导出会拉到同租户其他用户的收件记录
            ...(userId
              ? [
                  <TableExportButton
                    key="export"
                    fetcher={fetchListUserInbox}
                    columns={columns}
                    filename="im-inbox"
                    params={{ recipient_user_id: String(userId) }}
                  />,
                ]
              : []),
            <Button
              key="mark-all-read"
              icon={<CheckOutlined />}
              disabled={!userId}
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
