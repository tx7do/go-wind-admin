import { Card } from 'antd';
import ReactECharts from 'echarts-for-react';
import { usePreferences } from '@/core/preferences';
import { useI18n } from '@/core/i18n';

/**
 * 访问来源环形图组件
 */
export const SourceDonutChart = () => {
  const { isDark } = usePreferences();
  const { t } = useI18n('dashboard');

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)',
    },
    legend: {
      orient: 'horizontal',
      bottom: 0,
      data: [
        t('sources.searchEngine'),
        t('sources.directAccess'),
        t('sources.emailMarketing'),
        t('sources.allianceAdvertising'),
      ],
      textStyle: {
        color: isDark ? 'rgba(255, 255, 255, 0.65)' : 'rgba(0, 0, 0, 0.65)',
      },
    },
    series: [
      {
        name: t('charts.visitSource'),
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: isDark ? '#1f1f1f' : '#ffffff',
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
        },
        labelLine: {
          show: false,
        },
        data: [
          { value: 1048, name: t('sources.searchEngine'), itemStyle: { color: '#3b82f6' } },
          { value: 735, name: t('sources.directAccess'), itemStyle: { color: '#a78bfa' } },
          { value: 580, name: t('sources.emailMarketing'), itemStyle: { color: '#2dd4bf' } },
          { value: 484, name: t('sources.allianceAdvertising'), itemStyle: { color: '#14b8a6' } },
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
