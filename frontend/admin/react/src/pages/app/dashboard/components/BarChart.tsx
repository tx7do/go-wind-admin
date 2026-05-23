import ReactECharts from 'echarts-for-react';
import { usePreferences } from '@/core/preferences';
import { useI18n } from '@/core/i18n';

/**
 * 月访问量柱状图组件
 */
export const BarChart = () => {
  const { isDark } = usePreferences();
  const { t } = useI18n('dashboard');

  const option = {
    tooltip: {
      trigger: 'axis',
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: [
        t('months.jan'),
        t('months.feb'),
        t('months.mar'),
        t('months.apr'),
        t('months.may'),
        t('months.jun'),
        t('months.jul'),
        t('months.aug'),
        t('months.sep'),
        t('months.oct'),
        t('months.nov'),
        t('months.dec'),
      ],
      axisLine: {
        lineStyle: {
          color: isDark ? 'rgba(255, 255, 255, 0.2)' : 'rgba(0, 0, 0, 0.1)',
        },
      },
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: isDark ? 'rgba(255, 255, 255, 0.2)' : 'rgba(0, 0, 0, 0.1)',
        },
      },
      splitLine: {
        lineStyle: {
          color: isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.05)',
        },
      },
    },
    series: [
      {
        name: t('data.monthlyVisitCount'),
        type: 'bar',
        itemStyle: {
          color: '#1677ff',
        },
        data: [3000, 2000, 3300, 5000, 3200, 4200, 3200, 2100, 3000, 5100, 6000, 3200],
      },
    ],
  };

  return <ReactECharts option={option} style={{ height: 300 }} />;
};
