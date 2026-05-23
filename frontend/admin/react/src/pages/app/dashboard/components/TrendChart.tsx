import { Card, Tabs } from 'antd';
import { useState } from 'react';
import type { TabsProps } from 'antd';
import { useI18n } from '@/core/i18n';
import { LineChart } from './LineChart';
import { BarChart } from './BarChart';

/**
 * 访问趋势图组件（支持折线图和柱状图切换）
 */
export const TrendChart = () => {
  const { t } = useI18n('dashboard');
  const [activeTab, setActiveTab] = useState<'trend' | 'monthly'>('trend');

  const tabItems: TabsProps['items'] = [
    {
      key: 'trend',
      label: t('charts.visitTrend'),
      children: <LineChart />,
    },
    {
      key: 'monthly',
      label: t('charts.monthlyVisits'),
      children: <BarChart />,
    },
  ];

  return (
    <Card style={{ marginTop: 16 }} styles={{ body: { padding: 0 } }}>
      <Tabs
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key as 'trend' | 'monthly')}
        items={tabItems}
        style={{ padding: '16px 16px 0' }}
      />
    </Card>
  );
};
