import { useRef } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import ListTable from '@/components/common/ListTable';
import TableActions from '@/components/common/TableActions';
import { App, Tag, Tooltip } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type {
  online_sessionservicev1_OnlineSession as OnlineSession,
} from '@/api/generated/admin/service/v1';
import {
  fetchListOnlineSessions,
  useForceLogoutSession,
} from '@/api/hooks/online-session';
import TableExportButton from '@/components/common/TableExportButton';
import { TABLE } from '@/config/constants';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 在线用户（在线会话）页面
 * 数据来自 Redis 会话元数据：登录 / 刷新轮换 / MFA 验证通过时写入，
 * 随 refresh token 过期或吊销消失。支持按用户名/IP 搜索与强制下线。
 */
const OnlineSessionPage = () => {
  const { t } = useTranslation('online-session');
  const actionRef = useRef<ActionType>(null);
  const { message } = App.useApp();
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const forceLogoutMutation = useForceLogoutSession({
    onSuccess: () => {
      message.success(t('forceLogoutSuccess'));
      actionRef.current?.reload();
    },
  });

  const handleForceLogout = async (record: OnlineSession) => {
    if (!record.jti || record.userId === undefined) {
      return;
    }
    try {
      await forceLogoutMutation.mutateAsync({
        clientType: record.clientType,
        userId: record.userId,
        jti: record.jti,
      });
    } catch (error: any) {
      message.error(error.message || t('forceLogoutFailed'));
    }
  };

  const columns: ProColumns<OnlineSession>[] = [
    {
      title: t('username'),
      dataIndex: 'username',
      width: 130,
      ellipsis: true,
      hideInSearch: true, // 后端仅支持 keyword 聚合过滤（用户名/IP），单独的 username 搜索不生效
    },
    {
      title: t('tenant'),
      dataIndex: 'tenantId',
      width: 90,
      hideInSearch: true,
      render: (_, record) =>
        (record.tenantId ?? 0) === 0 ? (
          <Tag>{t('platform')}</Tag>
        ) : (
          `#${record.tenantId}`
        ),
    },
    {
      title: t('clientType'),
      dataIndex: 'clientType',
      width: 100,
      hideInSearch: true,
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
      ellipsis: true,
      hideInSearch: true, // 同上，IP 过滤并入 keyword
    },
    {
      title: t('userAgent'),
      dataIndex: 'userAgent',
      hideInSearch: true,
      ellipsis: { showTitle: false },
      render: (_, record) => (
        <Tooltip title={record.userAgent || '-'} placement="topLeft">
          <span style={{ color: 'inherit' }}>{record.userAgent || '-'}</span>
        </Tooltip>
      ),
    },
    {
      title: t('deviceId'),
      dataIndex: 'deviceId',
      width: 130,
      hideInSearch: true,
      ellipsis: true,
      render: (_, record) => record.deviceId || '-',
    },
    {
      title: t('loginAt'),
      dataIndex: 'loginAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      // 搜索框只有一个"用户名/IP"关键词，对应后端 keyword 过滤
      title: t('keyword'),
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: t('keywordPlaceholder'),
      },
    },
    {
      title: t('actions'),
      valueType: 'option',
      // 实测需 109px，110 仅剩 1px 余量，抬到 120 留字体渲染误差空间
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'force-logout',
              label: t('forceLogout'),
              icon: <DeleteOutlined />,
              danger: true,
              confirm: {
                title: t('forceLogoutConfirm'),
                description: t('forceLogoutConfirmDesc', {
                  user: record.username || record.userId,
                }),
                okLoading: forceLogoutMutation.isPending,
              },
              onClick: () => handleForceLogout(record),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ListTable<OnlineSession>
          actionRef={actionRef}
          columns={columns}
          request={async (params) => {
            const response = await fetchListOnlineSessions({
              page: params.current || 1,
              pageSize: params.pageSize || 20,
              keyword: (params.keyword as string) || undefined,
            });
            return {
              data: response.items || [],
              total: Number(response.total ?? 0),
              success: true,
            };
          }}
          toolBarRender={() => [
            <TableExportButton
              key="export"
              fetcher={(q) =>
                fetchListOnlineSessions({
                  page: Number(q.paging?.page) || 1,
                  pageSize: Number(q.paging?.pageSize) || 1000,
                  keyword: (q.formValues?.keyword as string) || undefined,
                })
              }
              columns={columns}
              filename="online-sessions"
            />,
          ]}
          rowKey={(record) => `${record.clientType}-${record.userId}-${record.jti}`}
          search={{ optionRender: searchOptionRender,
            labelWidth: 'auto',
            defaultCollapsed: false,
          }}
          polling={30}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
          options={{
            density: true,
            fullScreen: true,
            setting: { settingIcon: <ColumnSlidersIcon /> },
            reload: true,
          }}
          size="middle"
          bordered
          cardBordered={false}
          scroll={{
            y: tableScrollY,
            x: 1200,
          }}
        />
      </div>
    </ContentContainer>
  );
};

export default OnlineSessionPage;
