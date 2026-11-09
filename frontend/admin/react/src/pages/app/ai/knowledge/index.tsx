import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import ListTable from '@/components/common/ListTable';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import { Button, Tag, App } from 'antd';
import { DeleteOutlined, EditOutlined, FileTextOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchListAiKnowledgeBases, useDeleteAiKnowledgeBase } from '@/api/hooks/ai-knowledge';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { TABLE } from '@/config/constants';
import KnowledgeDrawer from './components/KnowledgeDrawer';
import DocsDrawer from './components/DocsDrawer';

export default function AiKnowledgePage() {
  const { t } = useTranslation('aiKnowledge');
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<AiKnowledgeBase | undefined>(undefined);
  const [docsTarget, setDocsTarget] = useState<AiKnowledgeBase | undefined>(undefined);
  const [docsOpen, setDocsOpen] = useState(false);

  const deleteMutation = useDeleteAiKnowledgeBase({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listAiKnowledgeBases'] });
    },
    onError: (error: Error) => message.error(error.message || t('fetchFailed')),
  });

  const columns: ProColumns[] = [
    { title: t('name'), dataIndex: 'name', minWidth: 160 },
    { title: t('description'), dataIndex: 'description', ellipsis: true },
    { title: t('embeddingModel'), dataIndex: 'embeddingModel', width: 180 },
    {
      title: t('docCount'),
      dataIndex: 'docCount',
      width: 90,
      render: (_, record) => <Tag color="blue">{record.docCount ?? 0}</Tag>,
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 230,
      fixed: 'right',
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'docs',
              label: t('docs'),
              icon: <FileTextOutlined />,
              onClick: () => {
                setDocsTarget(record);
                setDocsOpen(true);
              },
            },
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
              confirm: { title: t('deleteConfirmTitle'), description: t('deleteConfirmDesc') },
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
        <ListTable<AiKnowledgeBase>
          actionRef={actionRef}
          rowKey="id"
          search={false}
          scroll={{ y: tableScrollY, x: 900 }}
          request={async (params) => {
            const { current, pageSize } = params;
            const query = new PaginationQuery({
              paging: { page: current || 1, pageSize: pageSize || TABLE.DEFAULT_PAGE_SIZE },
            });
            const res = await fetchListAiKnowledgeBases(query);
            return { data: res.items || [], total: Number(res.total ?? 0), success: true };
          }}
          toolBarRender={() => [
            <TableExportButton key="export" fetcher={fetchListAiKnowledgeBases} columns={columns} filename="ai-knowledge-bases" />,
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
            </Button>,
          ]}
          columns={columns}
        />
        <KnowledgeDrawer
          open={drawerOpen}
          mode={drawerMode}
          data={selected}
          onClose={() => setDrawerOpen(false)}
          onSuccess={() => {
            setDrawerOpen(false);
            actionRef.current?.reload();
            queryClient.invalidateQueries({ queryKey: ['listAiKnowledgeBases'] });
          }}
        />
        <DocsDrawer
          open={docsOpen}
          data={docsTarget}
          onClose={() => setDocsOpen(false)}
        />
      </div>
    </ContentContainer>
  );
}
