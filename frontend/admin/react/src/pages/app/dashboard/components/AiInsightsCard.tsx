import { useState } from 'react';
import { Button, Skeleton, Tag, Typography, App } from 'antd';
import { RobotOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useMutation } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { useI18n } from '@go-wind/admin-react';
import type { AiInsightAlert, AiSensitiveOpItem } from '@/api/generated/admin/service/v1';

/** 严重度 → 左边条/标签色（语义一致：红=高、橙=中、蓝=低）。
 *  条色取 antd 语义 token，与 ele 端 `.sev-*` 规则同一映射（danger/warning/primary）；
 *  原先的 red/orange/blue-500 是 Tailwind 调色板， hues 与 §2.1 语义色不一致。
 *  tagColor 保持预设名：那是 antd Tag 库内预设，非本页色值。 */
const SEVERITY_STYLE: Record<string, { bar: string; tagColor: string }> = {
  HIGH: { bar: 'bg-[color:var(--ant-color-error)]', tagColor: 'red' },
  MEDIUM: { bar: 'bg-[color:var(--ant-color-warning)]', tagColor: 'orange' },
  LOW: { bar: 'bg-[color:var(--ant-color-primary)]', tagColor: 'blue' },
};

/** 告警 type → i18n title/detail 模板键（文案在 dashboard.json，后端只回结构化事实）。 */
const ALERT_I18N_KEY: Record<string, { title: string; detail?: string }> = {
  BRUTE_FORCE: { title: 'alerts.bruteForce.title', detail: 'alerts.bruteForce.detail' },
  NIGHT_OPS: { title: 'alerts.nightOps.title', detail: 'alerts.nightOps.detail' },
  FAILED_OPS: { title: 'alerts.failedOps.title', detail: 'alerts.failedOps.detail' },
  SENSITIVE_OPS: { title: 'alerts.sensitiveOps.title' },
};

/**
 * 安全与异常洞察卡片（dashboard）：
 * 后端规则预筛审计明细里的行为模式（疑似口令尝试 / 深夜操作 / 失败集中 / 敏感操作），
 * 只回传结构化事实（severity + type + facts）；文案由前端 i18n 模板按界面语言渲染，
 * LLM 总评按请求 lang 生成——保证中英文界面内容一致。
 * 平台用户专属（后端对租户返回 403——明细日志为全平台数据且会外发到模型端点）。
 */
const AiInsightsCard = () => {
  const { t, i18n } = useI18n('dashboard');
  const { message } = App.useApp();
  const [alerts, setAlerts] = useState<AiInsightAlert[] | null>(null);
  const [summary, setSummary] = useState('');

  const insightsMutation = useMutation({
    mutationFn: () => apiClient.dashboardService.GetAiInsights({ lang: i18n.language }),
    onSuccess: (resp) => {
      setAlerts(resp.alerts ?? []);
      setSummary(resp.summary ?? '');
    },
    onError: (error: Error) => {
      console.error('dashboard ai insights failed:', error);
      message.error(error.message || t('aiInsights.failed'));
    },
  });

  const renderAlertText = (alert: AiInsightAlert, kind: 'title' | 'detail') => {
    const keys = ALERT_I18N_KEY[alert.type ?? ''];
    if (!keys || !keys[kind]) return alert.facts?.detail ?? '';
    return t(`aiInsights.${keys[kind]}`, alert.facts ?? {});
  };

  return (
    <div
      className={
        'group relative h-full overflow-hidden rounded-xl border ' +
        'border-[color:var(--ant-color-border-secondary)] ' +
        'bg-[color:var(--ant-color-bg-container)] p-5 transition-all duration-200 ' +
        'hover:border-[color:var(--ant-color-primary-border-hover)] ' +
        'hover:shadow-[0_8px_24px_rgba(0,0,0,0.08)] dark:hover:shadow-[0_8px_24px_rgba(0,0,0,0.35)]'
      }
    >
      {/* 左上角主色微光，与 StatsCard 同一装饰语言 */}
      <div
        className={
          'pointer-events-none absolute -top-10 -left-10 h-28 w-28 rounded-full ' +
          'bg-gradient-to-br from-blue-500/10 to-transparent opacity-60 blur-2xl'
        }
      />

      {/* 头部 */}
      <div className="relative mb-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-500/12 text-blue-600 dark:text-blue-400">
            <RobotOutlined style={{ fontSize: 18 }} />
          </div>
          <div>
            <div className="text-sm font-semibold text-[color:var(--ant-color-text)]">
              {t('aiInsights.title')}
            </div>
            <div className="text-xs text-[color:var(--ant-color-text-secondary)]">
              {t('aiInsights.subtitle')}
            </div>
          </div>
        </div>
        <Button
          type="primary"
          icon={<ThunderboltOutlined />}
          loading={insightsMutation.isPending}
          onClick={() => insightsMutation.mutate()}
        >
          {alerts !== null ? t('aiInsights.regenerate') : t('aiInsights.generate')}
        </Button>
      </div>

      {insightsMutation.isPending && alerts === null ? (
        <Skeleton active paragraph={{ rows: 4 }} />
      ) : alerts === null ? (
        <Typography.Text type="secondary">{t('aiInsights.empty')}</Typography.Text>
      ) : alerts.length === 0 ? (
        <Typography.Text type="secondary">{t('aiInsights.noAnomaly')}</Typography.Text>
      ) : (
        <>
          {/* 告警条目：severity 左边条 + Tag + i18n 模板标题/明细 */}
          <div className="relative space-y-3">
            {alerts.map((alert, i) => {
              const style = SEVERITY_STYLE[alert.severity ?? ''] ?? SEVERITY_STYLE.LOW!;
              return (
                <div
                  key={i}
                  className="flex items-stretch gap-3 rounded-lg border border-solid border-[color:var(--ant-color-border-secondary)] p-3"
                >
                  <span className={`w-1 shrink-0 rounded-full ${style.bar}`} />
                  <div className="min-w-0">
                    <div className="mb-0.5 flex flex-wrap items-center gap-2">
                      <Tag color={style.tagColor} className="!m-0">
                        {alert.severity}
                      </Tag>
                      <span className="text-sm font-medium text-[color:var(--ant-color-text)]">
                        {renderAlertText(alert, 'title')}
                      </span>
                    </div>
                    {ALERT_I18N_KEY[alert.type ?? '']?.detail && (
                      <div className="text-xs leading-relaxed text-[color:var(--ant-color-text-secondary)]">
                        {renderAlertText(alert, 'detail')}
                      </div>
                    )}
                    {/* SENSITIVE_OPS 明细行（数据本身：用户/动作/资源/时间） */}
                    {(alert.items ?? []).length > 0 && (
                      <div className="mt-1 space-y-0.5">
                        {(alert.items ?? []).map((item: AiSensitiveOpItem, j: number) => (
                          <div
                            key={j}
                            className="text-xs text-[color:var(--ant-color-text-secondary)]"
                          >
                            {item.createdAt} · {item.username} · {item.action} · {item.resourceType}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* LLM 总体评估（按界面语言生成；告警为确定性事实，此处仅为措辞归纳） */}
          {summary && (
            <div className="mt-3 border-t border-solid border-[color:var(--ant-color-border-secondary)] pt-3">
              <Typography.Text type="secondary" className="!text-xs">
                {t('aiInsights.assessment')}
              </Typography.Text>
              <div className="mt-1 text-sm leading-relaxed text-[color:var(--ant-color-text)]">
                {summary}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default AiInsightsCard;
