import { Card } from 'antd';
import ReactECharts from 'echarts-for-react';
import { usePreferences } from '@/core/preferences';
import { useI18n } from '@/core/i18n';

/**
 * 访问数量雷达图组件
 */
export const VisitRadarChart = () => {
  const { isDark } = usePreferences();
  const { t } = useI18n('dashboard');

  const option = {
    tooltip: {
      trigger: 'item',
    },
    legend: {
      data: [t('data.visit'), t('data.trend')],
      bottom: 0,
      textStyle: {
        color: isDark ? 'rgba(255, 255, 255, 0.65)' : 'rgba(0, 0, 0, 0.65)',
      },
    },
    radar: {
      indicator: [
        { name: t('devices.webpage'), max: 100 },
        { name: t('devices.other'), max: 100 },
        { name: t('devices.thirdParty'), max: 100 },
        { name: t('devices.client'), max: 100 },
        { name: t('devices.ipad'), max: 100 },
        { name: t('devices.mobile'), max: 100 },
      ],
      axisName: {
        color: isDark ? 'rgba(255, 255, 255, 0.65)' : 'rgba(0, 0, 0, 0.65)',
      },
      splitLine: {
        lineStyle: {
          color: isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)',
        },
      },
      splitArea: {
        show: false,
      },
      axisLine: {
        lineStyle: {
          color: isDark ? 'rgba(255, 255, 255, 0.2)' : 'rgba(0, 0, 0, 0.1)',
        },
      },
    },
    series: [
      {
        name: t('charts.visitCount'),
        type: 'radar',
        data: [
          {
            value: [42, 30, 20, 35, 50, 60],
            name: t('data.visit'),
            itemStyle: {
              color: '#a78bfa',
            },
            areaStyle: {
              opacity: 0.3,
            },
          },
          {
            value: [50, 40, 35, 45, 60, 70],
            name: t('data.trend'),
            itemStyle: {
              color: '#3b82f6',
            },
            areaStyle: {
              opacity: 0.3,
            },
          },
        ],
      },
    ],
  };

  return (
    <Card title={t('charts.visitCount')} style={{ height: '100%' }}>
      <ReactECharts option={option} style={{ height: 280 }} />
    </Card>
  );
};
