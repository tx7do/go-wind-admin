import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App, Empty, Popconfirm } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined, SyncOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { fetchListPermissions, useDeletePermission, useSyncPermissions } from '@/api/hooks/permission';
import { getStatusMap, getStatusOptions } from './constants';
import PermissionDrawer from './PermissionDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

interface PermissionListProps {
  groupId: number | null;
}

/**
 * 权限点列表
 */
const PermissionList: React.FC<PermissionListProps> = ({ groupId }) => {
  const { t } = useTranslation('permission');
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [editingPermission, setEditingPermission] = useState<any>(null);

  const statusMap = getStatusMap(t);

  // 删除 mutation
  const deleteMutation = useDeletePermission({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listPermissions'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 同步权限点：按菜单树重建业务权限分组与权限点
  const syncMutation = useSyncPermissions({
    onSuccess: () => {
      message.success(t('syncSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listPermissions'] });
      queryClient.invalidateQueries({ queryKey: ['listPermissionGroups'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('syncFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<any>[] = [
    {
      title: t('name'),
      dataIndex: 'name',
      width: 150,
      fixed: 'left',
    },
    {
      title: t('code'),
      dataIndex: 'code',
      width: 150,
      fixed: 'left',
    },
    {
      title: t('groupName'),
      dataIndex: 'groupName',
      width: 120,
      hideInSearch: true,
    },
    {
      title: t('status'),
      dataIndex: 'status',
      width: 95,
      valueType: 'select',
      fieldProps: {
        options: getStatusOptions(t),
      },
      render: (_, record) => {
        const status = record.status as keyof typeof statusMap;
        const config = statusMap[status] || { text: status, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 150,
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'edit',
              label: t('common:button.edit'),
              icon: <EditOutlined />,
              onClick: () => {
                setEditingPermission(record);
                setDrawerMode('edit');
                setDrawerOpen(true);
              },
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
              onClick: () => record.id && deleteMutation.mutate({ id: record.id }),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <>
      <div ref={containerRef} className="page-container-content" style={{ padding: '0 8px', height: '100%' }}>
        {groupId ? (
          <ListTable<any>
            actionRef={actionRef}
            columns={columns}
            headerTitle={false}
            params={{ groupId }}
            request={async (params) => {
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || TABLE.DEFAULT_PAGE_SIZE,
                },
                formValues: {
                  ...Object.fromEntries(
                    Object.entries(params).filter(
                      ([key]) => !['current', 'pageSize', 'groupId'].includes(key),
                    ),
                  ),
                  group_id: groupId,
                },
              });

              const response = await fetchListPermissions(query);

              return {
                data: response.items || [],
                total: Number(response.total ?? 0),
                success: true,
              };
            }}
            rowKey="id"
            search={{ optionRender: searchOptionRender,
              labelWidth: 'auto',
              defaultCollapsed: false,
            }}
            pagination={{
              defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
              showSizeChanger: true,
              showQuickJumper: true,
            }}
            toolBarRender={() => [
              <TableExportButton key="export" fetcher={fetchListPermissions} columns={columns} filename="permissions" />,
              <Popconfirm
                key="syncPermissions"
                title={t('syncConfirmTitle')}
                description={t('syncConfirmDesc')}
                onConfirm={() => syncMutation.mutateAsync()}
                okText={t('common:button.ok')}
                cancelText={t('common:button.cancel')}
              >
                <Button color="orange" variant="outlined" icon={<SyncOutlined />} loading={syncMutation.isPending}>
                  {t('syncPermissions')}
                </Button>
              </Popconfirm>,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditingPermission(null);
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
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
            <Empty description={t('selectGroupFirst')} />
          </div>
        )}
      </div>

      <PermissionDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={editingPermission}
        groupId={groupId!}
        onClose={() => {
          setDrawerOpen(false);
          setEditingPermission(null);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default PermissionList;
