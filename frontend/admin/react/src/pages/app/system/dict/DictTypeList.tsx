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
import { fetchListDictTypes, useDeleteDictType } from '@/api/hooks/dict';
import { enableBoolOptions, getEnableColor, getEnableLabel } from './constants';
import DictTypeDrawer from './DictTypeDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';

interface DictTypeListProps {
  currentTypeId: number | null;
  onTypeSelect: (typeId: number) => void;
}

/**
 * 字典类型列表（左侧）
 */
const DictTypeList: React.FC<DictTypeListProps> = ({ currentTypeId, onTypeSelect }) => {
  const { t } = useTranslation('dict-type');
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [editingType, setEditingType] = useState<any>(null);

  // 删除 mutation
  const deleteMutation = useDeleteDictType({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listDictTypes'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<any>[] = [
    {
      title: t('typeName'),
      dataIndex: 'typeName',
    },
    {
      title: t('typeCode'),
      dataIndex: 'typeCode',
    },
    {
      title: t('status'),
      dataIndex: 'isEnabled',
      valueType: 'select',
      fieldProps: {
        options: enableBoolOptions(t),
      },
      render: (_, record) => (
        <Tag color={getEnableColor(record.isEnabled)}>
          {getEnableLabel(t, record.isEnabled)}
        </Tag>
      ),
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
                setEditingType(record);
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
              onClick: () => record.id && deleteMutation.mutate({ ids: [record.id] }),
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
          headerTitle={t('dictTypeList')}
          request={async (params) => {
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
            });

            const response = await fetchListDictTypes(query);

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
            <TableExportButton key="export" fetcher={fetchListDictTypes} columns={columns} filename="dict-types" />,
            <Button
              key="create"
              color="primary"
              variant="outlined"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditingType(null);
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
              onTypeSelect(record.id);
            },
            className: record.id === currentTypeId ? 'ant-table-row-selected' : undefined,
            style: {
              cursor: 'pointer',
            },
          })}
        />
      </div>

      <DictTypeDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={editingType}
        onClose={() => {
          setDrawerOpen(false);
          setEditingType(null);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default DictTypeList;
