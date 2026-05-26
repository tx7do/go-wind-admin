import { useRef, useState, useEffect } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, App } from 'antd';
import {
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { fetchListUsers, useDeleteUser } from '@/api/hooks/user';
import { getUserStatusMap, getUserStatusOptions } from './constants';
import UserDrawer from './UserDrawer';

interface UserListProps {
  tenantId: number | undefined;
  orgUnitId: number | undefined;
}

/**
 * 用户列表（右侧）
 */
const UserList: React.FC<UserListProps> = ({ tenantId, orgUnitId }) => {
  const { t } = useTranslation('user');
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [editingUser, setEditingUser] = useState<any>(null);

  const statusMap = getUserStatusMap(t);

  // 当 tenantId 或 orgUnitId 变化时自动刷新列表
  useEffect(() => {
    actionRef.current?.reload();
  }, [tenantId, orgUnitId]);

  // 删除 mutation
  const deleteMutation = useDeleteUser({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listUsers'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<any>[] = [
    {
      title: t('username'),
      dataIndex: 'username',
      width: 120,
      fixed: 'left',
    },
    {
      title: t('realname'),
      dataIndex: 'realname',
      width: 100,
    },
    {
      title: t('nickname'),
      dataIndex: 'nickname',
      width: 100,
    },
    {
      title: t('email'),
      dataIndex: 'email',
      width: 160,
    },
    {
      title: t('mobile'),
      dataIndex: 'mobile',
      width: 130,
    },
    {
      title: t('orgUnitIds'),
      dataIndex: 'orgUnitNames',
      width: 130,
      hideInSearch: true,
      render: (_, record) => (
        <span>
          {(record.orgUnitNames || []).map((name: string, idx: number) => (
            <Tag key={idx} style={{ marginBottom: 2 }}>{name}</Tag>
          ))}
        </span>
      ),
    },
    {
      title: t('positionIds'),
      dataIndex: 'positionNames',
      width: 130,
      hideInSearch: true,
      render: (_, record) => (
        <span>
          {(record.positionNames || []).map((name: string, idx: number) => (
            <Tag key={idx} color="blue" style={{ marginBottom: 2 }}>{name}</Tag>
          ))}
        </span>
      ),
    },
    {
      title: t('roleIds'),
      dataIndex: 'roleNames',
      width: 120,
      hideInSearch: true,
      render: (_, record) => (
        <span>
          {(record.roleNames || []).map((name: string, idx: number) => (
            <Tag key={idx} color="purple" style={{ marginBottom: 2 }}>{name}</Tag>
          ))}
        </span>
      ),
    },
    {
      title: t('status'),
      dataIndex: 'status',
      width: 95,
      valueType: 'select',
      fieldProps: {
        options: getUserStatusOptions(t),
      },
      render: (_, record) => {
        const status = record.status as string;
        const config = statusMap[status] || { text: status, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('lastLoginAt'),
      dataIndex: 'lastLoginAt',
      width: 160,
      hideInSearch: true,
      render: (_, record) => {
        if (!record.lastLoginAt) return '-';
        return formatTimestamp(record.lastLoginAt);
      },
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 160,
      hideInSearch: true,
      render: (_, record) => {
        if (!record.createdAt) return '-';
        return formatTimestamp(record.createdAt);
      },
    },
    {
      title: t('remark'),
      dataIndex: 'remark',
      width: 200,
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 100,
      fixed: 'right',
      render: (_, record) => [
        <a
          key="detail"
          title={t('detail')}
        >
          <InfoCircleOutlined />
        </a>,
        <a
          key="edit"
          onClick={() => {
            setEditingUser(record);
            setDrawerMode('edit');
            setDrawerOpen(true);
          }}
        >
          <EditOutlined />
        </a>,
        <Popconfirm
          key="delete"
          title={t('deleteConfirmTitle')}
          description={t('deleteConfirmDesc', { moduleName: t('moduleName') })}
          onConfirm={() => record.id && deleteMutation.mutate(record.id)}
          okText={t('common:button.ok')}
          cancelText={t('common:button.cancel')}
        >
          <a style={{ color: '#ff4d4f' }}><DeleteOutlined /></a>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <>
      <div ref={containerRef} className="page-container-content" style={{ padding: '0 8px', height: '100%' }}>
        <ProTable<any>
          actionRef={actionRef}
          columns={columns}
          headerTitle={false}
          params={{ tenantId, orgUnitId }}
          request={async (params) => {
            try {
              const formValues: Record<string, any> = {
                ...Object.fromEntries(
                  Object.entries(params).filter(
                    ([key]) => !['current', 'pageSize', 'tenantId', 'orgUnitId'].includes(key),
                  ),
                ),
              };
              // tenantId 存在时带上租户筛选
              if (tenantId != null) {
                formValues.tenant_id = tenantId;
              }
              // orgUnitId 存在时带上组织筛选
              if (orgUnitId != null) {
                formValues.org_unit_id = orgUnitId;
              }
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || TABLE.DEFAULT_PAGE_SIZE,
                },
                formValues,
              });

              const response = await fetchListUsers(query);

              return {
                data: response.items || [],
                total: response.total || 0,
                success: true,
              };
            } catch (error: any) {
              message.error(error.message || t('fetchFailed'));
              return { data: [], total: 0, success: false };
            }
          }}
          rowKey="id"
          search={{
            labelWidth: 'auto',
            defaultCollapsed: true,
          }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
          toolBarRender={() => [
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              size="small"
              onClick={() => {
                setEditingUser(null);
                setDrawerMode('create');
                setDrawerOpen(true);
              }}
            >
              {t('create')}
            </Button>,
          ]}
          options={{
            density: true,
            fullScreen: true,
            setting: true,
            reload: true,
          }}
          size="small"
          bordered
          cardBordered={false}
          scroll={{ y: tableScrollY, x: 1600 }}
        />
      </div>

      <UserDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={editingUser}
        tenantId={tenantId}
        onClose={() => {
          setDrawerOpen(false);
          setEditingUser(null);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default UserList;

/**
 * 格式化 wellKnownTimestamp（秒级或毫秒级时间戳）
 */
function formatTimestamp(ts: any): string {
  if (!ts) return '-';
  let ms: number;
  if (typeof ts === 'object' && ts.seconds !== undefined) {
    // protobuf Timestamp: { seconds: string|number, nanos?: number }
    ms = Number(ts.seconds) * 1000 + Math.floor((ts.nanos || 0) / 1e6);
  } else if (typeof ts === 'number') {
    ms = ts > 1e12 ? ts : ts * 1000;
  } else if (typeof ts === 'string') {
    ms = new Date(ts).getTime();
  } else {
    return '-';
  }
  const d = new Date(ms);
  if (isNaN(d.getTime())) return '-';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
