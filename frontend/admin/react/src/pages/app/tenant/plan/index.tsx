import { Splitter } from 'antd';
import { useState } from 'react';

import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import PlanList from './PlanList';
import PlanQuotaList from './PlanQuotaList';

/**
 * 套餐管理页面
 * 使用 Splitter 实现左右分栏布局：左侧套餐目录，右侧套餐配额
 */
const PlanManagement = () => {
  const [currentPlanId, setCurrentPlanId] = useState<number | null>(null);

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <Splitter style={{ height: '100%', flex: 1, minHeight: 0 }}>
        <Splitter.Panel collapsible defaultSize="45%" min="25%" max="75%" style={{ display: 'flex', flexDirection: 'column' }}>
          <PlanList currentPlanId={currentPlanId} onPlanSelect={setCurrentPlanId} />
        </Splitter.Panel>
        <Splitter.Panel style={{ display: 'flex', flexDirection: 'column' }}>
          <PlanQuotaList planId={currentPlanId} />
        </Splitter.Panel>
      </Splitter>
    </ContentContainer>
  );
};

export default PlanManagement;
