import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Tag, Button, App } from 'antd';
import {
  DeleteOutlined,
  CopyOutlined,
  PlusOutlined,
  BgColorsOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { biservicev1_BiDashboard as BiDashboard } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import {
  fetchListBiDashboards,
  useCopyBiDashboard,
  useDeleteBiDashboard,
} from '@/api/hooks/bi-dashboard';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import BiDesignerDrawer from './components/BiDesignerDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

const KIND_TAGS: Record<string, { color: string; key: string }> = {
  DASHBOARD: { color: 'processing', key: 'kindDashboard' },
  SCREEN: { color: 'purple', key: 'kindScreen' },
  TEMPLATE: { color: 'gold', key: 'kindTemplate' },
};

/**
 * BI 仪表盘管理页：设计器（@go-wind/bi-*）schema 载体的列表；
 * 「设计」进全屏抽屉画布；内置行禁删可复制派生。
 */
const BiDashboardManagement = () => {
  const { t } = useTranslation('biDashboard');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<BiDashboard | undefined>();

  const deleteMutation = useDeleteBiDashboard({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listBiDashboards'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  const copyMutation = useCopyBiDashboard({
    onSuccess: () => {
      message.success(t('copySuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listBiDashboards'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('copyFailed'));
    },
  });

  const columns: ProColumns<BiDashboard>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 70,
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
      title: t('code'),
      dataIndex: 'code',
      width: 190,
      copyable: true,
    },
    {
      title: t('kind'),
      dataIndex: 'kind',
      width: 100,
      valueEnum: Object.fromEntries(
        Object.entries(KIND_TAGS).map(([value, meta]) => [value, { text: t(meta.key) }]),
      ),
      render: (_, record) => {
        const meta = KIND_TAGS[record.kind as string] ?? { color: 'default', key: 'kindDashboard' };
        return <Tag color={meta.color}>{t(meta.key)}</Tag>;
      },
    },
    {
      title: t('chartCount'),
      dataIndex: 'chartCount',
      width: 90,
      hideInSearch: true,
    },
    {
      title: t('themeKey'),
      dataIndex: 'themeKey',
      width: 110,
      hideInSearch: true,
      render: (_, record) => record.themeKey || '-',
    },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 90,
      hideInSearch: true,
      render: (_, record) => (
        <Tag color={record.isEnabled ? 'success' : 'default'}>
          {record.isEnabled ? t('enabledTag') : t('disabledTag')}
        </Tag>
      ),
    },
    {
      title: t('isBuiltIn'),
      dataIndex: 'isBuiltIn',
      width: 90,
      hideInSearch: true,
      render: (_, record) => (
        <Tag color={record.isBuiltIn ? 'gold' : 'default'}>
          {record.isBuiltIn ? t('builtInTag') : t('customTag')}
        </Tag>
      ),
    },
    {
      title: t('version'),
      dataIndex: 'version',
      width: 80,
      hideInSearch: true,
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 220,
      fixed: 'right',
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'design',
              label: t('openDesigner'),
              icon: <BgColorsOutlined />,
              onClick: () => {
                setDrawerMode('edit');
                setSelected(record);
                setDrawerOpen(true);
              },
            },
            {
              key: 'copy',
              label: t('common:button.copy'),
              icon: <CopyOutlined />,
              onClick: () => record.id && copyMutation.mutate({ id: record.id }),
            },
            {
              key: 'delete',
              label: t('common:button.delete'),
              icon: <DeleteOutlined />,
              danger: true,
              hidden: !!record.isBuiltIn,
              confirm: {
                title: t('deleteConfirmTitle'),
                description: t('deleteConfirmDesc', { name: record.name || record.code || '' }),
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
          <ListTable<BiDashboard>
            actionRef={actionRef}
            columns={columns}
            request={async (params, sorter) => {
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

              const response = await fetchListBiDashboards(query);

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
              <TableExportButton
                key="export"
                fetcher={fetchListBiDashboards}
                columns={columns}
                filename="bi-dashboards"
              />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelected(undefined);
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
              x: 1200,
            }}
          />
        </div>
      </ContentContainer>

      <BiDesignerDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selected}
        onClose={() => {
          setDrawerOpen(false);
          setSelected(undefined);
        }}
        onSuccess={() => {
          setDrawerOpen(false);
          setSelected(undefined);
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default BiDashboardManagement;
