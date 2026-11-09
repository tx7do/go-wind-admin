import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { internal_messageservicev1_InternalMessageCategory as Category } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchListMessageCategories, useDeleteMessageCategory } from '@/api/hooks/internal-message';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { enableBoolToColor, getEnableBoolMap } from './constants';
import CategoryDrawer from './components/CategoryDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 内部消息分类管理页面
 */
const InternalMessageCategoryManagement = () => {
  const { t } = useTranslation('message-category');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedCategory, setSelectedCategory] = useState<Category | undefined>();

  const enableBoolMap = getEnableBoolMap(t);

  // 删除操作
  const deleteMutation = useDeleteMessageCategory({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listMessageCategories'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<Category>[] = [
    {
      title: t('name'),
      dataIndex: 'name',
    },
    {
      title: t('code'),
      dataIndex: 'code',
    },
    {
      title: t('sortOrder'),
      dataIndex: 'sortOrder',
      width: 70,
      hideInSearch: true,
    },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 95,
      hideInSearch: true,
      render: (_, record) => {
        const boolValue = !!record.isEnabled;
        const enabled = String(boolValue) as keyof typeof enableBoolMap;
        const config = enableBoolMap[enabled];
        return <Tag color={enableBoolToColor(boolValue)}>{config?.text || String(enabled)}</Tag>;
      },
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('remark'),
      dataIndex: 'remark',
      hideInSearch: true,
      ellipsis: true,
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
                setSelectedCategory(record);
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
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ListTable<Category>
            actionRef={actionRef}
            columns={columns}
            request={async (params, _sorter, _filter) => {
              const query = new PaginationQuery({
                formValues: Object.fromEntries(
                  Object.entries(params).filter(
                    ([key]) => !['current', 'pageSize'].includes(key),
                  ),
                ),
              });

              const response = await fetchListMessageCategories(query);

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
            pagination={false}
            toolBarRender={() => [
              <TableExportButton key="export" fetcher={fetchListMessageCategories} columns={columns} filename="im-categories" />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedCategory(undefined);
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
            scroll={{ y: tableScrollY }}
          />
        </div>
      </ContentContainer>

      {/* 分类编辑/创建 Drawer */}
      <CategoryDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedCategory}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedCategory(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default InternalMessageCategoryManagement;
