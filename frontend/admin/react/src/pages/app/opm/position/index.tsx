import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import {
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { identityservicev1_Position as Position } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchListPositions, useCreatePosition, useDeletePosition } from '@/api/hooks/position';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import ImportModal from '@/components/common/ImportModal';
import type { ImportField } from '@/utils/import';
import {
  getStatusMap,
  getStatusOptions,
  getPositionTypeMap,
  getPositionTypeOptions,
} from './constants';
import PositionDrawer from './components/PositionDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 职位管理页面
 */
const PositionManagement = () => {
  const { t } = useTranslation('position');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedPosition, setSelectedPosition] = useState<Position | undefined>();

  // Excel 导入：字段与创建表单一致，唯排除 orgUnitId（外键需名称解析，属后续演进）
  const [importOpen, setImportOpen] = useState(false);
  const createMutation = useCreatePosition();
  const importFields: ImportField[] = [
    { label: t('name'), prop: 'name' },
    { label: t('code'), prop: 'code' },
    { label: t('type'), prop: 'type' },
    { label: t('status'), prop: 'status' },
    { label: t('headcount'), prop: 'headcount' },
    { label: t('sortOrder'), prop: 'sortOrder' },
    { label: t('description'), prop: 'description' },
    { label: t('remark'), prop: 'remark' },
  ];

  const statusMap = getStatusMap(t);
  const positionTypeMap = getPositionTypeMap(t);

  // 删除操作
  const deleteMutation = useDeletePosition({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listPositions'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<Position>[] = [
    {
      title: t('name'),
      dataIndex: 'name',
    },
    {
      title: t('code'),
      dataIndex: 'code',
    },
    {
      title: t('type'),
      dataIndex: 'type',
      width: 95,
      valueType: 'select',
      fieldProps: {
        options: getPositionTypeOptions(t),
      },
      render: (_, record) => {
        const type = record.type as keyof typeof positionTypeMap;
        const config = positionTypeMap[type] || { text: type, color: 'default' };
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
      title: t('orgUnitName'),
      dataIndex: 'orgUnitName',
      width: 150,
      hideInSearch: true,
    },
    {
      title: t('headcount'),
      dataIndex: 'headcount',
      width: 80,
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
      title: t('sortOrder'),
      dataIndex: 'sortOrder',
      width: 70,
      hideInSearch: true,
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      valueType: 'dateTime',
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
                setSelectedPosition(record);
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
          <ListTable<Position>
            actionRef={actionRef}
            columns={columns}
            request={async (params, _sorter, _filter) => {
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || 20,
                },
                formValues: Object.fromEntries(
                  Object.entries(params).filter(
                    ([key]) => !['current', 'pageSize'].includes(key),
                  ),
                ),
              });

              const response = await fetchListPositions(query);

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
              defaultPageSize: 20,
              showSizeChanger: true,
              showQuickJumper: true,
            }}
            toolBarRender={() => [
              <TableExportButton key="export" fetcher={fetchListPositions} columns={columns} filename="positions" />,
              <Button
                key="import"
                icon={<UploadOutlined />}
                onClick={() => setImportOpen(true)}
              >
                {t('common:import.title')}
              </Button>,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedPosition(undefined);
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

      {/* 职位编辑/创建 Drawer */}
      <PositionDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedPosition}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedPosition(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />

      {/* Excel 导入 */}
      <ImportModal
        open={importOpen}
        fields={importFields}
        createRow={(values) => createMutation.mutateAsync({ data: values })}
        onClose={() => setImportOpen(false)}
        onSuccess={() => {
          actionRef.current?.reload();
          queryClient.invalidateQueries({ queryKey: ['listPositions'] });
        }}
      />
    </>
  );
};

export default PositionManagement;
