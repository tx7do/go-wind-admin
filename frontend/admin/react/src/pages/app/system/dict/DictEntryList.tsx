import { useRef, useState, useEffect } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, App, Empty } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { fetchListDictEntries, useDeleteDictEntry } from '@/api/hooks/dict';
import { enableBoolOptions, getEnableColor, getEnableLabel, getEntryLabel } from './constants';
import DictEntryDrawer from './DictEntryDrawer';
import { usePreferencesStore } from '@/core/preferences/store';

interface DictEntryListProps {
  typeId: number | null;
}

/**
 * 字典条目列表（右侧）
 */
const DictEntryList: React.FC<DictEntryListProps> = ({ typeId }) => {
  const { t } = useTranslation('dict-entry');
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  // 获取当前语言
  const locale = usePreferencesStore((state) => state.preferences.app.locale);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [editingEntry, setEditingEntry] = useState<any>(null);

  // 当 typeId 变化时自动刷新列表
  useEffect(() => {
    if (typeId) {
      actionRef.current?.reload();
    }
  }, [typeId]);

  // 删除 mutation
  const deleteMutation = useDeleteDictEntry({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listDictEntries'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<any>[] = [
    {
      title: t('entryLabel'),
      dataIndex: 'entryLabel',
      width: 150,
      fixed: 'left',
      render: (_, record) => getEntryLabel(record, locale),
    },
    {
      title: t('entryValue'),
      dataIndex: 'entryValue',
      width: 150,
    },
    {
      title: t('numericValue'),
      dataIndex: 'numericValue',
      width: 95,
      hideInSearch: true,
    },
    {
      title: t('sortOrder'),
      dataIndex: 'sortOrder',
      width: 95,
      hideInSearch: true,
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
          onClick={() => {
            setEditingEntry(record);
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
          onConfirm={() => record.id && deleteMutation.mutate({ ids: [record.id] })}
          okText={t('common:button.ok')}
          cancelText={t('common:button.cancel')}
        >
          <a style={{ color: '#ff4d4f' }}><DeleteOutlined /></a>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <>
      <div ref={containerRef} className="page-container-content" style={{ padding: '0 8px', height: '100%' }}>
        {typeId ? (
          <ProTable<any>
            actionRef={actionRef}
            columns={columns}
            headerTitle={false}
            params={{ typeId }}
            request={async (params) => {
              try {
                const query = new PaginationQuery({
                  paging: {
                    page: params.current || 1,
                    pageSize: params.pageSize || TABLE.DEFAULT_PAGE_SIZE,
                  },
                  formValues: {
                    ...Object.fromEntries(
                      Object.entries(params).filter(
                        ([key]) => !['current', 'pageSize', 'typeId'].includes(key),
                      ),
                    ),
                    type_id: typeId,
                  },
                });

                const response = await fetchListDictEntries(query);

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
                  setEditingEntry(null);
                  setDrawerMode('create');
                  setDrawerOpen(true);
                }}
              >
                {t('create')}
              </Button>,
            ]}
            options={{
              density: true,
              fullScreen: true,
              setting: true,
              reload: true,
            }}
            size="small"
            bordered
            cardBordered={false}
            scroll={{ y: tableScrollY, x: 700 }}
          />
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
            <Empty description={t('selectTypeFirst')} />
          </div>
        )}
      </div>

      <DictEntryDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={editingEntry}
        typeId={typeId!}
        onClose={() => {
          setDrawerOpen(false);
          setEditingEntry(null);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default DictEntryList;
