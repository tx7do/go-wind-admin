import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { configservicev1_Config as Config } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListConfigs, useDeleteConfig } from '@/api/hooks/config';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getValueTypeLabel, valueTypeToColor } from './constants';
import ConfigDrawer from './components/ConfigDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 参数管理页面（系统参数，动态 KV 运行时配置）
 */
const ConfigManagement = () => {
  const { t } = useTranslation('config');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedConfig, setSelectedConfig] = useState<Config | undefined>();

  // 删除操作
  const deleteMutation = useDeleteConfig({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listConfigs'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<Config>[] = [
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
      title: t('name'),
      dataIndex: 'name',
      width: 180,
    },
    {
      title: t('key'),
      dataIndex: 'key',
      width: 220,
      copyable: true,
    },
    {
      title: t('value'),
      dataIndex: 'value',
      width: 180,
      ellipsis: true,
      hideInSearch: true,
    },
    {
      title: t('valueType'),
      dataIndex: 'valueType',
      width: 100,
      hideInSearch: true,
      render: (_, record) => (
        <Tag color={valueTypeToColor(record.valueType)}>
          {getValueTypeLabel(t, record.valueType)}
        </Tag>
      ),
    },
    {
      title: t('isBuiltIn'),
      dataIndex: 'isBuiltIn',
      width: 100,
      hideInSearch: true,
      render: (_, record) => (
        <Tag color={record.isBuiltIn ? 'gold' : 'default'}>
          {record.isBuiltIn ? t('builtInTag') : t('customTag')}
        </Tag>
      ),
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
                setSelectedConfig(record);
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
                description: t('deleteConfirmDesc', { name: record.name || record.key || '' }),
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
          <ListTable<Config>
            actionRef={actionRef}
            columns={columns}
            request={async (params, sorter, _filter) => {
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

              const response = await fetchListConfigs(query);

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
              <TableExportButton key="export" fetcher={fetchListConfigs} columns={columns} filename="configs" />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedConfig(undefined);
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
              x: 1100,
            }}
          />
        </div>
      </ContentContainer>

      {/* 参数编辑/创建 Drawer */}
      <ConfigDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedConfig}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedConfig(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default ConfigManagement;
