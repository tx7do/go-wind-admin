import { useRef } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import dayjs, { type Dayjs } from 'dayjs';
import ListTable from '@/components/common/ListTable';
import { Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import type { internal_messageservicev1_InternalMessageRecipient as InboxItem } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListUserInbox } from '@/api/hooks/internal-message';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import {
  getRecipientStatusMap,
  getRecipientStatusOptions,
} from '@/pages/app/internal-message/inbox/constants';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

interface InternalMessagePageProps {
  userId: number | undefined;
}

/**
 * 用户详情 - 站内信 Tab
 */
const InternalMessagePage: React.FC<InternalMessagePageProps> = ({ userId }) => {
  const { t } = useTranslation('inbox');
  const actionRef = useRef<ActionType>(null);

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const statusMap = getRecipientStatusMap(t);

  const columns: ProColumns<InboxItem>[] = [
    {
      title: t('title'),
      dataIndex: 'title',
      ellipsis: true,
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
      title: t('createdAt'),
      dataIndex: 'createdAt',
      valueType: 'dateTimeRange',
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
        presets: [
          {
            label: t('common:dateRange.today'),
            value: [dayjs().startOf('day'), dayjs().endOf('day')],
          },
          {
            label: t('common:dateRange.yesterday'),
            value: [
              dayjs().subtract(1, 'day').startOf('day'),
              dayjs().subtract(1, 'day').endOf('day'),
            ],
          },
          {
            label: t('common:dateRange.thisWeek'),
            value: [dayjs().startOf('week'), dayjs().endOf('week')],
          },
          {
            label: t('common:dateRange.lastWeek'),
            value: [
              dayjs().subtract(1, 'week').startOf('week'),
              dayjs().subtract(1, 'week').endOf('week'),
            ],
          },
          {
            label: t('common:dateRange.thisMonth'),
            value: [dayjs().startOf('month'), dayjs().endOf('month')],
          },
          {
            label: t('common:dateRange.lastMonth'),
            value: [
              dayjs().subtract(1, 'month').startOf('month'),
              dayjs().subtract(1, 'month').endOf('month'),
            ],
          },
        ],
      },
    },
  ];

  return (
    <div
      ref={containerRef}
      style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
    >
      <ListTable<InboxItem>
        actionRef={actionRef}
        columns={columns}
        request={async (params) => {
          const formValues: Record<string, any> = {};
          Object.entries(params).forEach(([key, value]) => {
            if (!['current', 'pageSize'].includes(key) && value !== undefined) {
              formValues[key] = value;
            }
          });

          // 添加 userId 过滤
          if (userId) {
            formValues.recipient_user_id = userId.toString();
          }

          const query = new PaginationQuery({
            paging: {
              page: params.current || 1,
              pageSize: params.pageSize || 20,
            },
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
        }}
        pagination={{
          defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
          showSizeChanger: true,
          showQuickJumper: true,
        }}
        options={{
          density: true,
          fullScreen: true,
          setting: { settingIcon: <ColumnSlidersIcon /> },
          reload: true,
        }}
        size="middle"
        bordered
        cardBordered={false}
        scroll={{ y: tableScrollY }}
      />
    </div>
  );
};

export default InternalMessagePage;
