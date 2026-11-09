import { useRef } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import ListTable from '@/components/common/ListTable';
import TableExportButton from '@/components/common/TableExportButton';
import { Progress } from 'antd';
import { useTranslation } from 'react-i18next';
import { PaginationQuery } from '@/core';
import { useGetAiUsageSummary } from '@/api/hooks/ai-usage';
import { fetchListAiUsageLogs } from '@/api/hooks/ai-usage';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { TABLE } from '@/config/constants';

interface UsageRow {
  id?: string; // uint64 → proto3 JSON 下为字符串
  modelName?: string;
  promptTokens?: number;
  completionTokens?: number;
  totalTokens?: number;
  durationMs?: number;
  createdAt?: string;
}

/**
 * AI 用量页：当月汇总（tokens/调用次数/配额进度）+ 用量流水列表。
 * 两者口径相同：平台用户见全量，租户用户由后端租户隔离自动限定本租户。
 */
export default function AiUsagePage() {
  const { t } = useTranslation('aiUsage');
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const summary = useGetAiUsageSummary();

  // 数字列等宽数位（docs/design-language.md:163 既有建议；此前全仓未落，实测 td 取 normal）。
  // 这页一屏都是 tokens/耗时，位数错位比别的列表页更扎眼。
  const tabularNumsCell = () => ({ style: { fontVariantNumeric: 'tabular-nums' } });

  // ProTable 的 render 首参是**已经渲染好的值**（null 已被换成 '-'），所以取数一律从 record 读：
  // 早期写法 Number(v ?? 0) 在空值上拿到的是 '-'，Number('-') = NaN；`v ?? 0` 更是永远不触发。
  const intCell =
    (key: 'promptTokens' | 'completionTokens' | 'totalTokens') =>
    (_: unknown, record: UsageRow) =>
      Number(record[key] ?? 0).toLocaleString();

  const columns: ProColumns<UsageRow>[] = [
    { title: t('model'), dataIndex: 'modelName', minWidth: 160 },
    { title: t('promptTokens'), dataIndex: 'promptTokens', width: 130, onCell: tabularNumsCell, render: intCell('promptTokens') },
    { title: t('completionTokens'), dataIndex: 'completionTokens', width: 140, onCell: tabularNumsCell, render: intCell('completionTokens') },
    { title: t('totalTokens'), dataIndex: 'totalTokens', width: 110, onCell: tabularNumsCell, render: intCell('totalTokens') },
    {
      title: t('duration'),
      dataIndex: 'durationMs',
      width: 100,
      onCell: tabularNumsCell,
      // 向量化/embedding 一路径历史上没记耗时，NULL 就留空而不是渲染成「- ms」
      render: (_, record) => (record.durationMs == null ? '-' : `${record.durationMs} ms`),
    },
    { title: t('time'), dataIndex: 'createdAt', width: 180, valueType: 'dateTime' },
  ];

  const summaryData = summary.data;
  // quotaLimit 为 uint64（字符串），参与比较/运算前转回 number
  const quotaLimitNum = Number(summaryData?.quotaLimit ?? 0);
  const quotaPct =
    summaryData?.quotaConfigured && quotaLimitNum > 0
      ? Math.min(100, Math.round(((summaryData.monthTokens ?? 0) * 100) / (quotaLimitNum || 1)))
      : 0;

  // 汇总卡数值是"文字"，取值随主题走 --metric-* 档（定义在 styles/semantic-text.css，
  // 与 vue-element 端同值）：原实现用主色 token 和 tailwind cyan-500，实测
  // 暗色主色 2.90:1、cyan-500 对白底 2.37:1，24px 大字号的 3.0 下限都过不了。
  const cards = [
    {
      label: t('summary.monthTokens'),
      value: (summaryData?.monthTokens ?? 0).toLocaleString(),
      color: 'text-[color:var(--metric-blue)]',
    },
    {
      label: t('summary.monthCalls'),
      value: (summaryData?.monthCalls ?? 0).toLocaleString(),
      color: 'text-[color:var(--metric-cyan)]',
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        {/* 汇总卡 */}
        <div className="mb-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
          {cards.map((c) => (
            <div
              key={c.label}
              className="rounded-xl border border-solid border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900"
            >
              <div className="mb-2 text-sm text-[color:var(--ant-color-text-secondary)]">{c.label}</div>
              <div className={`text-2xl font-semibold tabular-nums ${c.color}`}>{c.value}</div>
            </div>
          ))}
          <div className="rounded-xl border border-solid border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
            <div className="mb-2 text-sm text-[color:var(--ant-color-text-secondary)]">{t('summary.quota')}</div>
            {summaryData?.quotaConfigured ? (
              <>
                <Progress percent={quotaPct} status={quotaPct > 80 ? 'exception' : 'normal'} />
                <div className="mt-1 text-xs text-[color:var(--ant-color-text-secondary)]">
                  {(summaryData.monthTokens ?? 0).toLocaleString()} / {(summaryData.quotaLimit ?? 0).toLocaleString()}
                </div>
              </>
            ) : (
              <div className="text-2xl font-semibold text-[color:var(--ant-color-text-secondary)]">∞</div>
            )}
          </div>
        </div>

        {/* 流水列表 */}
        <ListTable<UsageRow>
          actionRef={actionRef}
          rowKey="id"
          search={false}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
          toolBarRender={() => [
            // 导出：客户端聚合（小数据量快速路径）+ 服务端全量（本页无搜索条件，
            // 查询恒为空；租户用户由后端 viewer 语义收窄到本租户）
            <TableExportButton
              key="export"
              fetcher={fetchListAiUsageLogs}
              columns={columns}
              filename="ai-usage-logs"
              serverExport={{
                url: 'admin/v1/ai/usage-logs:export',
                buildQuery: () => new PaginationQuery({}),
              }}
            />,
          ]}
          scroll={{ y: tableScrollY, x: 800 }}
          request={async (params) => {
            const { current, pageSize } = params;
            const query = new PaginationQuery({
              paging: { page: current || 1, pageSize: pageSize || TABLE.DEFAULT_PAGE_SIZE },
            });
            const res = await fetchListAiUsageLogs(query);
            return { data: res.items || [], total: Number(res.total ?? 0), success: true };
          }}
          columns={columns}
        />
      </div>
    </ContentContainer>
  );
}
