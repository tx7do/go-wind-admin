import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import ListTable from '@/components/common/ListTable';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import { Button, Modal, Tag, Typography, App } from 'antd';
import {
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { access_keyservicev1_AccessKey as AccessKey } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import {
  fetchListAccessKeys,
  useDeleteAccessKey,
  useResetAccessKeySecret,
} from '@/api/hooks/access-key';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import AccessKeyDrawer from './components/AccessKeyDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';

export default function AccessKeyPage() {
  const { t } = useTranslation('access-key');
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<AccessKey>();

  const resetMutation = useResetAccessKeySecret();
  const [resetSecretFor, setResetSecretFor] = useState<AccessKey>();
  const [resetSecretValue, setResetSecretValue] = useState<string | null>(null);

  const deleteMutation = useDeleteAccessKey({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listAccessKeys'] });
    },
    onError: (error: Error) => message.error(error.message || t('fetchFailed')),
  });

  const statusMap: Record<string, { text: string; color: string }> = {
    ON: { text: t('statusMap.ON'), color: 'success' },
    OFF: { text: t('statusMap.OFF'), color: 'default' },
  };

  const columns: ProColumns<AccessKey>[] = [
    { title: t('name'), dataIndex: 'name', ellipsis: true },
    {
      title: t('accessKey'),
      dataIndex: 'accessKey',
      copyable: true,
      ellipsis: true,
    },
    {
      title: t('status'),
      dataIndex: 'status',
      width: 90,
      render: (_, record) => {
        const cfg = statusMap[record.status as string] ?? {
          text: record.status,
          color: 'default',
        };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      title: t('expiresAt'),
      dataIndex: 'expiresAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('lastUsedAt'),
      dataIndex: 'lastUsedAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 250,
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
                setSelected(record);
                setDrawerOpen(true);
              },
            },
            {
              key: 'reset',
              label: t('resetSecret'),
              icon: <ReloadOutlined />,
              onClick: () => {
                resetMutation.mutate(
                  { id: record.id! },
                  {
                    onSuccess: (resp) => {
                      message.success(t('resetSuccess'));
                      setResetSecretFor(record);
                      setResetSecretValue(resp.secret ?? null);
                    },
                    onError: (error: Error) => message.error(error.message || t('resetFailed')),
                  },
                );
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
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ListTable<AccessKey>
          actionRef={actionRef}
          columns={columns}
          rowKey="id"
          search={{ optionRender: searchOptionRender, labelWidth: 'auto', defaultCollapsed: false }}
          scroll={{ y: tableScrollY }}
          pagination={{
            defaultPageSize: 20,
            showSizeChanger: true,
          }}
          request={async (params) => {
            const { current, pageSize, ...rest } = params;
            const query = new PaginationQuery({
              paging: {
                page: current || 1,
                pageSize: pageSize || 20,
              },
              formValues: rest,
            });
            const res = await fetchListAccessKeys(query);
            return {
              data: res.items || [],
              total: Number(res.total ?? 0),
              success: true,
            };
          }}
          toolBarRender={() => [
            <TableExportButton key="export" fetcher={fetchListAccessKeys} columns={columns} filename="access-keys" />,
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
          size="middle"
          bordered
        />
      </div>
      <Modal
        title={t('secretDialogTitle')}
        open={resetSecretValue !== null}
        onOk={() => {
          setResetSecretValue(null);
          setResetSecretFor(undefined);
          actionRef.current?.reload();
        }}
        onCancel={() => {
          setResetSecretValue(null);
          setResetSecretFor(undefined);
        }}
        cancelButtonProps={{ style: { display: 'none' } }}
        width={560}
      >
        <Typography.Paragraph type="warning">
          {t('secretDialogHint')}
        </Typography.Paragraph>
        <Typography.Paragraph copyable={{ text: resetSecretValue ?? '' }}>
          <Typography.Text code style={{ wordBreak: 'break-all' }}>
            {resetSecretValue}
          </Typography.Text>
        </Typography.Paragraph>
        <Typography.Text type="secondary">
          {t('resetSecretFor')}:{' '}
          {resetSecretFor?.accessKey}
        </Typography.Text>
      </Modal>
      <AccessKeyDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selected}
        onClose={() => setDrawerOpen(false)}
        onSuccess={() => actionRef.current?.reload()}
      />
    </ContentContainer>
  );
}
