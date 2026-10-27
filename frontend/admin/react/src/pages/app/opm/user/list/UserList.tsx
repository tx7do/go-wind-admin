import { useRef, useState, useEffect } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined, InfoCircleOutlined , SafetyOutlined, PrinterOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { usePrintTemplateDialog } from '@/components/common/PrintTemplatePrint';

import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { fetchListUsers, useDeleteUser } from '@/api/hooks/user';
import { useAdminResetMfa } from '@/api/hooks/mfa';
import { useAuthStore } from '@/stores';
import { useUserStore } from '@/stores/user';
import { parseResourceHiddenFields } from '@/core/access';
import { getUserStatusMap, getUserStatusOptions } from '../constants';
import UserDrawer from './UserDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

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

  const navigate = useNavigate();

  // 行打印定式（docs/print_template.md §5）：模板按 resourceKey 匹配，无绑定回退通用模板
  const { node: printDialogNode, print: printRow } = usePrintTemplateDialog();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [editingUser, setEditingUser] = useState<any>(null);

  const statusMap = getUserStatusMap(t);

  // 字段权限：当前用户在 User 资源上被隐藏的字段，命中的列整列不渲染
  const hiddenFieldEntries = useUserStore((s) => s.hiddenFields);
  const userHiddenFields = parseResourceHiddenFields(hiddenFieldEntries, 'User');

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
  // 管理端救援重置：清空目标用户 TOTP 因子（认证器丢失解锁用，仅平台管理员生效）
  // 仅平台侧操作者（tenantId=0）显示救援重置；非平台用户由后端 403 兜底
  const isPlatformSide = Number(useAuthStore((s) => s.userInfo?.tenantId ?? 0)) === 0;
  const adminResetMfa = useAdminResetMfa({
    onSuccess: () => {
      message.success(t('resetMfaSuccess'));
      actionRef.current?.reload();
    },
    onError: (err: Error) => message.error(err.message || t('resetMfaFailed')),
  });

  const columns: ProColumns<any>[] = ([
    {
      title: t('username'),
      dataIndex: 'username',
      width: 120,
      fixed: 'left',
    },
    {
      title: t('realname'),
      dataIndex: 'realname',
      width: 130,
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
            <Tag key={idx} style={{ marginBottom: 2 }}>
              {name}
            </Tag>
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
            <Tag key={idx} color="blue" style={{ marginBottom: 2 }}>
              {name}
            </Tag>
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
            <Tag key={idx} color="purple" style={{ marginBottom: 2 }}>
              {name}
            </Tag>
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
      width: 370,
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'detail',
              label: t('detail'),
              icon: <InfoCircleOutlined />,
              onClick: () => navigate(`/opm/users/detail/${record.id}`),
            },
            {
              key: 'print',
              label: t('printTemplate:print'),
              icon: <PrinterOutlined />,
              onClick: () => printRow('system/user', record),
            },
            {
              key: 'edit',
              label: t('common:button.edit'),
              icon: <EditOutlined />,
              onClick: () => {
                setEditingUser(record);
                setDrawerMode('edit');
                setDrawerOpen(true);
              },
            },
            {
              key: 'reset-mfa',
              label: t('resetMfa'),
              icon: <SafetyOutlined />,
              hidden: !isPlatformSide,
              confirm: { title: t('resetMfaConfirmTitle'), description: t('resetMfaConfirmDesc') },
              onClick: () =>
                record.id && adminResetMfa.mutate({ userId: record.id, method: 'TOTP' } as any),
            },
            {
              key: 'delete',
              label: t('common:button.delete'),
              icon: <DeleteOutlined />,
              danger: true,
              confirm: {
                title: t('deleteConfirmTitle'),
                description: t('deleteConfirmDesc', { moduleName: t('moduleName') }),
              },
              onClick: () => record.id && deleteMutation.mutate(record.id),
            },
          ]}
        />
      ),
    },
  ] satisfies ProColumns<any>[]).filter((col) => {
    const key = typeof col.dataIndex === 'string' ? col.dataIndex : undefined;
    return !key || !userHiddenFields.has(key);
  });

  return (
    <>
      <div
        ref={containerRef}
        className="page-container-content"
        style={{ padding: '0 8px', height: '100%' }}
      >
        <ListTable<any>
          actionRef={actionRef}
          columns={columns}
          headerTitle={false}
          params={{ tenantId, orgUnitId }}
          request={async (params) => {
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
          }}
          rowKey="id"
          search={{ optionRender: searchOptionRender,
            labelWidth: 'auto',
            defaultCollapsed: true,
          }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
          toolBarRender={() => [
            <TableExportButton key="export" fetcher={fetchListUsers} columns={columns} filename="users" />,
            <Button
              key="create"
              color="primary"
              variant="outlined"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditingUser(null);
                setDrawerMode('create');
                setDrawerOpen(true);
              }}
            >
              {t('common:button.add')}
            </Button>,
          ]}
          options={{
            density: true,
            fullScreen: true,
            setting: { settingIcon: <ColumnSlidersIcon /> },
            reload: true,
          }}
          size="small"
          bordered
          cardBordered={false}
          scroll={{ y: tableScrollY }}
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

      {printDialogNode}
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
