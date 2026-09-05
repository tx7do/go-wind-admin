import { Card } from 'antd';
import type { LoginTrendResponse } from '@/api/generated/admin/service/v1';
import { useI18n } from '@/core/i18n';
import { LineChart } from './LineChart';

interface TrendChartProps {
  data?: LoginTrendResponse;
}

/**
 * 登录趋势图组件（单折线，数据由父组件从后端 GetLoginTrend 拉取后下发）
 */
export const TrendChart = ({ data }: TrendChartProps) => {
  const { t } = useI18n('dashboard');

  return (
    <Card title={t('charts.loginTrend')} style={{ marginTop: 16 }} styles={{ body: { padding: 16 } }}>
      <LineChart data={data} />
    </Card>
  );
};
