import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import dayjs, { type Dayjs } from 'dayjs';
import ListTable from '@/components/common/ListTable';
import TableActions from '@/components/common/TableActions';
import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Dropdown, Tag, App } from 'antd';
import { useTranslation } from 'react-i18next';
import type { permissionservicev1_PolicyEvaluationLog as PolicyEvaluationLog } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE, METHOD_LIST } from '@/config/constants';
import { fetchListPolicyEvaluationLogs } from '@/api/hooks/policy-evaluation-log';
import { exportAuditLogs, type AuditExportFormat } from '@/utils/csv';
import { exportAuditLogsServer } from '@/api/hooks/audit-export';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import PolicyEvaluationLogDetailDrawer from './DetailDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 评估结果筛选项
 */
const getResultStatusList = (t: (key: string) => string) => [
  { label: t('result.true'), value: 'true' },
  { label: t('result.false'), value: 'false' },
];

/**
 * 根据评估结果获取标签颜色
 */
function resultToColor(result: boolean | undefined): string {
  if (result === true) return 'success';
  if (result === false) return 'error';
  return 'default';
}

/**
 * 根据评估结果获取显示文本
 */
function resultToName(t: (key: string) => string, result: boolean | undefined): string {
  if (result === true) return t('result.true');
  if (result === false) return t('result.false');
  return '-';
}

/**
 * 策略评估日志页面
 */
const PolicyEvaluationLog = () => {
  const { t } = useTranslation('policy-evaluation-log');
  const actionRef = useRef<ActionType>(null);
  // 最近一次列表查询参数：导出 CSV 时沿用当前搜索/排序条件
  const latestParamsRef = useRef<Record<string, any>>({});
  const { message } = App.useApp();
  const [detailRecord, setDetailRecord] = useState<PolicyEvaluationLog | null>(null);

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const columns: ProColumns<PolicyEvaluationLog>[] = [
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
      title: t('result'),
      dataIndex: 'result',
      width: 100,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        getResultStatusList(t).map((item) => [
          item.value,
          { text: item.label, status: item.value === 'true' ? 'Success' : 'Error' },
        ]),
      ),
      render: (_, record) => (
        <Tag color={resultToColor(record.result)}>
          {resultToName(t, record.result)}
        </Tag>
      ),
    },
    {
      title: t('requestMethod'),
      dataIndex: 'requestMethod',
      width: 110,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        METHOD_LIST.map((item) => [item.value, { text: item.label, status: 'Default' }]),
      ),
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
          <Tag color={colorMap[record.requestMethod || ''] || 'default'}>
            {record.requestMethod}
          </Tag>
        );
      },
    },
    {
      title: t('requestPath'),
      dataIndex: 'requestPath',
      width: 240,
      ellipsis: true,
    },
    {
      title: t('policyId'),
      dataIndex: 'policyId',
      width: 90,
    },
    {
      title: t('permissionId'),
      dataIndex: 'permissionId',
      width: 100,
      hideInSearch: true,
    },
    {
      title: t('userId'),
      dataIndex: 'userId',
      width: 90,
    },
    {
      title: t('membershipId'),
      dataIndex: 'membershipId',
      width: 110,
      hideInSearch: true,
    },
    {
      title: t('tenantId'),
      dataIndex: 'tenantId',
      width: 90,
      hideInSearch: true,
    },
    {
      title: t('ipAddress'),
      dataIndex: 'ipAddress',
      width: 140,
    },
    {
      title: t('traceId'),
      dataIndex: 'traceId',
      width: 180,
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('evaluationContext'),
      dataIndex: 'evaluationContext',
      width: 200,
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('effectDetails'),
      dataIndex: 'effectDetails',
      width: 180,
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('scopeSql'),
      dataIndex: 'scopeSql',
      width: 160,
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('action'),
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
        'policy_evaluation',
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
        fetcher: (q) => fetchListPolicyEvaluationLogs(q),
        filename: `policy-evaluation-logs-${Date.now()}.csv`,
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
          <ListTable<PolicyEvaluationLog>
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
                  Object.entries(params).filter(
                    ([key]) => !['current', 'pageSize'].includes(key),
                  ),
                ),
                orderBy:
                  sorter && Object.keys(sorter).length > 0
                    ? Object.entries(sorter).map(([key, value]) =>
                        value === 'ascend' ? key : `-${key}`,
                      )
                    : undefined,
              });

              const response = await fetchListPolicyEvaluationLogs(query);

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
              x: 1600,
            }}
          />
        </div>
      </ContentContainer>

      <PolicyEvaluationLogDetailDrawer
        open={detailRecord !== null}
        data={detailRecord}
        onClose={() => setDetailRecord(null)}
      />
    </>
  );
};

export default PolicyEvaluationLog;
