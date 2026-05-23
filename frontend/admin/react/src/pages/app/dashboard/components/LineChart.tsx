import ReactECharts from 'echarts-for-react';
import { usePreferences } from '@/core/preferences';
import { useI18n } from '@/core/i18n';

/**
 * 访问趋势折线图组件
 */
export const LineChart = () => {
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
      boundaryGap: false,
      data: [
        '6:00',
        '7:00',
        '8:00',
        '9:00',
        '10:00',
        '11:00',
        '12:00',
        '13:00',
        '14:00',
        '15:00',
        '16:00',
        '17:00',
        '18:00',
        '19:00',
        '20:00',
        '21:00',
        '22:00',
        '23:00',
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
        name: t('data.visit'),
        type: 'line',
        smooth: true,
        areaStyle: {
          opacity: 0.3,
          color: '#1677ff',
        },
        lineStyle: {
          width: 2,
          color: '#1677ff',
        },
        itemStyle: {
          color: '#1677ff',
        },
        data: [
          0, 5000, 15000, 25000, 40000, 55000, 65000, 45000, 20000, 35000, 50000, 70000, 45000,
          25000, 15000, 8000, 4000, 2000,
        ],
      },
      {
        name: t('data.trend'),
        type: 'line',
        smooth: true,
        areaStyle: {
          opacity: 0.3,
          color: '#52c41a',
        },
        lineStyle: {
          width: 2,
          color: '#52c41a',
        },
        itemStyle: {
          color: '#52c41a',
        },
        data: [
          0, 1000, 3000, 8000, 15000, 20000, 22000, 15000, 8000, 12000, 18000, 23000, 15000, 8000,
          4000, 2000, 1000, 500,
        ],
      },
    ],
  };

  return <ReactECharts option={option} style={{ height: 300 }} />;
};
