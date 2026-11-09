import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import dayjs, { type Dayjs } from 'dayjs';
import ListTable from '@/components/common/ListTable';
import TableActions from '@/components/common/TableActions';
import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Dropdown, Tag, App } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_PermissionAuditLog as PermissionAuditLog } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListPermissionAuditLogs } from '@/api/hooks/permission-audit-log';
import { exportAuditLogs, type AuditExportFormat } from '@/utils/csv';
import { exportAuditLogsServer } from '@/api/hooks/audit-export';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getActionMap, getActionOptions } from './constants';
import PermissionAuditLogDetailDrawer from './DetailDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 权限审计日志页面
 */
const PermissionAuditLogPage = () => {
  const { t } = useTranslation('permission-audit-log');
  const actionRef = useRef<ActionType>(null);
  // 最近一次列表查询参数：导出 CSV 时沿用当前搜索/排序条件
  const latestParamsRef = useRef<Record<string, any>>({});
  const { message } = App.useApp();
  const [detailRecord, setDetailRecord] = useState<PermissionAuditLog | null>(null);

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const actionMap = getActionMap(t);
  const actionOptions = getActionOptions(t);

  const columns: ProColumns<PermissionAuditLog>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 60,
      hideInSearch: true,
      render: (_, _record, index) => {
        const pagination = actionRef.current?.pageInfo;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || 20;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
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
      title: t('action'),
      dataIndex: 'action',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: actionOptions,
      },
      render: (_, record) => {
        const config = actionMap[record.action as keyof typeof actionMap];
        if (!config) return record.action || '-';
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('targetType'),
      dataIndex: 'targetType',
      width: 120,
    },
    {
      title: t('targetName'),
      dataIndex: 'targetName',
      width: 150,
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('reason'),
      dataIndex: 'reason',
      width: 200,
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('operatorName'),
      dataIndex: 'operatorName',
      width: 120,
      hideInSearch: false,
      render: (_, record) => {
        const info = record.operatorName;
        if (!info) return '-';
        return info || '-';
      },
    },
    {
      title: t('ipAddress'),
      dataIndex: 'ipAddress',
      width: 140,
    },
    {
      // 行操作列标题：页面 ns 的 action 键已被「操作类型」数据列占用，走独立键
      title: t('rowActions'),
      valueType: 'option',
      width: 110,
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'detail',
              label: t('detail'),
              icon: <InfoCircleOutlined />,
              onClick: () => setDetailRecord(record),
            },
          ]}
        />
      ),
    },
  ];

  // 按当前搜索条件导出 CSV（客户端分页聚合，上限 1 万行；全量归档走后端 JSONL 任务）
  // 服务端全量导出（XLSX，上限 50 万行）：当前搜索条件透传给后端，
  // 突破客户端聚合导出的 1 万行上限
  const handleServerExport = async () => {
    try {
      await exportAuditLogsServer(
        'permission',
        // 与列表请求同构的搜索条件（剔除 ProTable params 里的分页键：后端
        // repo 对非表列字段直接报错），经 PaginationQuery 同源序列化
        // （contains 转换/空值清理），导出的即当前搜索看到的。
        new PaginationQuery({
          formValues: Object.fromEntries(
            Object.entries(latestParamsRef.current ?? {}).filter(
              ([k]) => !['current', 'pageSize'].includes(k),
            ),
          ),
        }),
      );
      message.success(t('exportServerSuccess'));
    } catch (error: any) {
      console.error('server-side audit export failed', error);
      message.error(error?.message || t('exportServerFailed'));
    }
  };

  const handleExport = async (format: AuditExportFormat) => {
    const exportColumns = columns
      .filter((c) => c.dataIndex && !c.hideInTable)
      .map((c) => ({
        key: String(c.dataIndex),
        title: typeof c.title === 'string' ? c.title : String(c.dataIndex),
      }));
    try {
      await exportAuditLogs({
        fetcher: (q) => fetchListPermissionAuditLogs(q),
        filename: `permission-audit-logs-${Date.now()}.csv`,
        columns: exportColumns,
        params: latestParamsRef.current,
      }, format);
    } catch (error: any) {
      message.error(error?.message || 'Export failed');
    }
  };

  return (
    <>
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ListTable<PermissionAuditLog>
            actionRef={actionRef}
            columns={columns}
            request={async (params, sorter) => {
              latestParamsRef.current = params;
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || 20,
                },
                formValues: Object.fromEntries(
                  Object.entries(params).filter(([key]) => !['current', 'pageSize'].includes(key)),
                ),
                orderBy:
                  sorter && Object.keys(sorter).length > 0
                    ? Object.entries(sorter).map(([key, value]) =>
                        value === 'ascend' ? key : `-${key}`,
                      )
                    : undefined,
              });

              const response = await fetchListPermissionAuditLogs(query);

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
            toolBarRender={() => [
              <Dropdown
                key="export"
                menu={{
                  items: [
                    { key: 'csv', label: t('exportCsv') },
                    { key: 'xlsx', label: t('exportXlsx') },
                    { type: 'divider' },
                    { key: 'server-xlsx', label: t('exportServerXlsx') },
                  ],
                  onClick: ({ key }) => {
                    if (key === 'server-xlsx') {
                      handleServerExport();
                      return;
                    }
                    handleExport(key as AuditExportFormat);
                  },
                }}
              >
                <Button>{t('export')}</Button>
              </Dropdown>,
            ]}

            options={{
              density: true,
              fullScreen: true,
              setting: { settingIcon: <ColumnSlidersIcon /> },
              reload: true,
            }}
            size="middle"
            bordered
            cardBordered={false}
            scroll={{
              y: tableScrollY,
              x: 1300,
            }}
          />
        </div>
      </ContentContainer>

      <PermissionAuditLogDetailDrawer
        open={detailRecord !== null}
        data={detailRecord}
        onClose={() => setDetailRecord(null)}
      />
    </>
  );
};

export default PermissionAuditLogPage;
