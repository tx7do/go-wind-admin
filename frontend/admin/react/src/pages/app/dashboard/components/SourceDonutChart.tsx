import { Card, theme } from 'antd';
import ReactECharts from 'echarts-for-react';
import { useMemo } from 'react';
import type { ActionDistributionResponse } from '@/api/generated/admin/service/v1';
import { useI18n } from '@/core/i18n';

interface SourceDonutChartProps {
  data?: ActionDistributionResponse;
}

/**
 * 操作类型分布环形图。
 * 后端返回 action 枚举名（CREATE/UPDATE/...），这里经 i18n 转成本地化文案展示，
 * 复用操作审计日志模块的 action.* 翻译。
 */
export const SourceDonutChart = ({ data }: SourceDonutChartProps) => {
  const { token } = theme.useToken();
  const { t } = useI18n('dashboard');
  const { t: tAuditLog } = useI18n('operation-audit-log');

  // 科技风暗色系调色板（靛蓝、翠绿、紫罗兰、琥珀）
  const palette = ['#6366f1', '#10b981', '#8b5cf6', '#f59e0b'];

  // 操作类型枚举名 → 本地化文案，复用操作审计日志模块的 action.* 翻译。
  const actionLabel = (label?: string): string => {
    switch (label) {
      case 'CREATE':
        return tAuditLog('action.CREATE');
      case 'UPDATE':
        return tAuditLog('action.UPDATE');
      case 'DELETE':
        return tAuditLog('action.DELETE');
      case 'READ':
        return tAuditLog('action.READ');
      case 'ASSIGN':
        return tAuditLog('action.ASSIGN');
      case 'UNASSIGN':
        return tAuditLog('action.UNASSIGN');
      case 'EXPORT':
        return tAuditLog('action.EXPORT');
      case 'IMPORT':
        return tAuditLog('action.IMPORT');
      case 'OTHER':
        return tAuditLog('action.OTHER');
      default:
        return label ?? '';
    }
  };

  const option = useMemo(() => {
    const items = data?.items ?? [];
    return {
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
        backgroundColor: 'rgba(20,20,30,0.95)',
        borderColor: 'rgba(255,255,255,0.1)',
        textStyle: { color: '#e2e8f0' },
      },
      legend: {
        orient: 'horizontal',
        bottom: 0,
        data: items.map((it) => actionLabel(it.label)),
        textStyle: {
          color: '#94a3b8',
          fontSize: 12,
        },
        itemWidth: 10,
        itemHeight: 10,
        itemGap: 16,
      },
      series: [
        {
          name: t('charts.operationActionDistribution'),
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['50%', '48%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 10,
            borderColor: token.colorBgContainer,
            borderWidth: 2,
          },
          label: {
            show: false,
            position: 'center',
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 20,
              fontWeight: 'bold',
            },
            scaleSize: 8,
          },
          labelLine: {
            show: false,
          },
          data: items.map((it, i) => ({
            value: it.count,
            name: actionLabel(it.label),
            itemStyle: { color: palette[i % palette.length] },
          })),
        },
      ],
    };
  }, [data, token, t, tAuditLog]);

  return (
    <Card title={t('charts.operationActionDistribution')} style={{ height: '100%' }}>
      <ReactECharts option={option} style={{ height: 280 }} />
    </Card>
  );
};
