import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { permissionservicev1_Role as Role } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchListRoles, useDeleteRole } from '@/api/hooks/role';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getStatusMap, getStatusOptions, getDataScopeMap } from './constants';
import RoleDrawer from './components/RoleDrawer';
import { TABLE } from '@/config/constants.ts';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 角色管理页面
 */
const RoleManagement = () => {
  const { t } = useTranslation('role');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedRole, setSelectedRole] = useState<Role | undefined>();

  const statusMap = getStatusMap(t);
  const dataScopeMap = getDataScopeMap(t);

  // 删除操作
  const deleteMutation = useDeleteRole({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listRoles'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<Role>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 60,
      hideInSearch: true,
      render: (_, _record, index) => {
        const pagination = actionRef.current?.pageInfo;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || TABLE.DEFAULT_PAGE_SIZE;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: t('name'),
      dataIndex: 'name',
      width: 150,
    },
    {
      title: t('code'),
      dataIndex: 'code',
      width: 150,
    },
    {
      title: t('parentName'),
      dataIndex: 'parentName',
      width: 130,
      hideInSearch: true,
    },
    {
      title: t('tenantName'),
      dataIndex: 'tenantName',
      width: 150,
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
      title: t('dataScope'),
      dataIndex: 'dataScope',
      width: 120,
      hideInSearch: true,
      render: (_, record) => {
        const ds = record.dataScope as keyof typeof dataScopeMap;
        const config = dataScopeMap[ds] || { text: ds, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('description'),
      dataIndex: 'description',
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
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
                setDrawerMode('edit');
                setSelectedRole(record);
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
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ListTable<Role>
            actionRef={actionRef}
            columns={columns}
            request={async (params, _sorter, _filter) => {
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || TABLE.DEFAULT_PAGE_SIZE,
                },
                formValues: Object.fromEntries(
                  Object.entries(params).filter(
                    ([key]) => !['current', 'pageSize'].includes(key),
                  ),
                ),
              });

              const response = await fetchListRoles(query);

              return {
                data: response.items || [],
                total: response.total || 0,
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
              <TableExportButton key="export" fetcher={fetchListRoles} columns={columns} filename="roles" />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedRole(undefined);
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
            size="middle"
            bordered
            cardBordered={false}
            scroll={{ y: tableScrollY }}
          />
        </div>
      </ContentContainer>

      {/* 角色编辑/创建 Drawer */}
      <RoleDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedRole}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedRole(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default RoleManagement;
