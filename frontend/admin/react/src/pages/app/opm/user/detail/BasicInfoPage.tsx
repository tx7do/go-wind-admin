import { Avatar, Descriptions, Tag, Spin, theme } from 'antd';
import { useTranslation } from 'react-i18next';
import type { identityservicev1_User as User } from '@/api/generated/admin/service/v1';
import { useGetUser } from '@/api/hooks/user';
import { getGenderMap } from '../constants';
import { getCharColor, getRandomColor } from '@/utils/color';
import { formatDateTime } from '@/utils/date';
import { useUserStore } from '@/stores/user';
import { isFieldHidden } from '@/core/access';

interface BasicInfoPageProps {
  userId: string | undefined; // uint64 → proto3 JSON 下为字符串
}

/**
 * 用户详情 - 基本信息 Tab
 */
const BasicInfoPage: React.FC<BasicInfoPageProps> = ({ userId }) => {
  const { t } = useTranslation('user-detail');
  const { t: tUser } = useTranslation('user');
  const genderMap = getGenderMap(tUser);
  const { token } = theme.useToken();
  const hiddenFieldEntries = useUserStore((s) => s.hiddenFields);

  const { data, isLoading } = useGetUser({ id: userId }, { enabled: !!userId } as any);

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
        <Spin size="large" />
      </div>
    );
  }

  const user = data as User | undefined;
  if (!user) return null;

  const firstChar = user?.username?.slice(0, 1).toUpperCase() || '?';
  const avatarColor = getCharColor(firstChar);

  return (
    <div style={{ padding: '24px 32px' }}>
      {/* 用户头像 + 基本信息头部 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 24,
          paddingBottom: 24,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          marginBottom: 24,
        }}
      >
        <Avatar
          src={user.avatar || undefined}
          size={88}
          style={!user.avatar ? { backgroundColor: avatarColor, fontSize: 40 } : {}}
        >
          {!user.avatar && firstChar}
        </Avatar>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 20, fontWeight: 600, color: token.colorTextHeading, marginBottom: 4 }}>
            {user.realname || user.username}
          </div>
          <div style={{ color: token.colorTextSecondary, fontSize: 14, marginBottom: 8 }}>
            @{user.username}
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <Tag color={genderMap[user.gender || '']?.color || 'default'}>
              {genderMap[user.gender || '']?.text || '-'}
            </Tag>
            {(user.roleNames || []).map((role) => (
              <Tag key={role} color="purple">{role}</Tag>
            ))}
          </div>
        </div>
      </div>

      {/* 详细信息区（字段权限：被隐藏字段整项不渲染） */}
      <Descriptions column={2} size="middle" labelStyle={{ color: token.colorTextSecondary }}>
        <Descriptions.Item label={t('desc.nickname')}>{user.nickname || '-'}</Descriptions.Item>
        {!isFieldHidden(hiddenFieldEntries, 'User', 'email') && (
          <Descriptions.Item label={t('desc.email')}>{user.email || '-'}</Descriptions.Item>
        )}
        {!isFieldHidden(hiddenFieldEntries, 'User', 'mobile') && (
          <Descriptions.Item label={t('desc.mobile')}>{user.mobile || '-'}</Descriptions.Item>
        )}
        <Descriptions.Item label={t('desc.tenantName')}>{user.tenantName || '-'}</Descriptions.Item>
        <Descriptions.Item label={t('desc.orgUnitName')}>
          {(user.orgUnitNames ?? []).length > 0
            ? (user.orgUnitNames ?? []).map((org) => (
                <Tag key={org} style={{ backgroundColor: getRandomColor(org), color: '#333', border: 'none' }}>
                  {org}
                </Tag>
              ))
            : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('desc.positionName')}>
          {(user.positionNames ?? []).length > 0
            ? (user.positionNames ?? []).map((pos) => (
                <Tag key={pos} color="blue">{pos}</Tag>
              ))
            : '-'}
        </Descriptions.Item>
        {!isFieldHidden(hiddenFieldEntries, 'User', 'region') && (
          <Descriptions.Item label={t('desc.region')}>{user.region || '-'}</Descriptions.Item>
        )}
        {!isFieldHidden(hiddenFieldEntries, 'User', 'address') && (
          <Descriptions.Item label={t('desc.address')}>{user.address || '-'}</Descriptions.Item>
        )}
        <Descriptions.Item label={t('desc.createdAt')}>
          {formatDateTime(user.createdAt)}
        </Descriptions.Item>
        {!isFieldHidden(hiddenFieldEntries, 'User', 'lastLoginAt') && (
          <Descriptions.Item label={t('desc.lastLoginAt')}>
            {formatDateTime(user.lastLoginAt)}
          </Descriptions.Item>
        )}
        {!isFieldHidden(hiddenFieldEntries, 'User', 'lastLoginIp') && (
          <Descriptions.Item label={t('desc.lastLoginIp')}>
            {user.lastLoginIp || '-'}
          </Descriptions.Item>
        )}
      </Descriptions>
    </div>
  );
};

export default BasicInfoPage;
