import { Card } from 'antd';
import { usePreferences } from '@/core/preferences';
import type { ReactNode } from 'react';

interface StatsCardProps {
  title: string;
  value: string;
  total: string;
  totalValue: string;
  icon: ReactNode;
}

/**
 * 统计卡片组件
 */
export const StatsCard = ({ title, value, total, totalValue, icon }: StatsCardProps) => {
  const { isDark } = usePreferences();

  return (
    <Card style={{ height: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <div
            style={{
              fontSize: 14,
              color: isDark ? 'rgba(255, 255, 255, 0.65)' : 'rgba(0, 0, 0, 0.65)',
              marginBottom: 8,
            }}
          >
            {title}
          </div>
          <div style={{ fontSize: 24, fontWeight: 600, marginBottom: 16 }}>{value}</div>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
            <span style={{ color: isDark ? 'rgba(255, 255, 255, 0.45)' : 'rgba(0, 0, 0, 0.45)' }}>
              {total}
            </span>
            <span style={{ color: isDark ? 'rgba(255, 255, 255, 0.65)' : 'rgba(0, 0, 0, 0.65)' }}>
              {totalValue}
            </span>
          </div>
        </div>
        <div>{icon}</div>
      </div>
    </Card>
  );
};
