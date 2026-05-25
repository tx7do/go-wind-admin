import { Row, Col } from 'antd';
import {
  UserOutlined,
  EyeOutlined,
  DownloadOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import {
  StatsCard,
  TrendChart,
  VisitRadarChart,
  SourceDonutChart,
  SourcePieChart,
} from './components';

const Dashboard = () => {
  const { t } = useTranslation('dashboard');

  // 统计卡片数据
  const statsData = [
    {
      title: t('stats.users'),
      value: '2,000',
      total: t('stats.totalUsers'),
      totalValue: '120,000',
      icon: <UserOutlined style={{ fontSize: 24, color: '#1677ff' }} />,
    },
    {
      title: t('stats.visits'),
      value: '20,000',
      total: t('stats.totalVisits'),
      totalValue: '500,000',
      icon: <EyeOutlined style={{ fontSize: 24, color: '#ff7a45' }} />,
    },
    {
      title: t('stats.downloads'),
      value: '8,000',
      total: t('stats.totalDownloads'),
      totalValue: '120,000',
      icon: <DownloadOutlined style={{ fontSize: 24, color: '#faad14' }} />,
    },
    {
      title: t('stats.usage'),
      value: '5,000',
      total: t('stats.totalUsage'),
      totalValue: '50,000',
      icon: <ClockCircleOutlined style={{ fontSize: 24, color: '#52c41a' }} />,
    },
  ];

  return (
    <ContentContainer heightMode="auto" scrollable padding="16px">
      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        {statsData.map((stat, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <StatsCard
              title={stat.title}
              value={stat.value}
              total={stat.total}
              totalValue={stat.totalValue}
              icon={stat.icon}
            />
          </Col>
        ))}
      </Row>

      {/* 访问趋势图 */}
      <TrendChart />

      {/* 第三行：访问数量和来源分析 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={8}>
          <VisitRadarChart />
        </Col>
        <Col xs={24} lg={8}>
          <SourceDonutChart />
        </Col>
        <Col xs={24} lg={8}>
          <SourcePieChart />
        </Col>
      </Row>
    </ContentContainer>
  );
};

export default Dashboard;
