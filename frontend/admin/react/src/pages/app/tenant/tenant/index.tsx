import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Tag, Button, Popconfirm, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type {
  identityservicev1_Tenant,
  identityservicev1_Tenant_Type,
  identityservicev1_Tenant_Status,
  identityservicev1_Tenant_AuditStatus,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListTenants, useDeleteTenant } from '@/api/hooks/tenant';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import TenantDrawer from './components/TenantDrawer';

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
      width: 80,
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
      width: 150,
      formItemProps: {
        rules: [{ max: 50, message: t('maxChars', { max: 50 }) }],
      },
    },
    {
      title: t('code'),
      dataIndex: 'code',
      width: 150,
    },
    {
      title: t('adminUsername'),
      dataIndex: 'adminUsername',
      width: 150,
      hideInSearch: true,
    },
    {
      title: t('tenantType'),
      dataIndex: 'tenantType',
      width: 100,
      valueType: 'select',
      valueEnum: {
        TRIAL: { text: t('type.TRIAL'), status: 'Default' },
        PAID: { text: t('type.PAID'), status: 'Success' },
        INTERNAL: { text: t('type.INTERNAL'), status: 'Processing' },
        PARTNER: { text: t('type.PARTNER'), status: 'Warning' },
        CUSTOM: { text: t('type.CUSTOM'), status: 'Default' },
      },
      render: (_, record) => {
        const typeMap: Record<string, { text: string; color: string }> = {
          TRIAL: { text: t('type.TRIAL'), color: 'default' },
          PAID: { text: t('type.PAID'), color: 'success' },
          INTERNAL: { text: t('type.INTERNAL'), color: 'processing' },
          PARTNER: { text: t('type.PARTNER'), color: 'warning' },
          CUSTOM: { text: t('type.CUSTOM'), color: 'default' },
        };
        const type = record.type as identityservicev1_Tenant_Type;
        const config = typeMap[type] || { text: t('typeUnknown'), color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('auditStatus'),
      dataIndex: 'auditStatus',
      width: 100,
      valueType: 'select',
      valueEnum: {
        PENDING: { text: t('audit.PENDING'), status: 'Warning' },
        APPROVED: { text: t('audit.APPROVED'), status: 'Success' },
        REJECTED: { text: t('audit.REJECTED'), status: 'Error' },
      },
      render: (_, record) => {
        const statusMap: Record<string, { text: string; color: string }> = {
          PENDING: { text: t('audit.PENDING'), color: 'warning' },
          APPROVED: { text: t('audit.APPROVED'), color: 'success' },
          REJECTED: { text: t('audit.REJECTED'), color: 'error' },
        };
        const status = record.auditStatus as identityservicev1_Tenant_AuditStatus;
        const config = statusMap[status] || { text: t('auditUnknown'), color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('status'),
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      valueEnum: {
        ON: { text: t('tenantStatus.ON'), status: 'Success' },
        OFF: { text: t('tenantStatus.OFF'), status: 'Error' },
        EXPIRED: { text: t('tenantStatus.EXPIRED'), status: 'Warning' },
        FREEZE: { text: t('tenantStatus.FREEZE'), status: 'Default' },
      },
      render: (_, record) => {
        const statusMap: Record<string, { text: string; color: string }> = {
          ON: { text: t('tenantStatus.ON'), color: 'success' },
          OFF: { text: t('tenantStatus.OFF'), color: 'error' },
          EXPIRED: { text: t('tenantStatus.EXPIRED'), color: 'warning' },
          FREEZE: { text: t('tenantStatus.FREEZE'), color: 'default' },
        };
        const status = record.status as identityservicev1_Tenant_Status;
        const config = statusMap[status] || { text: t('statusUnknown'), color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
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
      width: 120,
      fixed: 'right',
      render: (_, record) => [
        <a
          key="edit"
          onClick={() => {
            setDrawerMode('edit');
            setSelectedTenant(record);
            setDrawerOpen(true);
          }}
        >
          <EditOutlined /> {t('common:button.edit')}
        </a>,
        <Popconfirm
          key="delete"
          title={t('deleteConfirmTitle')}
          description={t('deleteConfirmDesc', { name: record.name })}
          onConfirm={() => record.id && deleteMutation.mutate({ id: record.id })}
          okText={t('common:button.ok')}
          cancelText={t('common:button.cancel')}
        >
          <a style={{ color: '#ff4d4f' }}>
            <DeleteOutlined /> {t('common:button.delete')}
          </a>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <>
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ProTable<identityservicev1_Tenant>
            actionRef={actionRef}
            columns={columns}
            request={async (params, sorter, _filter) => {
              try {
                // 构建查询对象
                const query = new PaginationQuery({
                  paging: {
                    page: params.current || 1,
                    pageSize: params.pageSize || 10,
                  },
                  formValues: Object.fromEntries(
                    Object.entries(params).filter(([key]) => !['current', 'pageSize'].includes(key)),
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
                  total: response.total || 0,
                  success: true,
                };
              } catch (error: any) {
                message.error(error.message || t('fetchFailed'));
                return {
                  data: [],
                  total: 0,
                  success: false,
                };
              }
            }}
            rowKey="id"
            search={{
              labelWidth: 'auto',
              defaultCollapsed: false,
            }}
            pagination={{
              defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
              showSizeChanger: true,
              showQuickJumper: true,
              position: ['bottomRight'],
            }}
            toolBarRender={() => [
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedTenant(undefined);
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
            size="middle"
            bordered
            cardBordered={false}
            scroll={{
              y: tableScrollY,
              x: 1300,
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
