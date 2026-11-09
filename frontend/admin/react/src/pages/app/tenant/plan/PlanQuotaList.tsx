import { useRef, useState, useEffect } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App, Empty } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { fetchListPlanQuotas, useDeletePlanQuota } from '@/api/hooks/plan';
import { getQuotaTypeMap, getQuotaTypeOptions } from './constants';
import PlanQuotaDrawer from './components/PlanQuotaDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

interface PlanQuotaListProps {
  planId: number | null;
}

/**
 * 套餐配额列表（右侧）
 */
const PlanQuotaList: React.FC<PlanQuotaListProps> = ({ planId }) => {
  const { t } = useTranslation('plan');
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const quotaTypeMap = getQuotaTypeMap(t);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [editingQuota, setEditingQuota] = useState<any>(null);

  // 当 planId 变化时自动刷新列表
  useEffect(() => {
    if (planId) {
      actionRef.current?.reload();
    }
  }, [planId]);

  // 删除 mutation
  const deleteMutation = useDeletePlanQuota({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listPlanQuotas'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<any>[] = [
    {
      title: t('quotaTypeLabel'),
      dataIndex: 'quotaType',
      width: 120,
      valueType: 'select',
      fieldProps: {
        options: getQuotaTypeOptions(t),
      },
      render: (_, record) => {
        const item = (quotaTypeMap as Record<string, { text: string; color: string }>)[record.quotaType as string];
        return item ? <Tag color={item.color}>{item.text}</Tag> : '-';
      },
    },
    {
      title: t('quotaValue'),
      dataIndex: 'quotaValue',
      width: 120,
      hideInSearch: true,
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 150,
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
                setEditingQuota(record);
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
              onClick: () => record.id && deleteMutation.mutate({ id: record.id }),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <>
      <div ref={containerRef} className="page-container-content" style={{ padding: '0 8px', height: '100%' }}>
        {planId ? (
          <ListTable<any>
            actionRef={actionRef}
            columns={columns}
            headerTitle={false}
            params={{ planId }}
            request={async (params, sorter) => {
              const orderBy: string[] = [];
              if (sorter && Object.keys(sorter).length > 0) {
                for (const key in sorter) {
                  orderBy.push((sorter[key] === 'ascend' ? '' : '-') + key);
                }
              }
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || TABLE.DEFAULT_PAGE_SIZE,
                },
                formValues: {
                  ...Object.fromEntries(
                    Object.entries(params).filter(
                      ([key]) => !['current', 'pageSize', 'planId'].includes(key),
                    ),
                  ),
                  plan_id: planId,
                },
                orderBy,
              });

              const response = await fetchListPlanQuotas(query);

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
              <TableExportButton key="export" fetcher={fetchListPlanQuotas} columns={columns} filename="plan-quotas" />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditingQuota(null);
                  setDrawerMode('create');
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
            size="small"
            bordered
            cardBordered={false}
            scroll={{ y: tableScrollY }}
          />
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
            <Empty description={t('selectPlanFirst')} />
          </div>
        )}
      </div>

      <PlanQuotaDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={editingQuota}
        planId={planId!}
        onClose={() => {
          setDrawerOpen(false);
          setEditingQuota(null);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default PlanQuotaList;
