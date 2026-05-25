import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined, SyncOutlined } from '@ant-design/icons';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import type { resourceservicev1_Api as Api } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { listApis, deleteApi } from '@/api/service/api';
import { useTableScrollHeight } from '@/hooks/useTableScrollHeight';
import ApiDrawer from './components/ApiDrawer';

/**
 * HTTP 方法列表
 */
const methodList = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'DELETE', value: 'DELETE' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'HEAD', value: 'HEAD' },
  { label: 'OPTIONS', value: 'OPTIONS' },
];

/**
 * API管理页面
 */
const ApiManagement = () => {
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();

  const { message } = App.useApp();

  // 动态计算表格内容区域高度（搜索栏固定，表格数据区滚动）
  const tableScrollY = useTableScrollHeight();

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedApi, setSelectedApi] = useState<Api | undefined>();

  // 删除操作
  const deleteMutation = useMutation({
    mutationFn: (req: { id: number }) => deleteApi(req),
    onSuccess: () => {
      message.success('删除成功');
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listApis'] });
    },
    onError: (error: Error) => {
      message.error(error.message || '删除失败');
    },
  });

  // 同步操作
  const syncMutation = useMutation({
    mutationFn: async () => {
      // TODO: 调用同步 API 接口
      // return await syncApis();
      // 临时模拟同步操作
      await new Promise(resolve => setTimeout(resolve, 1000));
      return {};
    },
    onSuccess: () => {
      message.success('同步成功');
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listApis'] });
    },
    onError: (error: Error) => {
      message.error(error.message || '同步失败');
    },
  });

  // 处理同步
  const handleSync = async () => {
    await syncMutation.mutateAsync();
  };

  // 列配置
  const columns: ProColumns<Api>[] = [
    {
      title: '序号',
      dataIndex: 'id',
      width: 80,
      hideInSearch: true,
      render: (_, record, index) => {
        const pagination = record.id !== undefined ? actionRef.current?.pageInfo : undefined;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || 10;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: 'API描述',
      dataIndex: 'description',
      width: 200,
      formItemProps: {
        rules: [{ max: 200, message: '最多 200 个字符' }],
      },
    },
    {
      title: 'API路径',
      dataIndex: 'path',
      width: 250,
    },
    {
      title: '请求方法',
      dataIndex: 'method',
      width: 100,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        methodList.map((item) => [item.value, { text: item.label, status: 'Default' }]),
      ),
      render: (_, record) => {
        const colorMap: Record<string, string> = {
          GET: 'success',
          POST: 'processing',
          PUT: 'warning',
          DELETE: 'error',
          PATCH: 'default',
          HEAD: 'default',
          OPTIONS: 'default',
        };
        return <span style={{ color: colorMap[record.method || ''] || '#666' }}>{record.method}</span>;
      },
    },
    {
      title: '模块',
      dataIndex: 'module',
      width: 150,
    },
    {
      title: '模块描述',
      dataIndex: 'moduleDescription',
      width: 200,
      hideInSearch: true,
      ellipsis: true,
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
      title: '操作',
      valueType: 'option',
      width: 120,
      fixed: 'right',
      render: (_, record) => [
        <a
          key="edit"
          onClick={() => {
            setDrawerMode('edit');
            setSelectedApi(record);
            setDrawerOpen(true);
          }}
        >
          <EditOutlined /> 编辑
        </a>,
        <Popconfirm
          key="delete"
          title="确认删除"
          description={`确定要删除 API "${record.path}" 吗？`}
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
    <>
      <ProTable<Api>
      actionRef={actionRef}
      columns={columns}
      request={async (params, sorter, _filter) => {
        try {
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

          const response = await listApis(query);

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
      rowKey="id"
      search={{
        labelWidth: 'auto',
        defaultCollapsed: false,
      }}
      pagination={{
        defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
        showSizeChanger: true,
        showQuickJumper: true,
        // showTotal: (total) => `共 ${total} 条`,
        position: ['bottomRight'],
      }}
      toolBarRender={() => [
        <Button
          key="create"
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setDrawerMode('create');
            setSelectedApi(undefined);
            setDrawerOpen(true);
          }}
        >
          新建 API
        </Button>,
        <Popconfirm
          key="sync"
          title="确认同步"
          description="确定要同步 API 吗？这将从后端重新扫描并同步所有 API 接口。"
          onConfirm={handleSync}
          okText="确定"
          cancelText="取消"
        >
          <Button
            type="primary"
            danger
            icon={<SyncOutlined />}
          >
            同步 API
          </Button>
        </Popconfirm>,
      ]}
      options={{
        density: true,
        fullScreen: true,
        setting: true,
        reload: true,
      }}
      size="middle"
      bordered
      scroll={{
        y: tableScrollY, // 动态计算表格高度
        x: 1300, // 如果列太宽，启用横向滚动
      }}
    />

    {/* API 编辑/创建 Drawer */}
    <ApiDrawer
      open={drawerOpen}
      mode={drawerMode}
      data={selectedApi}
      onClose={() => {
        setDrawerOpen(false);
        setSelectedApi(undefined);
      }}
      onSuccess={() => {
        actionRef.current?.reload();
      }}
    />
    </>
  );
};

export default ApiManagement;
