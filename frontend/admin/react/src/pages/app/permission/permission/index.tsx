import { useState } from 'react';

import SplitPane from '@/components/common/SplitPane';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import PermissionGroupList from './PermissionGroupList';
import PermissionList from './PermissionList';

/**
 * 权限点管理页面
 * 使用 SplitPane 实现左右分栏布局（窄容器自动上下堆叠）
 */
const PermissionPointManagement = () => {
  const [currentGroupId, setCurrentGroupId] = useState<number | null>(null);

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <SplitPane>
        <SplitPane.Panel
          collapsible
          defaultSize="40%"
          min="25%"
          max="55%"
          style={{ display: 'flex', flexDirection: 'column' }}
        >
          <PermissionGroupList currentGroupId={currentGroupId} onGroupSelect={setCurrentGroupId} />
        </SplitPane.Panel>
        <SplitPane.Panel style={{ display: 'flex', flexDirection: 'column' }}>
          <PermissionList groupId={currentGroupId} />
        </SplitPane.Panel>
      </SplitPane>
    </ContentContainer>
  );
};

export default PermissionPointManagement;
