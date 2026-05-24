import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Tag, Button, Popconfirm, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import type {
  identityservicev1_Tenant,
  identityservicev1_Tenant_Type,
  identityservicev1_Tenant_Status,
  identityservicev1_Tenant_AuditStatus,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { listTenants, deleteTenant } from '@/api/service/tenant';
import TenantDrawer from './components/TenantDrawer';

/**
 * 租户列表页面
 */
const TenantList = () => {
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();

  const { message } = App.useApp();

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedTenant, setSelectedTenant] = useState<identityservicev1_Tenant | undefined>();

  // 删除操作（单独用 useMutation）
  const deleteMutation = useMutation({
    mutationFn: (req: { id: number }) => deleteTenant(req),
    onSuccess: () => {
      message.success('删除成功');
      // ProTable 会自动刷新
      actionRef.current?.reload();
      // 清除 React Query 缓存
      queryClient.invalidateQueries({ queryKey: ['listTenants'] });
    },
    onError: (error: Error) => {
      message.error(error.message || '删除失败');
    },
  });

  // 列配置（ProColumns 提供更丰富的 valueType）
  const columns: ProColumns<identityservicev1_Tenant>[] = [
    {
      title: '序号',
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
      title: '租户名称',
      dataIndex: 'name',
      width: 150,
      formItemProps: {
        rules: [{ max: 50, message: '最多 50 个字符' }],
      },
    },
    {
      title: '租户编码',
      dataIndex: 'code',
      width: 150,
    },
    {
      title: '管理员用户名',
      dataIndex: 'adminUsername',
      width: 150,
      hideInSearch: true,
    },
    {
      title: '租户类型',
      dataIndex: 'tenantType',
      width: 100,
      valueType: 'select',
      valueEnum: {
        TRIAL: { text: '试用', status: 'Default' },
        PAID: { text: '付费', status: 'Success' },
        INTERNAL: { text: '内部', status: 'Processing' },
        PARTNER: { text: '合作伙伴', status: 'Warning' },
        CUSTOM: { text: '自定义', status: 'Default' },
      },
      render: (_, record) => {
        const typeMap: Record<string, { text: string; color: string }> = {
          TRIAL: { text: '试用', color: 'default' },
          PAID: { text: '付费', color: 'success' },
          INTERNAL: { text: '内部', color: 'processing' },
          PARTNER: { text: '合作伙伴', color: 'warning' },
          CUSTOM: { text: '自定义', color: 'default' },
        };
        const type = record.type as identityservicev1_Tenant_Type;
        const config = typeMap[type] || { text: '未知', color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '审核状态',
      dataIndex: 'auditStatus',
      width: 100,
      valueType: 'select',
      valueEnum: {
        PENDING: { text: '待审核', status: 'Warning' },
        APPROVED: { text: '已通过', status: 'Success' },
        REJECTED: { text: '已拒绝', status: 'Error' },
      },
      render: (_, record) => {
        const statusMap: Record<string, { text: string; color: string }> = {
          PENDING: { text: '待审核', color: 'warning' },
          APPROVED: { text: '已通过', color: 'success' },
          REJECTED: { text: '已拒绝', color: 'error' },
        };
        const status = record.auditStatus as identityservicev1_Tenant_AuditStatus;
        const config = statusMap[status] || { text: '未知', color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      valueEnum: {
        ON: { text: '启用', status: 'Success' },
        OFF: { text: '禁用', status: 'Error' },
        EXPIRED: { text: '已过期', status: 'Warning' },
        FREEZE: { text: '已冻结', status: 'Default' },
      },
      render: (_, record) => {
        const statusMap: Record<string, { text: string; color: string }> = {
          ON: { text: '启用', color: 'success' },
          OFF: { text: '禁用', color: 'error' },
          EXPIRED: { text: '已过期', color: 'warning' },
          FREEZE: { text: '已冻结', color: 'default' },
        };
        const status = record.status as identityservicev1_Tenant_Status;
        const config = statusMap[status] || { text: '未知', color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
    },
    {
      title: '备注',
      dataIndex: 'remark',
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: '操作',
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
          <EditOutlined /> 编辑
        </a>,
        <Popconfirm
          key="delete"
          title="确认删除"
          description={`确定要删除租户 "${record.name}" 吗？`}
          onConfirm={() => record.id && deleteMutation.mutate({ id: record.id })}
          okText="确定"
          cancelText="取消"
        >
          <a style={{ color: '#ff4d4f' }}>
            <DeleteOutlined /> 删除
          </a>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <ProTable<identityservicev1_Tenant>
        actionRef={actionRef}
        columns={columns}
        // 核心：request 属性自动处理请求
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
            const response = await listTenants(query);

            // ProTable 要求返回格式：{ data, total, success }
            return {
              data: response.items || [],
              total: response.total || 0,
              success: true,
            };
          } catch (error: any) {
            message.error(error.message || '获取数据失败');
            return {
              data: [],
              total: 0,
              success: false,
            };
          }
        }}
        // 行键
        rowKey="id"
        // 搜索表单配置
        search={{
          labelWidth: 'auto',
          defaultCollapsed: false,
        }}
        // 分页配置
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
        // 工具栏配置
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
            新建租户
          </Button>,
        ]}
        // 其他配置
        options={{
          density: true,
          fullScreen: true,
          setting: true,
          reload: true,
        }}
        size="middle"
        bordered
        scroll={{ x: 1200 }}
      />

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
    </div>
  );
};

export default TenantList;
