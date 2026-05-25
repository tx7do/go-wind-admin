import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined, SyncOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { resourceservicev1_Api as Api } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE, METHOD_LIST } from '@/config/constants';
import { fetchListApis, useSyncApisApi, useDeleteApi } from '@/api/hooks/api';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getMethodColor } from './constants';
import ApiDrawer from './components/ApiDrawer';

/**
 * API管理页面
 */
const ApiManagement = () => {
  const { t } = useTranslation('api');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();

  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedApi, setSelectedApi] = useState<Api | undefined>();

  // 删除操作
  const deleteMutation = useDeleteApi({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listApis'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 同步操作
  const syncMutation = useSyncApisApi({
    onSuccess: () => {
      message.success(t('syncSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listApis'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('syncFailed'));
    },
  });

  // 处理同步
  const handleSync = async () => {
    await syncMutation.mutateAsync();
  };

  // 列配置
  const columns: ProColumns<Api>[] = [
    {
      title: t('serial'),
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
      title: t('description'),
      dataIndex: 'description',
      width: 200,
      formItemProps: {
        rules: [{ max: 200, message: t('maxChars', { max: 200 }) }],
      },
    },
    {
      title: t('path'),
      dataIndex: 'path',
      width: 250,
    },
    {
      title: t('method'),
      dataIndex: 'method',
      width: 100,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        METHOD_LIST.map((item) => [item.value, { text: item.label, status: 'Default' }]),
      ),
      render: (_, record) => {
        return <span style={{ color: getMethodColor(record.method) }}>{record.method}</span>;
      },
    },
    {
      title: t('module'),
      dataIndex: 'module',
      width: 150,
    },
    {
      title: t('moduleDescription'),
      dataIndex: 'moduleDescription',
      width: 200,
      hideInSearch: true,
      ellipsis: true,
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
      title: t('action'),
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
          <EditOutlined /> {t('common:button.edit')}
        </a>,
        <Popconfirm
          key="delete"
          title={t('deleteConfirmTitle')}
          description={t('deleteConfirmDesc', { path: record.path })}
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
          <ProTable<Api>
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
                const response = await fetchListApis(query);

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
                  setSelectedApi(undefined);
                  setDrawerOpen(true);
                }}
              >
                {t('create')}
              </Button>,
              <Popconfirm
                key="sync"
                title={t('syncConfirmTitle')}
                description={t('syncConfirmDesc')}
                onConfirm={handleSync}
                okText={t('common:button.ok')}
                cancelText={t('common:button.cancel')}
              >
                <Button type="primary" danger icon={<SyncOutlined />}>
                  {t('sync')}
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
            cardBordered={false}
            scroll={{
              y: tableScrollY,
              x: 1300,
            }}
          />
        </div>
      </ContentContainer>

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
