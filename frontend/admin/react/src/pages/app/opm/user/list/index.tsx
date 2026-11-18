import { useState } from 'react';

import SplitPane from '@/components/common/SplitPane';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import OrgList from './OrgList';
import UserList from './UserList';

/**
 * 用户管理页面
 * 使用 SplitPane 实现左右分栏布局：左侧组织树，右侧用户列表（窄容器自动上下堆叠）
 */
const UserManagement = () => {
  // 选中租户/组织 id（uint64 字符串；undefined=未选）
  const [currentTenantId, setCurrentTenantId] = useState<string | undefined>(undefined);
  const [currentOrgUnitId, setCurrentOrgUnitId] = useState<string | undefined>(undefined);

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <SplitPane>
        <SplitPane.Panel
          collapsible
          defaultSize="25%"
          min="15%"
          max="40%"
          style={{ display: 'flex', flexDirection: 'column' }}
        >
          <OrgList
            currentOrgUnitId={currentOrgUnitId}
            currentTenantId={currentTenantId}
            onTenantSelect={setCurrentTenantId}
            onOrgSelect={setCurrentOrgUnitId}
          />
        </SplitPane.Panel>
        <SplitPane.Panel style={{ display: 'flex', flexDirection: 'column' }}>
          <UserList tenantId={currentTenantId} orgUnitId={currentOrgUnitId} />
        </SplitPane.Panel>
      </SplitPane>
    </ContentContainer>
  );
};

export default UserManagement;
