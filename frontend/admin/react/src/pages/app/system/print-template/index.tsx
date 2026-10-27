import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Tag, Button, App } from 'antd';
import {
  EditOutlined,
  DeleteOutlined,
  CopyOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { printservicev1_PrintTemplate as PrintTemplate } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListPrintTemplates, useDeletePrintTemplate } from '@/api/hooks/print-template';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { orientationLabel } from './constants';
import TemplateDesignerDrawer from './components/TemplateDesignerDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 打印模板管理页（单据打印设计器 TemplateData 载体；内置模板禁删可复制派生）
 */
const PrintTemplateManagement = () => {
  const { t } = useTranslation('printTemplate');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit' | 'copy'>('create');
  const [selectedTemplate, setSelectedTemplate] = useState<PrintTemplate | undefined>();

  // 删除操作
  const deleteMutation = useDeletePrintTemplate({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listPrintTemplates'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<PrintTemplate>[] = [
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
      width: 200,
    },
    {
      title: t('code'),
      dataIndex: 'code',
      width: 200,
      copyable: true,
    },
    {
      title: t('resourceKey'),
      dataIndex: 'resourceKey',
      width: 160,
      hideInSearch: true,
      render: (_, record) => record.resourceKey || '-',
    },
    {
      title: t('paperSize'),
      dataIndex: 'paperSize',
      width: 110,
      hideInSearch: true,
    },
    {
      title: t('orientation'),
      dataIndex: 'orientation',
      width: 110,
      hideInSearch: true,
      render: (_, record) => orientationLabel(t, record.orientation),
    },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 100,
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
      width: 200,
      fixed: 'right',
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
                setSelectedTemplate(record);
                setDrawerOpen(true);
              },
            },
            {
              key: 'copy',
              label: t('common:button.copy'),
              icon: <CopyOutlined />,
              onClick: () => {
                setDrawerMode('copy');
                setSelectedTemplate(record);
                setDrawerOpen(true);
              },
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
          <ListTable<PrintTemplate>
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

              const response = await fetchListPrintTemplates(query);

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
              <TableExportButton
                key="export"
                fetcher={fetchListPrintTemplates}
                columns={columns}
                filename="print-templates"
              />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedTemplate(undefined);
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

      {/* 模板设计器 Drawer（create/edit/copy 三态） */}
      <TemplateDesignerDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedTemplate}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedTemplate(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default PrintTemplateManagement;
