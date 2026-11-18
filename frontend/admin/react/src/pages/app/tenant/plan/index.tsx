import { useState } from 'react';

import SplitPane from '@/components/common/SplitPane';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import PlanList from './PlanList';
import PlanQuotaList from './PlanQuotaList';

/**
 * 套餐管理页面
 * 使用 SplitPane 实现左右分栏布局：左侧套餐目录，右侧套餐配额（窄容器自动上下堆叠）
 */
const PlanManagement = () => {
  const [currentPlanId, setCurrentPlanId] = useState<number | null>(null);

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <SplitPane>
        <SplitPane.Panel
          collapsible
          defaultSize="45%"
          min="25%"
          max="75%"
          style={{ display: 'flex', flexDirection: 'column' }}
        >
          <PlanList currentPlanId={currentPlanId} onPlanSelect={setCurrentPlanId} />
        </SplitPane.Panel>
        <SplitPane.Panel style={{ display: 'flex', flexDirection: 'column' }}>
          <PlanQuotaList planId={currentPlanId} />
        </SplitPane.Panel>
      </SplitPane>
    </ContentContainer>
  );
};

export default PlanManagement;
