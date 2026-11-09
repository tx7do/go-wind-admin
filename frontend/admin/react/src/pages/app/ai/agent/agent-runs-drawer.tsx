import { useState } from 'react';
import {
  Drawer,
  Tag,
  Timeline,
  Tooltip,
} from 'antd';
import { useTranslation } from 'react-i18next';
import type {
  aiservicev1_AiAgentApp as AiAgentApp,
} from '@/api/generated/admin/service/v1';
import { useListAiAgentRuns } from '@/api/hooks/ai-agent-run';
import { PaginationQuery } from '@/core';

/**
 * Agent 应用运行台账抽屉：该应用的历次试运行记录
 * （状态/入参/产出/错误/逐节点轨迹/token 用量）。数据源 ListRuns/GetRun。
 */
export default function AgentRunsDrawer({
  app,
  open,
  onClose,
}: {
  app: AiAgentApp | undefined;
  open: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation('ai-agent');
  const [page, setPage] = useState(1);
  const pageSize = 10;

  const runsQuery = useListAiAgentRuns(app?.id, new PaginationQuery({ paging: { page, pageSize } }));
  // 抽屉关闭时不发查询（enabled 由 hook 的 appId 参数控制，这里叠加 open 状态）

  const runs = runsQuery.data?.items ?? [];
  const total = Number(runsQuery.data?.total ?? 0); // uint64 → string，转回 number 供比较/展示

  const statusTag = (status?: string) => {
    if (status === 'SUCCEEDED') return <Tag color="success">{t('runs.succeeded')}</Tag>;
    if (status === 'FAILED') return <Tag color="error">{t('runs.failed')}</Tag>;
    return <Tag color="processing">{t('runs.running')}</Tag>;
  };

  const parseTrace = (raw?: string): Array<Record<string, any>> => {
    if (!raw) return [];
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  };

  return (
    <Drawer
      title={`${t('runs.title')} · ${app?.name ?? ''}`}
      open={open}
      onClose={onClose}
      width={640}
      destroyOnHidden
    >
      {runs.length === 0 && !runsQuery.isFetching && (
        <div style={{ color: 'var(--ant-color-text-tertiary, #aaa)' }}>{t('runs.empty')}</div>
      )}
      {runs.map((run) => {
        const trace = parseTrace(run.trace);
        return (
          <div
            key={run.id}
            style={{
              marginBottom: 12,
              padding: '10px 12px',
              borderRadius: 8,
              border: '1px solid var(--ant-color-border-secondary, #f0f0f0)',
            }}
          >
            <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
              {statusTag(run.status)}
              <span style={{ color: 'var(--ant-color-text-tertiary, #aaa)' }}>#{run.id}</span>
              {run.totalTokens != null && run.totalTokens > 0 && (
                <Tag>{t('runs.tokens', { total: run.totalTokens })}</Tag>
              )}
              {run.durationMs != null && <Tag>{run.durationMs}ms</Tag>}
            </div>
            {run.output && (
              <div
                style={{
                  marginTop: 6,
                  fontSize: 12,
                  whiteSpace: 'pre-wrap',
                  maxHeight: 72,
                  overflow: 'hidden',
                }}
              >
                {run.output}
              </div>
            )}
            {run.error && (
              <Tooltip title={run.error}>
                <div
                  style={{
                    marginTop: 6,
                    fontSize: 12,
                    color: 'var(--ant-color-error, #cf1322)',
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}
                >
                  {run.error}
                </div>
              </Tooltip>
            )}
            {trace.length > 0 && (
              <Timeline
                style={{ marginTop: 10 }}
                items={trace.map((s: any) => ({
                  color: s.status === 'ok' ? 'green' : 'red',
                  children: (
                    <div style={{ fontSize: 12 }}>
                      <code>{s.nodeId}</code> <Tag>{s.type}</Tag> {s.ms}ms
                    </div>
                  ),
                }))}
              />
            )}
          </div>
        );
      })}
      {total > pageSize && (
        <div style={{ textAlign: 'center', marginTop: 8 }}>
          {page * pageSize < total && (
            <a onClick={() => setPage(page + 1)} style={{ cursor: 'pointer', marginRight: 16 }}>
              {t('runs.loadMore')}
            </a>
          )}
          <span style={{ color: 'var(--ant-color-text-tertiary, #aaa)' }}>
            {t('runs.total', { total })}
          </span>
        </div>
      )}
    </Drawer>
  );
}

import { useEffect } from 'react';
import { globalSSEClient, SSE_EVENT } from '@/core';

/**
 * 订阅本用户流上的 ai_agent_run 帧（试运行实时观感）。
 * 帧形：AgentRunEvent protojson（runId/appId/seq/phase/nodeId/summary/status）。
 * 丢帧可容忍：Run 同步响应携带完整 trace，以响应为准。
 */
export function useAgentRunStream(
  appId: string | undefined,
  onFrame: (frame: {
    // runId/appId 为 uint64（protojson 下是字符串），seq 为 int64 同理
    runId: string;
    appId: string;
    seq: string | number;
    phase: string;
    nodeId?: string;
    summary?: string;
    status?: string;
  }) => void,
  enabled = true,
) {
  useEffect(() => {
    if (!enabled || !appId) return;
    const handler = (data: unknown) => {
      const frame = data as { appId?: string };
      if (!frame || typeof frame !== 'object' || frame.appId !== appId) return;
      onFrame(frame as Parameters<typeof onFrame>[0]);
    };
    globalSSEClient.on(SSE_EVENT.AgentRun, handler);
    return () => {
      globalSSEClient.off(SSE_EVENT.AgentRun, handler);
    };
  }, [appId, enabled, onFrame]);
}
