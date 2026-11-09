import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Tag, Button, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { identityservicev1_Tenant } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListTenants, useDeleteTenant } from '@/api/hooks/tenant';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import {
  getTenantTypeMap,
  getAuditStatusMap,
  getTenantStatusMap,
  TENANT_TYPE_STATUS,
  AUDIT_STATUS_STATUS,
  TENANT_STATUS_STATUS,
} from './constants';
import TenantDrawer from './components/TenantDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 租户列表页面
 */
const TenantList = () => {
  const { t } = useTranslation('tenant');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();

  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedTenant, setSelectedTenant] = useState<identityservicev1_Tenant | undefined>();

  const tenantTypeMap = getTenantTypeMap(t);
  const auditStatusMap = getAuditStatusMap(t);
  const tenantStatusMap = getTenantStatusMap(t);

  // 删除操作
  const deleteMutation = useDeleteTenant({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listTenants'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置（ProColumns 提供更丰富的 valueType）
  const columns: ProColumns<identityservicev1_Tenant>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 60,
      hideInSearch: true,
      render: (_, record, index) => {
        // ProTable 自动传递 pagination 信息
        const pagination = record.id !== undefined ? actionRef.current?.pageInfo : undefined;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || 10;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: t('name'),
      dataIndex: 'name',
      formItemProps: {
        rules: [{ max: 50, message: t('maxChars', { max: 50 }) }],
      },
    },
    {
      title: t('code'),
      dataIndex: 'code',
    },
    {
      title: t('adminUsername'),
      dataIndex: 'adminUserName',
      hideInSearch: true,
    },
    {
      title: t('tenantType'),
      dataIndex: 'type',
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(TENANT_TYPE_STATUS).map(([key, status]) => [
          key,
          {
            text: tenantTypeMap[key as keyof typeof tenantTypeMap]?.text || t('typeUnknown'),
            status,
          },
        ]),
      ),
      render: (_, record) => {
        const type = record.type as keyof typeof tenantTypeMap;
        const config = tenantTypeMap[type] || { text: t('typeUnknown'), color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('auditStatus'),
      dataIndex: 'auditStatus',
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(AUDIT_STATUS_STATUS).map(([key, status]) => [
          key,
          {
            text: auditStatusMap[key as keyof typeof auditStatusMap]?.text || t('auditUnknown'),
            status,
          },
        ]),
      ),
      render: (_, record) => {
        const status = record.auditStatus as keyof typeof auditStatusMap;
        // 未送审（auditStatus 为空/未指定）的租户显示 "-" 而非 "未知"
        const config = auditStatusMap[status] || {
          text: status ? t('auditUnknown') : '-',
          color: 'default',
        };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('status'),
      dataIndex: 'status',
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(TENANT_STATUS_STATUS).map(([key, status]) => [
          key,
          {
            text: tenantStatusMap[key as keyof typeof tenantStatusMap]?.text || t('statusUnknown'),
            status,
          },
        ]),
      ),
      render: (_, record) => {
        const status = record.status as keyof typeof tenantStatusMap;
        const config = tenantStatusMap[status] || { text: t('statusUnknown'), color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
    },
    {
      title: t('memberCount'),
      dataIndex: 'memberCount',
      hideInSearch: true,
    },
    {
      title: t('remark'),
      dataIndex: 'remark',
      hideInSearch: true,
      ellipsis: true,
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
                setSelectedTenant(record);
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
                description: t('deleteConfirmDesc', { name: record.name }),
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
          <ListTable<identityservicev1_Tenant>
            actionRef={actionRef}
            columns={columns}
            request={async (params, sorter, _filter) => {
              // 构建查询对象
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || 10,
                },
                formValues: Object.fromEntries(
                  Object.entries(params).filter(
                    ([key]) => !['current', 'pageSize'].includes(key),
                  ),
                ),
                orderBy:
                  sorter && Object.keys(sorter).length > 0
                    ? Object.entries(sorter).map(([key, value]) =>
                        value === 'ascend' ? key : `-${key}`,
                      )
                    : undefined,
              });

              // 调用 API
              const response = await fetchListTenants(query);

              // ProTable 要求返回格式：{ data, total, success }
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
              span: 4,
            }}
            pagination={{
              defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
              showSizeChanger: true,
              showQuickJumper: true,
            }}
            toolBarRender={() => [
              <TableExportButton key="export" fetcher={fetchListTenants} columns={columns} filename="tenants" />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedTenant(undefined);
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
            scroll={{
              y: tableScrollY,
            }}
          />
        </div>
      </ContentContainer>

      {/* 租户编辑/创建 Drawer */}
      <TenantDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedTenant}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedTenant(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default TenantList;
