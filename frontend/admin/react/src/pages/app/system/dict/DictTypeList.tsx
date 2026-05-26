import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { fetchListDictTypes, useDeleteDictType } from '@/api/hooks/dict';
import { enableBoolOptions, getEnableColor, getEnableLabel } from './constants';
import DictTypeDrawer from './DictTypeDrawer';

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
      width: 150,
      fixed: 'left',
    },
    {
      title: t('typeCode'),
      dataIndex: 'typeCode',
      width: 150,
    },
    {
      title: t('status'),
      dataIndex: 'isEnabled',
      width: 95,
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
      width: 90,
      fixed: 'right',
      render: (_, record) => [
        <a
          key="edit"
          onClick={(e) => {
            e.stopPropagation();
            setEditingType(record);
            setDrawerMode('edit');
            setDrawerOpen(true);
          }}
        >
          <EditOutlined />
        </a>,
        <Popconfirm
          key="delete"
          title={t('deleteConfirmTitle')}
          description={t('deleteConfirmDesc', { moduleName: t('moduleName') })}
          onConfirm={(e) => {
            e?.stopPropagation();
            record.id && deleteMutation.mutate({ ids: [record.id] });
          }}
          onCancel={(e) => e?.stopPropagation()}
          okText={t('common:button.ok')}
          cancelText={t('common:button.cancel')}
        >
          <a style={{ color: '#ff4d4f' }} onClick={(e) => e.stopPropagation()}>
            <DeleteOutlined />
          </a>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <>
      <div ref={containerRef} className="page-container-content" style={{ padding: '0 8px', height: '100%' }}>
        <ProTable<any>
          actionRef={actionRef}
          columns={columns}
          headerTitle={t('dictTypeList')}
          request={async (params) => {
            try {
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
                total: response.total || 0,
                success: true,
              };
            } catch (error: any) {
              message.error(error.message || t('fetchFailed'));
              return { data: [], total: 0, success: false };
            }
          }}
          rowKey="id"
          search={{
            labelWidth: 'auto',
            defaultCollapsed: false,
            span: 24,
          }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
          toolBarRender={() => [
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              size="small"
              onClick={() => {
                setEditingType(null);
                setDrawerMode('create');
                setDrawerOpen(true);
              }}
            >
              {t('create')}
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
          scroll={{ y: tableScrollY, x: 500 }}
          onRow={(record) => ({
            onClick: () => {
              onTypeSelect(record.id);
            },
            style: {
              cursor: 'pointer',
              outline: record.id === currentTypeId ? '2px solid var(--ant-color-primary)' : undefined,
              outlineOffset: '-2px',
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
