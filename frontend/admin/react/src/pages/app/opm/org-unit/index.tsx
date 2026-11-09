import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App } from 'antd';
import {
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
  NodeExpandOutlined,
  NodeCollapseOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { identityservicev1_OrgUnit as OrgUnit } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchListOrgUnits, useDeleteOrgUnit } from '@/api/hooks/org-unit';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getStatusMap, getStatusOptions, getOrgTypeMap, getOrgTypeOptions } from './constants';
import OrgDrawer from './components/OrgDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 组织架构管理页面
 *
 * 树形表格展示组织层级结构，支持搜索、CRUD 操作
 */
const OrgUnitManagement = () => {
  const { t } = useTranslation('org-unit');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedOrg, setSelectedOrg] = useState<OrgUnit | undefined>();

  // 树数据
  const [treeData, setTreeData] = useState<OrgUnit[]>([]);

  const statusMap = getStatusMap(t);
  const orgTypeMap = getOrgTypeMap(t);

  // 删除操作
  const deleteMutation = useDeleteOrgUnit({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listOrgUnits'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 清理 API 返回树形数据中的空 children 数组
  const cleanEmptyChildren = (nodes: any[]): void => {
    nodes.forEach((n) => {
      if (n.children) {
        if (n.children.length === 0) {
          delete n.children;
        } else {
          cleanEmptyChildren(n.children);
        }
      }
    });
  };

  // 展开全部/折叠全部
  const [expandedRowKeys, setExpandedRowKeys] = useState<readonly React.Key[]>([]);

  const handleExpandAll = (data: OrgUnit[]) => {
    const keys: React.Key[] = [];
    const collectKeys = (items: OrgUnit[]) => {
      items.forEach((item) => {
        keys.push(item.id ?? '');
        if ((item as any).children?.length) {
          collectKeys((item as any).children);
        }
      });
    };
    collectKeys(data);
    setExpandedRowKeys(keys);
  };

  const handleCollapseAll = () => {
    setExpandedRowKeys([]);
  };

  // 列配置
  const columns: ProColumns<OrgUnit>[] = [
    {
      title: t('name'),
      dataIndex: 'name',
      width: 220,
      fixed: 'left',
    },
    {
      title: t('code'),
      dataIndex: 'code',
      width: 140,
    },
    {
      title: t('type'),
      dataIndex: 'type',
      width: 95,
      valueType: 'select',
      fieldProps: {
        options: getOrgTypeOptions(t),
      },
      render: (_, record) => {
        const type = record.type as keyof typeof orgTypeMap;
        const config = orgTypeMap[type] || { text: type, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('description'),
      dataIndex: 'description',
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('leaderName'),
      dataIndex: 'leaderName',
      width: 100,
      hideInSearch: true,
    },
    {
      title: t('status'),
      dataIndex: 'status',
      width: 95,
      valueType: 'select',
      fieldProps: {
        options: getStatusOptions(t),
      },
      render: (_, record) => {
        const status = record.status as keyof typeof statusMap;
        const config = statusMap[status] || { text: status, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('sortOrder'),
      dataIndex: 'sortOrder',
      width: 70,
      hideInSearch: true,
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
                setSelectedOrg(record);
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
          <ListTable<OrgUnit>
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

              const response = await fetchListOrgUnits(query);
              const items = (response.items || []) as OrgUnit[];
              // API 已返回树形结构，只需清理空 children
              cleanEmptyChildren(items as any[]);
              setTreeData(items);

              return {
                data: items,
                total: (response as any).total ?? items.length,
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
              <TableExportButton key="export" fetcher={fetchListOrgUnits} columns={columns} filename="org-units" />,
              <Button
                key="expandAll"
                icon={<NodeExpandOutlined />}
                onClick={() => handleExpandAll(treeData)}
              >
                {t('expandAll')}
              </Button>,
              <Button key="collapseAll" icon={<NodeCollapseOutlined />} onClick={handleCollapseAll}>
                {t('collapseAll')}
              </Button>,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedOrg(undefined);
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
            expandable={{
              expandedRowKeys,
              onExpandedRowsChange: (keys) => setExpandedRowKeys(keys),
            }}
          />
        </div>
      </ContentContainer>

      {/* 组织架构编辑/创建 Drawer */}
      <OrgDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedOrg}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedOrg(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default OrgUnitManagement;
