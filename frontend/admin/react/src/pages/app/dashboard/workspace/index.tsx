import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Avatar,
  Button,
  Card,
  Col,
  Empty,
  List,
  Row,
  Spin,
  Tag,
  Typography,
} from 'antd';
import {
  BookOutlined,
  ClockCircleOutlined,
  FileSearchOutlined,
  FolderOutlined,
  GlobalOutlined,
  MenuOutlined,
  RightOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { PaginationQuery } from '@/core';
import { useI18n } from '@/core/i18n';
import { useUserStore } from '@/stores';
import { usePreferencesStore } from '@/core/preferences/store';
import { useListMyOnlineSessions } from '@/api/hooks/online-session';
import { fetchListUserInbox } from '@/api/hooks/internal-message';
import { formatDateTime } from '@/utils/date';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';

const { Text, Title } = Typography;

/**
 * 快捷入口：与 ele/vben 共用的叶子路由（三端路由对账过），label 复用
 * routes 命名空间的既有菜单文案，不新增同义 key。不做权限过滤——与
 * 「菜单可见但禁止访问」口径一致，无权限点击由路由闸渲染 403。
 */
const QUICK_LINKS = [
  { path: '/opm/users', labelKey: 'users', icon: <UserOutlined /> },
  { path: '/permission/roles', labelKey: 'roles', icon: <TeamOutlined /> },
  { path: '/permission/menus', labelKey: 'menus', icon: <MenuOutlined /> },
  { path: '/system/dict', labelKey: 'dict', icon: <BookOutlined /> },
  { path: '/system/tasks', labelKey: 'tasks', icon: <ClockCircleOutlined /> },
  { path: '/system/files', labelKey: 'files', icon: <FolderOutlined /> },
  { path: '/system/online-sessions', labelKey: 'online-sessions', icon: <GlobalOutlined /> },
  { path: '/log/login-audit-logs', labelKey: 'login-audit-log', icon: <FileSearchOutlined /> },
];

/** 按本地时段返回问候语 key（与 en/zh 各四条文案一一对应） */
const greetingKeyByHour = (hour: number) => {
  if (hour < 6) return 'workspace.greeting.night';
  if (hour < 12) return 'workspace.greeting.morning';
  if (hour < 18) return 'workspace.greeting.afternoon';
  return 'workspace.greeting.evening';
};

/**
 * 工作台：操作入口聚合页（概览组的第二页，与分析页的"数据洞察"分工）。
 * 数据全部来自既有接口，不新增后端资源。
 */
const Workspace = () => {
  const { t } = useI18n('dashboard');
  const { t: tRoutes } = useI18n('routes');
  const { t: tSession } = useI18n('online-session');
  const navigate = useNavigate();

  const userInfo = useUserStore((s) => s.userInfo);
  const locale = usePreferencesStore((s) => s.preferences.app.locale);

  const greetingKey = useMemo(() => greetingKeyByHour(new Date().getHours()), []);
  const displayName = userInfo?.nickname || userInfo?.username || '';
  const greetingText = displayName
    ? t(`${greetingKey}With`, { name: displayName })
    : t(greetingKey);

  // 日期只在挂载/语言切换时格式化一次，避免跨午夜渲染抖动
  const todayText = useMemo(() => {
    const date = new Intl.DateTimeFormat(locale, {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      weekday: 'long',
    }).format(new Date());
    return t('workspace.today', { date });
  }, [locale, t]);

  // 站内信未读数与最近消息：queryKey 与顶栏通知弹层共用，缓存天然共享
  const { data: inboxUnread } = useQuery({
    queryKey: ['inboxPreview', userInfo?.id],
    queryFn: async () => {
      const query = new PaginationQuery({
        paging: { page: 1, pageSize: 1 },
        // recipient_user_id 必传：不传会查到同租户其他用户的收件记录（顶栏同款口径）
        formValues: { recipient_user_id: String(userInfo?.id), status: 'RECEIVED' },
      });
      return await fetchListUserInbox(query);
    },
    enabled: Boolean(userInfo?.id),
  });

  const inboxRecentQuery = useQuery({
    queryKey: ['inboxPreviewList', userInfo?.id],
    queryFn: async () => {
      const query = new PaginationQuery({
        paging: { page: 1, pageSize: 5 },
        formValues: { recipient_user_id: String(userInfo?.id) },
      });
      return await fetchListUserInbox(query);
    },
    enabled: Boolean(userInfo?.id),
  });

  const sessionsQuery = useListMyOnlineSessions();

  const unreadCount = inboxUnread?.total ?? 0;
  const inboxItems = inboxRecentQuery.data?.items ?? [];

  return (
    <ContentContainer heightMode="auto" scrollable padding="16px">
      {/* 欢迎横幅 */}
      <Card styles={{ body: { display: 'flex', alignItems: 'center', gap: 16 } }}>
        <Avatar size={48} src={userInfo?.avatar} icon={<UserOutlined />} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <Title level={4} style={{ margin: 0 }}>
            {greetingText}
          </Title>
          <Text type="secondary">{todayText}</Text>
        </div>
      </Card>

      {/* 快捷入口 */}
      <Card title={t('workspace.quickAccess.title')} style={{ marginTop: 16 }}>
        <Row gutter={[8, 8]}>
          {QUICK_LINKS.map((link) => (
            <Col xs={12} sm={8} lg={6} key={link.path}>
              <div
                role="button"
                tabIndex={0}
                onClick={() => navigate(link.path)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') navigate(link.path);
                }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '14px 16px',
                  borderRadius: 8,
                  cursor: 'pointer',
                  background: 'var(--ant-color-fill-quaternary)',
                  transition: 'background 0.2s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--ant-color-fill-tertiary)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'var(--ant-color-fill-quaternary)';
                }}
              >
                <span
                  style={{
                    fontSize: 20,
                    color: 'var(--ant-color-primary)',
                    display: 'inline-flex',
                  }}
                >
                  {link.icon}
                </span>
                <Text style={{ fontSize: 14 }}>{tRoutes(link.labelKey)}</Text>
              </div>
            </Col>
          ))}
        </Row>
      </Card>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {/* 我的站内信 */}
        <Col xs={24} lg={12}>
          <Card
            title={t('workspace.myMessages.title')}
            extra={
              <Button
                type="link"
                size="small"
                icon={<RightOutlined />}
                iconPosition="end"
                onClick={() => navigate('/internal-message/inbox')}
              >
                {t('workspace.myMessages.viewAll')}
                {unreadCount > 0 ? ` (${unreadCount})` : ''}
              </Button>
            }
          >
            {inboxRecentQuery.isLoading ? (
              <div style={{ textAlign: 'center', padding: 24 }}>
                <Spin />
              </div>
            ) : inboxRecentQuery.isError ? (
              <Empty description={inboxRecentQuery.error.message} />
            ) : (
              <List
                size="small"
                dataSource={inboxItems}
                locale={{ emptyText: t('workspace.myMessages.empty') }}
                renderItem={(item) => {
                  const unread = item.status === 'RECEIVED';
                  return (
                    <List.Item
                      style={{ cursor: 'pointer', padding: '8px 4px', gap: 8 }}
                      onClick={() => navigate('/internal-message/inbox')}
                    >
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          flexShrink: 0,
                          background: unread
                            ? 'var(--ant-color-primary)'
                            : 'transparent',
                        }}
                      />
                      <span
                        style={{
                          flex: 1,
                          minWidth: 0,
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                          fontWeight: unread ? 600 : 400,
                          fontSize: 13,
                        }}
                      >
                        {item.title || '-'}
                      </span>
                      <Text type="secondary" style={{ fontSize: 12, flexShrink: 0 }}>
                        {formatDateTime(item.createdAt as any)}
                      </Text>
                    </List.Item>
                  );
                }}
              />
            )}
          </Card>
        </Col>

        {/* 我的活跃会话 */}
        <Col xs={24} lg={12}>
          <Card
            title={t('workspace.mySessions.title')}
            extra={
              <Button
                type="link"
                size="small"
                icon={<RightOutlined />}
                iconPosition="end"
                onClick={() => navigate('/system/online-sessions')}
              >
                {t('workspace.mySessions.manage')}
              </Button>
            }
          >
            {sessionsQuery.isLoading ? (
              <div style={{ textAlign: 'center', padding: 24 }}>
                <Spin />
              </div>
            ) : sessionsQuery.isError ? (
              <Empty description={sessionsQuery.error.message} />
            ) : (
              <List
                size="small"
                dataSource={sessionsQuery.data?.items ?? []}
                locale={{ emptyText: t('workspace.mySessions.empty') }}
                renderItem={(item) => (
                  <List.Item style={{ padding: '8px 4px', gap: 8 }}>
                    <div
                      style={{
                        flex: 1,
                        minWidth: 0,
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                      }}
                    >
                      <Tag color={item.clientType === 'app' ? 'purple' : 'blue'}>
                        {item.clientType === 'app'
                          ? tSession('clientApp')
                          : tSession('clientAdmin')}
                      </Tag>
                      {item.current && (
                        <Tag color="green">{tSession('currentSession')}</Tag>
                      )}
                      <Text style={{ fontSize: 13 }}>{item.ipAddress || '-'}</Text>
                    </div>
                    <Text type="secondary" style={{ fontSize: 12, flexShrink: 0 }}>
                      {formatDateTime(item.loginAt as any)}
                    </Text>
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>
      </Row>
    </ContentContainer>
  );
};

export default Workspace;
