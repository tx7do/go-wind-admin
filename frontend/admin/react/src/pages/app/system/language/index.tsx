import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { dictservicev1_Language as Language } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListLanguages, useDeleteLanguage } from '@/api/hooks/language';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { enableBoolToColor } from './constants';
import LanguageDrawer from './components/LanguageDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 语言管理页面
 */
const LanguageManagement = () => {
  const { t } = useTranslation('language');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedLanguage, setSelectedLanguage] = useState<Language | undefined>();

  // 删除操作
  const deleteMutation = useDeleteLanguage({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listLanguages'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<Language>[] = [
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
      title: t('nativeName'),
      dataIndex: 'nativeName',
      width: 120,
    },
    {
      title: t('languageName'),
      dataIndex: 'languageName',
      width: 150,
    },
    {
      title: t('languageCode'),
      dataIndex: 'languageCode',
      width: 150,
    },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 100,
      hideInSearch: true,
      render: (_, record) => {
        const enabled = record.isEnabled as boolean;
        return <Tag color={enableBoolToColor(enabled)}>{enabled ? t('yes') : t('no')}</Tag>;
      },
    },
    {
      title: t('isDefault'),
      dataIndex: 'isDefault',
      width: 100,
      hideInSearch: true,
      render: (_, record) => {
        const isDefault = record.isDefault as boolean;
        return <Tag color={enableBoolToColor(isDefault)}>{isDefault ? t('yes') : t('no')}</Tag>;
      },
    },
    {
      title: t('sortOrder'),
      dataIndex: 'sortOrder',
      width: 100,
      hideInSearch: true,
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
                setSelectedLanguage(record);
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
                description: t('deleteConfirmDesc', { name: record.languageName || '' }),
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
          <ListTable<Language>
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

              const response = await fetchListLanguages(query);

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
              <TableExportButton key="export" fetcher={fetchListLanguages} columns={columns} filename="languages" />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedLanguage(undefined);
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
              x: 1100,
            }}
          />
        </div>
      </ContentContainer>

      {/* 语言编辑/创建 Drawer */}
      <LanguageDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedLanguage}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedLanguage(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default LanguageManagement;
