import { useState } from 'react';
import { Button, Tabs, Popconfirm, App, theme } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useUpdateUser } from '@/api/hooks/user';
import { queryClient } from '@/core';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { TabEnum } from './types';
import BasicInfoPage from './BasicInfoPage';
import ApiLogPage from './ApiLogPage';
import InternalMessagePage from './InternalMessagePage';
import EditPasswordModal from './EditPasswordModal';

/**
 * 用户详情页面
 */
const UserDetail = () => {
  const { t } = useTranslation('user-detail');
  const navigate = useNavigate();
  const { message } = App.useApp();

  const { id } = useParams<{ id: string }>();
  // 路由参数本就是字符串，ID 为 uint64 字符串直接透传，无需数值转换
  const userId = id || undefined;

  const [activeTab, setActiveTab] = useState<string>(TabEnum.BASIC_INFO);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);

  // 禁用账户
  const banMutation = useUpdateUser({
    onSuccess: () => {
      message.success(t('banSuccess'));
      queryClient.invalidateQueries({ queryKey: ['getUser'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('banFailed'));
    },
  });

  const handleBanAccount = () => {
    if (userId) {
      banMutation.mutate({ id: userId, values: { status: 'DISABLED' } });
    }
  };

  const { token } = theme.useToken();

  const goBack = () => {
    navigate('/opm/users');
  };

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      {/* 内容卡片 */}
      <div
        style={{
          flex: 1,
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          borderRadius: token.borderRadiusLG,
          border: `1px solid ${token.colorBorderSecondary}`,
          backgroundColor: token.colorBgContainer,
          overflow: 'hidden',
        }}
      >
        {/* 顶部操作栏 */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px 20px',
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Button type="text" icon={<ArrowLeftOutlined />} onClick={goBack} />
            <span style={{ fontSize: 16, fontWeight: 600, color: token.colorTextHeading }}>
              {t('title')}
            </span>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <Popconfirm
              title={t('banConfirmTitle')}
              description={t('banConfirmDesc')}
              onConfirm={handleBanAccount}
              okText={t('common:button.ok')}
              cancelText={t('common:button.cancel')}
            >
              <Button danger>{t('button.banAccount')}</Button>
            </Popconfirm>
            <Button type="primary" onClick={() => setPasswordModalOpen(true)}>
              {t('button.editPassword')}
            </Button>
          </div>
        </div>

        {/* Tabs 导航 */}
        <div style={{ padding: '0 20px', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            style={{ marginBottom: 0 }}
            items={[
              { key: TabEnum.BASIC_INFO, label: t('tab.basicInfo') },
              { key: TabEnum.API_AUDIT_LOG, label: t('tab.apiAuditLog') },
              { key: TabEnum.INTERNAL_MESSAGE, label: t('tab.internalMessage') },
            ]}
          />
        </div>

        {/* Tab 内容区 */}
        <div style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
          {activeTab === TabEnum.BASIC_INFO && <BasicInfoPage userId={userId} />}
          {activeTab === TabEnum.API_AUDIT_LOG && <ApiLogPage userId={userId} />}
          {activeTab === TabEnum.INTERNAL_MESSAGE && <InternalMessagePage userId={userId} />}
        </div>
      </div>

      {/* 修改密码弹窗 */}
      <EditPasswordModal
        open={passwordModalOpen}
        userId={userId}
        onClose={() => setPasswordModalOpen(false)}
      />
    </ContentContainer>
  );
};

export default UserDetail;
