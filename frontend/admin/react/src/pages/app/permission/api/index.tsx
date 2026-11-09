import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Popconfirm, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined, SyncOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { permissionservicev1_Api as Api } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE, METHOD_LIST } from '@/config/constants';
import { fetchListApis, useSyncApisApi, useDeleteApi } from '@/api/hooks/api';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getMethodColor } from './constants';
import ApiDrawer from './components/ApiDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

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
      width: 60,
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
      formItemProps: {
        rules: [{ max: 200, message: t('maxChars', { max: 200 }) }],
      },
    },
    {
      title: t('path'),
      dataIndex: 'path',
    },
    {
      title: t('method'),
      dataIndex: 'method',
      valueType: 'select',
      valueEnum: Object.fromEntries(
        METHOD_LIST.map((item) => [item.value, { text: item.label, status: 'Default' }]),
      ),
      render: (_, record) => {
        // Tag 预设色名不能当 CSS color 用：'success' 这类不是 <color>，整条声明被浏览器丢弃
        return <Tag color={getMethodColor(record.method)}>{record.method}</Tag>;
      },
    },
    {
      title: t('module'),
      dataIndex: 'module',
    },
    {
      title: t('moduleDescription'),
      dataIndex: 'moduleDescription',
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
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
                setSelectedApi(record);
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
                description: t('deleteConfirmDesc', { path: record.path }),
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
          <ListTable<Api>
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
              <TableExportButton key="export" fetcher={fetchListApis} columns={columns} filename="apis" />,
              <Popconfirm
                key="sync"
                title={t('syncConfirmTitle')}
                description={t('syncConfirmDesc')}
                onConfirm={handleSync}
                okText={t('common:button.ok')}
                cancelText={t('common:button.cancel')}
              >
                <Button color="orange" variant="outlined" icon={<SyncOutlined />}>
                  {t('sync')}
                </Button>
              </Popconfirm>,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedApi(undefined);
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
