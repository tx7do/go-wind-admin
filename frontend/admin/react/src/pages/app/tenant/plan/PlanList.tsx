import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { fetchListPlans, useDeletePlan } from '@/api/hooks/plan';
import {
  getPlanVersionMap,
  getPlanVersionOptions,
  getExpiryPolicyMap,
  getExpiryPolicyOptions,
} from './constants';
import PlanDrawer from './components/PlanDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';

interface PlanListProps {
  currentPlanId: number | null;
  onPlanSelect: (planId: number) => void;
}

/**
 * 套餐目录列表（左侧）
 */
const PlanList: React.FC<PlanListProps> = ({ currentPlanId, onPlanSelect }) => {
  const { t } = useTranslation('plan');
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [editingPlan, setEditingPlan] = useState<any>(null);

  const versionMap = getPlanVersionMap(t);
  const expiryPolicyMap = getExpiryPolicyMap(t);

  // 删除 mutation
  const deleteMutation = useDeletePlan({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listPlans'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<any>[] = [
    {
      title: t('name'),
      dataIndex: 'name',
    },
    {
      title: t('versionLabel'),
      dataIndex: 'version',
      valueType: 'select',
      fieldProps: {
        options: getPlanVersionOptions(t),
      },
      render: (_, record) => {
        const item = (versionMap as Record<string, { text: string; color: string }>)[record.version as string];
        return item ? <Tag color={item.color}>{item.text}</Tag> : '-';
      },
    },
    {
      title: t('expiryPolicyLabel'),
      dataIndex: 'expiryPolicy',
      valueType: 'select',
      fieldProps: {
        options: getExpiryPolicyOptions(t),
      },
      render: (_, record) => {
        const item = (expiryPolicyMap as Record<string, { text: string; color: string }>)[record.expiryPolicy as string];
        return item ? <Tag color={item.color}>{item.text}</Tag> : '-';
      },
    },
    {
      title: t('dataRetentionDays'),
      dataIndex: 'dataRetentionDays',
      hideInSearch: true,
    },
    {
      title: t('description'),
      dataIndex: 'description',
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('remark'),
      dataIndex: 'remark',
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
                setEditingPlan(record);
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
        <ListTable<any>
          actionRef={actionRef}
          columns={columns}
          headerTitle={t('planList')}
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
              formValues: Object.fromEntries(
                Object.entries(params).filter(
                  ([key]) => !['current', 'pageSize'].includes(key),
                ),
              ),
              orderBy,
            });

            const response = await fetchListPlans(query);

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
            span: 12,
          }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
          toolBarRender={() => [
            <TableExportButton key="export" fetcher={fetchListPlans} columns={columns} filename="plans" />,
            <Button
              key="create"
              color="primary"
              variant="outlined"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditingPlan(null);
                setDrawerMode('create');
                setDrawerOpen(true);
              }}
            >
              {t('common:button.add')}
            </Button>,
          ]}
          options={{
            density: false,
            fullScreen: false,
            setting: false,
            reload: true,
          }}
          size="small"
          bordered
          cardBordered={false}
          scroll={{ y: tableScrollY }}
          onRow={(record) => ({
            onClick: () => {
              onPlanSelect(record.id);
            },
            className: record.id === currentPlanId ? 'ant-table-row-selected' : undefined,
            style: {
              cursor: 'pointer',
            },
          })}
        />
      </div>

      <PlanDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={editingPlan}
        onClose={() => {
          setDrawerOpen(false);
          setEditingPlan(null);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default PlanList;
