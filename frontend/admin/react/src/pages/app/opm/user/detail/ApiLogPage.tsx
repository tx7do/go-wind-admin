import { useRef } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import dayjs, { type Dayjs } from 'dayjs';
import ListTable from '@/components/common/ListTable';
import { Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_ApiAuditLog as ApiAuditLog } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE, METHOD_LIST } from '@/config/constants';
import { fetchListApiAuditLogs } from '@/api/hooks/api-audit-log';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

interface ApiLogPageProps {
  userId: number | undefined;
}

/**
 * 成功状态获取颜色
 */
function successToColor(success: boolean | undefined): string {
  if (success === true) return 'success';
  if (success === false) return 'error';
  return 'default';
}

/**
 * 成功状态获取文本
 */
function successToName(
  t: (key: string) => string,
  success: boolean | undefined,
  statusCode?: number,
): string {
  if (success === true) return `${t('success.true')}${statusCode ? ` (${statusCode})` : ''}`;
  if (success === false) return `${t('success.false')}${statusCode ? ` (${statusCode})` : ''}`;
  return '-';
}

/**
 * 用户详情 - API审计日志 Tab
 */
const ApiLogPage: React.FC<ApiLogPageProps> = ({ userId }) => {
  const { t } = useTranslation('api-audit-log');
  const actionRef = useRef<ActionType>(null);

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const columns: ProColumns<ApiAuditLog>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 60,
      hideInSearch: true,
      render: (_, _record, index) => {
        const pagination = actionRef.current?.pageInfo;
        const page = pagination?.current || 1;
        // 首次渲染时 pageInfo 可能尚未就绪，回退到表格默认分页大小，
        // 避免第一页序号与实际 pageSize 不一致导致错位。
        const pageSize = pagination?.pageSize || TABLE.DEFAULT_PAGE_SIZE;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 170,
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
    {
      title: t('success'),
      dataIndex: 'success',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: [
          { label: t('success.true'), value: true },
          { label: t('success.false'), value: false },
        ],
      },
      render: (_, record) => (
        <Tag color={successToColor(record.success)}>
          {successToName(t, record.success, record.statusCode)}
        </Tag>
      ),
    },
    {
      title: t('httpMethod'),
      dataIndex: 'httpMethod',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: METHOD_LIST,
      },
      render: (_, record) => {
        const colorMap: Record<string, string> = {
          GET: 'success',
          POST: 'processing',
          PUT: 'warning',
          DELETE: 'error',
          PATCH: 'default',
          HEAD: 'default',
          OPTIONS: 'default',
        };
        // Tag 预设色名不能当 CSS color 用：'success' 这类不是 <color>，整条声明被浏览器丢弃
        return (
          <Tag color={colorMap[record.httpMethod || ''] || 'default'}>
            {record.httpMethod}
          </Tag>
        );
      },
    },
    {
      title: t('path'),
      dataIndex: 'path',
      width: 250,
      ellipsis: true,
    },
    {
      title: t('latencyMs'),
      dataIndex: 'latencyMs',
      width: 100,
      hideInSearch: true,
    },
    {
      title: t('platform'),
      dataIndex: 'deviceInfo',
      width: 180,
      hideInSearch: true,
      render: (_, record) => {
        const info = record.deviceInfo as any;
        if (!info) return '-';
        return `${info.osName || ''} ${info.browserName || ''}`.trim() || '-';
      },
    },
    {
      title: t('geoLocation'),
      dataIndex: 'geoLocation',
      width: 150,
      hideInSearch: true,
      render: (_, record) => {
        const geo = record.geoLocation as any;
        if (!geo) return '-';
        return `${geo.province || ''} ${geo.city || ''}`.trim() || '-';
      },
    },
    {
      title: t('ipAddress'),
      dataIndex: 'ipAddress',
      width: 140,
    },
  ];

  return (
    <div
      ref={containerRef}
      style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
    >
      <ListTable<ApiAuditLog>
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
            formValues.user_id = userId.toString();
          }

          const query = new PaginationQuery({
            paging: {
              page: params.current || 1,
              pageSize: params.pageSize || 20,
            },
            formValues,
          });

          const response = await fetchListApiAuditLogs(query);

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

export default ApiLogPage;
