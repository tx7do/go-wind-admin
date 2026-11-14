import { useState } from 'react';
import { Card, Tabs } from 'antd';
import type { TabsProps } from 'antd';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@go-wind/admin-react';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import BaseSetting from './BaseSetting';
import EditPassword from './EditPassword';
import SecureSetting from './SecureSetting';
import AccountBind from './AccountBind';
import MySessions from './MySessions';
import NotificationPreference from './NotificationPreference';
import TenantUsagePanel from './TenantUsagePanel';
import { useGetUserProfile } from '@/api/hooks/user-profile';
import { useMyTenantUsage } from '@/api/hooks/my-tenant-usage';

/**
 * 个人中心页面 — 左页签栏布局（与 vben/ele 统一）
 */
const UserProfile = () => {
  const { t } = useTranslation('profile');
  const [activeKey, setActiveKey] = useState('1');

  // 获取当前用户信息
  const { data: userData } = useGetUserProfile();
  const user = userData as any;

  // 租户自助用量：租户管理员可见本租户套餐用量与配额（平台用户不展示）
  const accessStore = useAuthStore((st) => st.accessToken);
  const isTenantUser = !!accessStore && (user?.tenantId ?? 0) > 0;
  const usageQuery = useMyTenantUsage(isTenantUser);

  const handleSwitchTab = (key: string) => {
    setActiveKey(key);
  };

  const settingList: TabsProps['items'] = [
    {
      key: '1',
      label: t('tab.basicSettings'),
      children: <BaseSetting />,
    },
    {
      key: '2',
      label: t('tab.editPassword'),
      children: <EditPassword />,
    },
    {
      key: '3',
      label: t('tab.securitySettings'),
      children: <SecureSetting onSwitchTab={handleSwitchTab} />,
    },
    {
      key: '4',
      label: t('tab.accountBind'),
      children: <AccountBind onSwitchTab={handleSwitchTab} />,
    },
    {
      key: '5',
      label: t('tab.activeSessions'),
      children: <MySessions />,
    },
    {
      key: '6',
      label: t('tab.notification'),
      children: <NotificationPreference />,
    },
    ...(isTenantUser
      ? [{
          key: '7',
          label: t('tab.tenantUsage'),
          children: <TenantUsagePanel usageQuery={usageQuery} />,
        }]
      : []),
  ];

  return (
    <ContentContainer heightMode="fixed" padding="24px" scrollable>
      <Card styles={{ body: { display: 'flex', gap: 0, height: '100%' } }}>
        <Tabs
          tabPosition="left"
          activeKey={activeKey}
          onChange={setActiveKey}
          items={settingList.map((item) => ({
            ...item,
            children: (
              <div style={{ paddingLeft: 24, paddingTop: 8 }}>
                <h2 style={{ margin: '0 0 24px', fontSize: 18, fontWeight: 600 }}>{item.label}</h2>
                {item.children}
              </div>
            ),
          }))}
          style={{ width: '100%', height: '100%' }}
          tabBarStyle={{ width: 220, marginRight: 0 }}
        />
      </Card>
    </ContentContainer>
  );
};

export default UserProfile;
