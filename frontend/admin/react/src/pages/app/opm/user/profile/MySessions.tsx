import { Table, Tag, Typography, App, Popconfirm, Button } from 'antd';
import type { TableColumnsType } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  useListMyOnlineSessions,
  useRevokeMyOnlineSession,
} from '@/api/hooks/online-session';
import type { online_sessionservicev1_OnlineSession as OnlineSession } from '@/api/generated/admin/service/v1';
import { formatDateTime } from '@go-wind/admin-react';

const { Text } = Typography;

/**
 * 个人中心「活跃会话」：当前登录用户自己的全部在线会话自助管理。
 * 当前会话仅展示不可踢除（踢除会立即登出）；其它会话可单独下线。
 */
const MySessions = () => {
  const { t } = useTranslation('online-session');
  const { message } = App.useApp();
  const { data, isLoading } = useListMyOnlineSessions();
  const revokeMutation = useRevokeMyOnlineSession({
    onSuccess: () => {
      message.success(t('revokeSuccess'));
    },
  });

  const handleRevoke = async (record: OnlineSession) => {
    if (!record.jti) return;
    try {
      await revokeMutation.mutateAsync({
        clientType: record.clientType,
        jti: record.jti,
      });
    } catch (error: any) {
      console.error('Failed to revoke session:', error);
      message.error(error.message || t('revokeFailed'));
    }
  };

  const columns: TableColumnsType<OnlineSession> = [
    {
      title: t('clientType'),
      dataIndex: 'clientType',
      width: 110,
      render: (_, record) =>
        record.clientType === 'app' ? (
          <Tag color="purple">{t('clientApp')}</Tag>
        ) : (
          <Tag color="blue">{t('clientAdmin')}</Tag>
        ),
    },
    {
      title: t('ipAddress'),
      dataIndex: 'ipAddress',
      width: 140,
      render: (val) => val || '-',
    },
    {
      title: t('userAgent'),
      dataIndex: 'userAgent',
      ellipsis: true,
      render: (val) => val || '-',
    },
    {
      title: t('loginAt'),
      dataIndex: 'loginAt',
      width: 170,
      render: (val) => formatDateTime(val as any),
    },
    {
      title: t('status'),
      width: 140,
      render: (_, record) =>
        record.current ? (
          <Tag color="green">{t('currentSession')}</Tag>
        ) : (
          <Popconfirm
            title={t('revokeConfirm')}
            okButtonProps={{ loading: revokeMutation.isPending }}
            onConfirm={() => handleRevoke(record)}
          >
            <Button danger type="link" size="small">
              {t('revoke')}
            </Button>
          </Popconfirm>
        ),
    },
  ];

  return (
    <div>
      <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 8 }}>
        {t('mySessionsHint')}
      </Text>
      <Table<OnlineSession>
        rowKey={(record) => `${record.clientType}-${record.jti}`}
        dataSource={data?.items ?? []}
        loading={isLoading}
        pagination={false}
        size="small"
        columns={columns}
        locale={{ emptyText: t('noSessions') }}
      />
    </div>
  );
};

export default MySessions;
