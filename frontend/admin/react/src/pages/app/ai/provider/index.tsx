import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import ListTable from '@/components/common/ListTable';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import { Button, Tag, Tooltip, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined, CloudOutlined, LaptopOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { aiservicev1_AiProvider as AiProvider } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchListAiProviders, useDeleteAiProvider } from '@/api/hooks/ai-provider';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { TABLE } from '@/config/constants';
import AiProviderDrawer from './components/AiProviderDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';

export default function AiProviderPage() {
  const { t } = useTranslation('aiProvider');
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<AiProvider | undefined>(undefined);

  const deleteMutation = useDeleteAiProvider({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listAiProviders'] });
    },
    onError: (error: Error) => message.error(error.message || t('fetchFailed')),
  });

  const columns: ProColumns[] = [
    {
      title: t('name'),
      dataIndex: 'name',
      render: (_, record) => (
        <span className="inline-flex items-center gap-2">
          {record.modelType === 'CLOUD' ? (
            <CloudOutlined style={{ color: 'var(--gowind-primary-text)' }} />
          ) : (
            // 内联 style 同上：.anticon 的无层规则会压过 Tailwind 颜色类，
            // 原先的 text-green-600 实际从未生效（本地/云端图标一直同色）。
            <LaptopOutlined style={{ color: 'var(--gowind-success-text)' }} />
          )}
          {record.name}
          {record.isDefault && (
            <Tag color="blue" className="ml-1">
              {t('isDefault')}
            </Tag>
          )}
        </span>
      ),
    },
    {
      title: t('modelType'),
      dataIndex: 'modelType',
      width: 110,
      render: (_, record) =>
        record.modelType === 'CLOUD' ? (
          <Tag color="blue">{t('modelTypeCloud')}</Tag>
        ) : (
          <Tag color="green">{t('modelTypeLocal')}</Tag>
        ),
    },
    { title: t('modelName'), dataIndex: 'modelName', ellipsis: true },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 90,
      render: (_, record) =>
        record.isEnabled ? (
          <Tag color="success">{t('statusMap.ON')}</Tag>
        ) : (
          <Tag color="default">{t('statusMap.OFF')}</Tag>
        ),
    },
    {
      title: 'Endpoint',
      dataIndex: 'baseUrl',
      ellipsis: true,
      render: (_, record) =>
        record.modelType === 'CLOUD'
          ? record.baseUrl || '-'
          : `${record.localHost || 'localhost'}:${record.localPort ?? 11434}`,
    },
    { title: t('remark'), dataIndex: 'remark', ellipsis: true },
    {
      title: t('action'),
      valueType: 'option',
      width: 150,
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
                setSelected(record);
                setDrawerMode('edit');
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
              onClick: () => deleteMutation.mutate({ id: record.id! }),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
      <ListTable<AiProvider>
        actionRef={actionRef}
        rowKey="id"
        search={{ optionRender: searchOptionRender, labelWidth: 'auto', defaultCollapsed: false }}
        scroll={{ y: tableScrollY, x: 1000 }}
        request={async (params) => {
          const { current, pageSize, ...rest } = params;
          const query = new PaginationQuery({
            paging: { page: current || 1, pageSize: pageSize || TABLE.DEFAULT_PAGE_SIZE },
            formValues: rest,
          });
          const res = await fetchListAiProviders(query);
          return { data: res.items || [], total: Number(res.total ?? 0), success: true };
        }}
        toolBarRender={() => [
          <TableExportButton key="export" fetcher={fetchListAiProviders} columns={columns} filename="ai-providers" />,
          <Tooltip key="create-tip" title={t('isDefaultTooltip')}>
            <Button
              key="create"
              color="primary"
              variant="outlined"
              icon={<PlusOutlined />}
              onClick={() => {
                setSelected(undefined);
                setDrawerMode('create');
                setDrawerOpen(true);
              }}
            >
              {t('common:button.add')}
            </Button>
          </Tooltip>,
        ]}
        columns={columns}
      />
      <AiProviderDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selected}
        onClose={() => setDrawerOpen(false)}
        onSuccess={() => {
          setDrawerOpen(false);
          actionRef.current?.reload();
          queryClient.invalidateQueries({ queryKey: ['listAiProviders'] });
        }}
      />
      </div>
    </ContentContainer>
  );
}
