import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { authenticationservicev1_LoginPolicy as LoginPolicy } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListLoginPolicies, useDeleteLoginPolicy } from '@/api/hooks/login-policy';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getPolicyTypeMap, getPolicyMethodMap, getPolicyMethodOptions } from './constants';
import LoginPolicyDrawer from './components/LoginPolicyDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 登录策略管理页面
 */
const LoginPolicyManagement = () => {
  const { t } = useTranslation('login-policy');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedPolicy, setSelectedPolicy] = useState<LoginPolicy | undefined>();

  const policyTypeMap = getPolicyTypeMap(t);
  const policyMethodMap = getPolicyMethodMap(t);

  // 删除操作
  const deleteMutation = useDeleteLoginPolicy({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listLoginPolicies'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<LoginPolicy>[] = [
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
      title: t('targetId'),
      dataIndex: 'targetId',
      hideInSearch: true,
    },
    {
      title: t('type'),
      dataIndex: 'type',
      width: 100,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(policyTypeMap).map(([key, config]) => [
          key,
          { text: config.text, status: key === 'ALLOW' ? 'Success' : 'Error' },
        ]),
      ),
      render: (_, record) => {
        const type = record.type as keyof typeof policyTypeMap;
        const config = policyTypeMap[type] || { text: type, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('method'),
      dataIndex: 'method',
      width: 120,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        // 与抽屉同源的过滤选项：MAC 不可选（HTTP 请求上下文拿不到，选了也永不命中），
        // 完整映射 policyMethodMap 仅用于存量 MAC 行的标签渲染
        getPolicyMethodOptions(t).map(({ label, value }) => [
          value,
          { text: label, status: 'Default' as const },
        ]),
      ),
      render: (_, record) => {
        const method = record.method as keyof typeof policyMethodMap;
        const config = policyMethodMap[method] || { text: method, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('value'),
      dataIndex: 'value',
      hideInSearch: true,
    },
    {
      title: t('reason'),
      dataIndex: 'reason',
      hideInSearch: true,
      ellipsis: true,
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
                setSelectedPolicy(record);
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
          <ListTable<LoginPolicy>
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

              const response = await fetchListLoginPolicies(query);

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
              <TableExportButton key="export" fetcher={fetchListLoginPolicies} columns={columns} filename="login-policies" />,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedPolicy(undefined);
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
              x: 1000,
            }}
          />
        </div>
      </ContentContainer>

      {/* 登录策略编辑/创建 Drawer */}
      <LoginPolicyDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedPolicy}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedPolicy(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default LoginPolicyManagement;
