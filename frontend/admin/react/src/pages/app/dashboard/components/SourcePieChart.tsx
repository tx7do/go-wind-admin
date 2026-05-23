import { Card } from 'antd';
import ReactECharts from 'echarts-for-react';
import { usePreferences } from '@/core/preferences';
import { useI18n } from '@/core/i18n';

/**
 * 访问来源饼图组件
 */
export const SourcePieChart = () => {
  const { isDark } = usePreferences();
  const { t } = useI18n('dashboard');

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)',
    },
    series: [
      {
        name: t('charts.visitSource'),
        type: 'pie',
        radius: ['20%', '70%'],
        center: ['50%', '50%'],
        roseType: 'area',
        itemStyle: {
          borderRadius: 8,
          borderColor: isDark ? '#1f1f1f' : '#ffffff',
          borderWidth: 2,
        },
        label: {
          color: isDark ? 'rgba(255, 255, 255, 0.85)' : 'rgba(0, 0, 0, 0.85)',
        },
        labelLine: {
          lineStyle: {
            color: isDark ? 'rgba(255, 255, 255, 0.45)' : 'rgba(0, 0, 0, 0.45)',
          },
        },
        data: [
          { value: 40, name: t('categories.outsourcing'), itemStyle: { color: '#2dd4bf' } },
          { value: 30, name: t('categories.remote'), itemStyle: { color: '#5eead4' } },
          { value: 20, name: t('categories.customization'), itemStyle: { color: '#a78bfa' } },
          { value: 15, name: t('categories.technicalSupport'), itemStyle: { color: '#3b82f6' } },
        ],
      },
    ],
  };

  return (
    <Card title={t('charts.visitSource')} style={{ height: '100%' }}>
      <ReactECharts option={option} style={{ height: 280 }} />
    </Card>
  );
};
