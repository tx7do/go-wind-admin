import { Button, Drawer, Popconfirm, Tag, Typography } from 'antd';
import { RollbackOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type {
  aiservicev1_AiAgentApp as AiAgentApp,
} from '@/api/generated/admin/service/v1';
import { useListAiAgentRevisions, useRestoreAiAgentRevision } from '@/api/hooks/ai-agent';

/**
 * Agent 应用版本历史抽屉（docs/ai_agent_orchestration.md「版本化」，BI 同款口径）：
 * 每次更新把更新前的 graph 留一行（SAVE），恢复把快照写回主表并先以 RESTORE 留底当前版本。
 * 数据源 ListRevisions/RestoreRevision。
 */
export default function AgentRevisionsDrawer({
  app,
  open,
  onClose,
}: {
  app: AiAgentApp | undefined;
  open: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation('ai-agent');
  const [expandedId, setExpandedId] = useState<number | undefined>();

  const revisionsQuery = useListAiAgentRevisions(open ? app?.id : undefined);
  const restoreMutation = useRestoreAiAgentRevision(app?.id);

  const revisions = revisionsQuery.data?.items ?? [];
  const total = revisionsQuery.data?.total ?? 0;

  const sourceTag = (source?: string) => {
    if (source === 'RESTORE') return <Tag color="processing">{t('revisions.sourceRestore')}</Tag>;
    return <Tag color="default">{t('revisions.sourceSave')}</Tag>;
  };

  return (
    <Drawer
      title={`${t('revisions.title')} · ${app?.name ?? ''}`}
      open={open}
      onClose={onClose}
      width={640}
      destroyOnHidden
    >
      {revisions.length === 0 && !revisionsQuery.isFetching && (
        <div style={{ color: 'var(--ant-color-text-tertiary, #aaa)' }}>{t('revisions.empty')}</div>
      )}
      {revisions.map((rev) => (
        <div
          key={rev.id}
          style={{
            marginBottom: 12,
            padding: '10px 12px',
            borderRadius: 8,
            border: '1px solid var(--ant-color-border-secondary, #f0f0f0)',
          }}
        >
          <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
            {sourceTag(rev.source)}
            <span style={{ color: 'var(--ant-color-text-tertiary, #aaa)' }}>#{rev.id}</span>
            <span style={{ color: 'var(--ant-color-text-tertiary, #aaa)' }}>{rev.createdAt}</span>
            <span style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
              <a onClick={() => setExpandedId(expandedId === rev.id ? undefined : rev.id)} style={{ cursor: 'pointer' }}>
                {expandedId === rev.id ? t('revisions.collapse') : t('revisions.viewGraph')}
              </a>
              <Popconfirm title={t('revisions.restoreConfirm')} onConfirm={() => restoreMutation.mutate({ id: rev.id })}>
                <Button size="small" icon={<RollbackOutlined />}>{t('revisions.restore')}</Button>
              </Popconfirm>
            </span>
          </div>
          {expandedId === rev.id && (
            <Typography.Paragraph
              copyable
              style={{
                marginTop: 8,
                marginBottom: 0,
                fontSize: 12,
                fontFamily: 'monospace',
                whiteSpace: 'pre-wrap',
                maxHeight: 240,
                overflow: 'auto',
                background: 'var(--ant-color-fill-quaternary, #f5f5f5)',
                padding: 8,
                borderRadius: 6,
              }}
            >
              {rev.graph || ''}
            </Typography.Paragraph>
          )}
        </div>
      ))}
      {total > 0 && (
        <div style={{ color: 'var(--ant-color-text-tertiary, #aaa)', marginTop: 8, fontSize: 12 }}>
          {t('revisions.total', { total })}
        </div>
      )}
    </Drawer>
  );
}
