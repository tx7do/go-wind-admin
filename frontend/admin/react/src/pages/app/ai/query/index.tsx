import { useEffect, useRef, useState } from 'react';
import { Button, Input, Segmented, Table, Tag, theme, Typography, App } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { SendOutlined, ThunderboltOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { useI18n } from '@go-wind/admin-react';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { useAskAiQuery, type AskAiQueryHistoryItem, type AskAiQueryParams } from '@/api/hooks/ai-query';
import type { aiservicev1_AiQueryRow } from '@/api/generated/admin/service/v1';

type Round = {
  question: string;
  sql?: string;
  columns?: string[];
  rows?: aiservicev1_AiQueryRow[];
  answer?: string;
  errorMessage?: string;
  loading: boolean;
};

/** 示例问题（点击直接填入输入框）。 */
const SAMPLE_KEYS = ['sample1', 'sample2', 'sample3'] as const;

/**
 * 智能问数页（平台管理员专属）：自然语言 → 只读 SQL → 结构化结果 → AI 结论。
 * 对话式布局：输入区吸底、消息流自动滚到最新；SQL 原样展示便于核对。
 */
export default function AiQueryPage() {
  const { t, i18n } = useI18n('aiQuery');
  const { message } = App.useApp();
  const [input, setInput] = useState('');
  const [rounds, setRounds] = useState<Round[]>([]);
  const bottomRef = useRef<HTMLDivElement>(null);

  // 新轮次追加/更新时滚到底部
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [rounds]);

  const askMutation = useAskAiQuery({
    onSuccess: (resp) => {
      setRounds((prev) => {
        if (prev.length === 0) return prev;
        const last = prev.length - 1;
        const next = [...prev];
        next[last] = {
          ...next[last]!,
          sql: resp.sql,
          columns: resp.columns ?? [],
          rows: resp.rows ?? [],
          answer: resp.answer ?? '',
          errorMessage: resp.errorMessage ?? '',
          loading: false,
        };
        return next;
      });
    },
    onError: (error: Error) => {
      setRounds((prev) => {
        if (prev.length === 0) return prev;
        const last = prev.length - 1;
        const next = [...prev];
        next[last] = { ...next[last]!, loading: false };
        return next;
      });
      message.error(error.message || t('failed'));
    },
  });

  const handleAsk = () => {
    const question = input.trim();
    if (!question || askMutation.isPending) return;
    // 携带最近 5 轮历史（问题+SQL+结果摘要），供模型消解追问里的指代
    const history: AskAiQueryHistoryItem[] = rounds
      .filter((r) => !r.loading && r.sql)
      .slice(-5)
      .map((r) => ({
        question: r.question,
        sql: r.sql ?? '',
        resultSummary: (r.rows ?? []).slice(0, 3).map((row) => (row.values ?? []).join(' | ')).join('；'),
      }));
    const params: AskAiQueryParams = { question, lang: i18n.language, withAnswer: true, history };
    setRounds((prev) => [...prev, { question, loading: true }]);
    setInput('');
    askMutation.mutate(params);
  };

  // 固定高度 + 内层消息区自滚：输入区才能常驻视口底（heightMode=auto 时容器与内容
  // 等高，sticky bottom-0 零行程失效，输入框会被排到内容流末尾）。定式同 ai/chat。
  return (
    <ContentContainer heightMode="fixed" padding="16px">
      <div className="mx-auto flex min-h-0 w-full max-w-4xl flex-1 flex-col gap-4 overflow-y-auto pb-2">
        {/* 本页品牌色一律取 antd token（--ant-color- 系列），不写 Tailwind 调色板类：
            blue-500 = #3B82F6 是 §3.2 已退役的旧主色，gray-400 = #9CA3AF 在浅底上仅 2.5:1。 */}
        {/* 标题 + 示例问题（仅首轮前展示） */}
        {rounds.length === 0 && (
          <div className="flex flex-col items-center gap-3 py-10 text-center">
            {/* 图标取内联 style：.anticon 上的 antd 规则（color: inherit）是无层的，
                会压过 @layer utilities 里的 Tailwind 颜色类——实测 text-[color:var(...)] 在
                普通 div 生效、在 .anticon 上无效。 */}
            <ThunderboltOutlined className="text-5xl" style={{ color: 'var(--ant-color-primary)' }} />
            <Typography.Title level={4} className="!mb-0">
              {t('title')}
            </Typography.Title>
            <Typography.Text type="secondary">{t('emptyDesc')}</Typography.Text>
            <div className="mt-2 flex flex-wrap justify-center gap-2">
              {SAMPLE_KEYS.map((key) => (
                <Button key={key} size="small" onClick={() => setInput(t(`samples.${key}`))}>
                  {t(`samples.${key}`)}
                </Button>
              ))}
            </div>
          </div>
        )}

        {/* 问答轮次 */}
        {rounds.map((round, i) => (
          <div key={i} className="flex flex-col gap-3">
            {/* 用户问题：右侧气泡 */}
            <div className="flex flex-row-reverse items-start gap-3">
              <div className="max-w-[80%] whitespace-pre-wrap break-words rounded-2xl rounded-tr-sm bg-[color:var(--ant-color-primary)] px-4 py-2 text-[color:var(--ant-color-text-light-solid)]">
                {round.question}
              </div>
            </div>

            {/* 结果卡片：左侧 */}
            <div className="flex items-start gap-3">
              <div className="min-w-0 flex-1 rounded-xl border border-solid border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
                {round.loading ? (
                  <div className="flex items-center gap-2 text-sm text-[color:var(--ant-color-text-secondary)]">
                    <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-[color:var(--ant-color-primary)]" />
                    {t('thinking')}
                  </div>
                ) : (
                  <>
                    {round.sql && (
                      <details className="mb-3" open>
                        <summary className="cursor-pointer text-xs text-[color:var(--ant-color-text-secondary)]">
                          {t('generatedSql')}
                        </summary>
                        <pre className="mt-2 overflow-x-auto rounded-lg bg-[color:var(--ant-color-fill-tertiary)] p-3 text-xs leading-relaxed text-[color:var(--ant-color-text)]">
                          {round.sql}
                        </pre>
                      </details>
                    )}

                    {round.columns && round.columns.length > 0 && (
                      <QueryResultCard columns={round.columns} rows={round.rows ?? []} />
                    )}
                    {round.errorMessage ? (
                      <Typography.Text type="danger" className="!text-xs">
                        {round.errorMessage}
                      </Typography.Text>
                    ) : (round.rows?.length ?? 0) === 0 ? (
                      <Typography.Text type="secondary" className="!text-xs">
                        {t('noRows')}
                      </Typography.Text>
                    ) : null}

                    {round.answer && (
                      <div className="mt-3 rounded-lg border border-solid border-[color:var(--ant-color-primary-border)] bg-[color:var(--ant-color-primary-bg)] p-3 text-sm leading-relaxed">
                        <Tag color="processing" className="mb-1">
                          {t('answerTag')}
                        </Tag>
                        <div className="markdown-body max-w-none overflow-x-auto break-words [&_li]:m-0 [&_p]:mb-1 [&_table]:w-full [&_table]:border-collapse [&_td]:border [&_td]:border-gray-300 [&_td]:px-2 [&_td]:py-1 [&_th]:border [&_th]:border-gray-300 [&_th]:px-2 [&_th]:py-1 [&_th]:bg-gray-100 dark:[&_td]:border-gray-600 dark:[&_th]:bg-gray-800 dark:[&_th]:border-gray-600">
                          <ReactMarkdown remarkPlugins={[remarkGfm]}>{round.answer}</ReactMarkdown>
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* 输入区（flex 常驻吸底，-mx-4/-mb-4 盖过容器 padding 全幅出血） */}
      <div className="-mx-4 -mb-4 shrink-0 border-t border-solid border-gray-200 bg-white/95 p-4 backdrop-blur dark:border-gray-700 dark:bg-gray-900/95">
        <div className="mx-auto flex max-w-4xl items-end gap-2">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onPressEnter={() => handleAsk()}
            placeholder={t('inputPlaceholder')}
            size="large"
            disabled={askMutation.isPending}
            allowClear
          />
          <Button
            type="primary"
            size="large"
            icon={<SendOutlined />}
            loading={askMutation.isPending}
            disabled={!input.trim()}
            onClick={() => handleAsk()}
          >
            {t('ask')}
          </Button>
        </div>
      </div>
    </ContentContainer>
  );
}

// ── 结果图表（确定性推断，零 LLM 参与） ─────────────────────────────

type ChartKind = 'bar' | 'line' | null;

/** 判断值列是否可当作数值序列（全部行都能 parseFloat 即可）。 */
function isNumericSeries(rows: aiservicev1_AiQueryRow[], colIdx: number): boolean {
  if (rows.length === 0) return false;
  return rows.every((r) => {
    const v = (r.values ?? [])[colIdx];
    return v !== undefined && v !== null && v !== '' && !Number.isNaN(Number(v));
  });
}

/** 推断图表类型：首列为文本分类 + 数值列 → 柱状（≤12 行）或折线（>12 行）；
 *  其余形状（无分类列/全文本列）返回 null 表示不适合画图。 */
function inferChart(
  columns: string[],
  rows: aiservicev1_AiQueryRow[],
): { kind: ChartKind; xIndex: number; series: number[] } | null {
  if (rows.length < 2 || columns.length < 2) return null;
  const numericCols = columns
    .map((_, i) => i)
    .filter((i) => i > 0 && isNumericSeries(rows, i));
  if (numericCols.length === 0) return null;
  const kind: ChartKind = rows.length > 12 ? 'line' : 'bar';
  return { kind, xIndex: 0, series: numericCols };
}

/** 查询结果图表：柱状/折线，x 轴取首列分类值。 */
function QueryChart({
  columns,
  rows,
  kind,
}: {
  columns: string[];
  rows: aiservicev1_AiQueryRow[];
  kind: ChartKind;
}) {
  const series = columns.map((_, i) => i).filter((i) => i > 0 && isNumericSeries(rows, i));
  // 画布文字不吃 CSS 层叠：不给颜色时 ECharts 用它自己的默认 #333 文字/浅灰线，
  // 暗色卡片（#111827）上实测只有 1.4:1。这里与 LineChart 同一口径取 antd token。
  const { token } = theme.useToken();
  const option = {
    textStyle: { color: token.colorText },
    // echarts 6：containLabel 已废弃，不加载 LegacyGridContainLabel 时整条被忽略
    // （Grid.js 直接 log 报错），等价写法是 outerBoundsMode/outerBoundsContain。
    grid: { left: 8, right: 8, top: 24, bottom: 8, outerBoundsMode: 'same', outerBoundsContain: 'axisLabel' },
    legend: { top: 0, textStyle: { color: token.colorTextSecondary } },
    tooltip: {
      trigger: 'axis',
      backgroundColor: token.colorBgElevated,
      borderColor: token.colorBorder,
      textStyle: { color: token.colorText },
    },
    xAxis: {
      type: 'category',
      data: rows.map((r) => (r.values ?? [])[0] ?? ''),
      axisLine: { lineStyle: { color: token.colorSplit } },
      axisTick: { lineStyle: { color: token.colorSplit } },
      axisLabel: { rotate: rows.length > 6 ? 30 : 0, color: token.colorTextSecondary },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: token.colorTextSecondary },
      splitLine: { lineStyle: { color: token.colorSplit } },
    },
    series: series.map((colIdx) => ({
      name: columns[colIdx],
      type: kind,
      barMaxWidth: 40,
      data: rows.map((r) => Number((r.values ?? [])[colIdx] ?? 0)),
    })),
  };
  return <ReactECharts option={option} style={{ height: 280 }} notMerge lazyUpdate />;
}

/** 查询结果卡片：图表/表格切换（数据可画图时默认图表）。 */
function QueryResultCard({
  columns,
  rows,
}: {
  columns: string[];
  rows: aiservicev1_AiQueryRow[];
}) {
  const { t } = useI18n('aiQuery');
  const [view, setView] = useState<'chart' | 'table'>('chart');
  const chart = inferChart(columns, rows);
  const chartable = chart !== null;

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        {chartable && (
          <Segmented
            size="small"
            value={view}
            onChange={(v) => setView(v as 'chart' | 'table')}
            options={[
              { label: t('viewChart'), value: 'chart' },
              { label: t('viewTable'), value: 'table' },
            ]}
          />
        )}
        <Typography.Text type="secondary" className="!text-xs">
          {t('rowCount', { count: rows.length })}
        </Typography.Text>
      </div>
      {chartable && view === 'chart' ? (
        <QueryChart columns={columns} rows={rows} kind={chart!.kind} />
      ) : (
        <QueryTable columns={columns} rows={rows} />
      )}
    </div>
  );
}

/** 查询结果表：以该轮返回的 columns 动态生成列。 */
function QueryTable({
  columns,
  rows,
}: {
  columns: string[];
  rows: aiservicev1_AiQueryRow[];
}) {
  const tableColumns: ColumnsType<{ key: number; cells: string[] }> = columns.map(
    (col, idx) => ({
      title: col,
      dataIndex: ['cells', String(idx)],
      key: col,
      ellipsis: true,
    }),
  );
  const data = rows.map((r, ri) => ({ key: ri, cells: r.values ?? [] }));

  return (
    <Table
      size="small"
      rowKey="key"
      columns={tableColumns}
      dataSource={data}
      pagination={data.length > 10 ? { pageSize: 10, showSizeChanger: false } : false}
      scroll={{ x: 'max-content' }}
    />
  );
}
