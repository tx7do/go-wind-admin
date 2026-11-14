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

import { PaginationQuery } from '@go-wind/admin-react';
import { useProTableScrollY } from '@go-wind/admin-react';
import { fetchListPermissionGroups, useDeletePermissionGroup } from '@/api/hooks/permission-group';
import { getStatusMap, getStatusOptions } from './constants';
import PermissionGroupDrawer from './PermissionGroupDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';

interface PermissionGroupListProps {
  currentGroupId: number | null;
  onGroupSelect: (groupId: number) => void;
}

/**
 * 权限分组列表（树形表格）
 */
const PermissionGroupList: React.FC<PermissionGroupListProps> = ({
  currentGroupId,
  onGroupSelect,
}) => {
  const { t } = useTranslation('permission-group');
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [editingGroup, setEditingGroup] = useState<any>(null);

  // 树形展开控制
  const [treeData, setTreeData] = useState<any[]>([]);
  const [expandedRowKeys, setExpandedRowKeys] = useState<readonly React.Key[]>([]);

  const statusMap = getStatusMap(t);

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

  // 删除 mutation
  const deleteMutation = useDeletePermissionGroup({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listPermissionGroups'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 展开全部
  const handleExpandAll = (data: any[]) => {
    const keys: React.Key[] = [];
    const collectKeys = (items: any[]) => {
      items.forEach((item) => {
        keys.push(item.id as number);
        if (item.children?.length) collectKeys(item.children);
      });
    };
    collectKeys(data);
    setExpandedRowKeys(keys);
  };

  // 折叠全部
  const handleCollapseAll = () => {
    setExpandedRowKeys([]);
  };

  // 列配置
  const columns: ProColumns<any>[] = [
    {
      title: t('name'),
      dataIndex: 'name',
    },
    {
      title: t('module'),
      dataIndex: 'module',
    },
    {
      title: t('status'),
      dataIndex: 'status',
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
                setEditingGroup(record);
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
          headerTitle={false}
          request={async (params) => {
            const query = new PaginationQuery({
              formValues: Object.fromEntries(
                Object.entries(params).filter(
                  ([key]) => !['current', 'pageSize'].includes(key),
                ),
              ),
            });

            const response = await fetchListPermissionGroups(query);
            const items = (response.items || []) as any[];
            // API 已返回树形结构，只需清理空 children
            cleanEmptyChildren(items);
            setTreeData(items);

            // 默认展开全部
            const keys: React.Key[] = [];
            const collectKeys = (nodes: any[]) => {
              nodes.forEach((n) => {
                keys.push(n.id as number);
                if (n.children?.length) collectKeys(n.children);
              });
            };
            collectKeys(items);
            setExpandedRowKeys(keys);

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
            span: 12,
          }}
          pagination={false}
          toolBarRender={() => [
            <TableExportButton key="export" fetcher={fetchListPermissionGroups} columns={columns} filename="permission-groups" />,
            <Button
              key="expandAll"
              icon={<NodeExpandOutlined />}
              size="small"
              onClick={() => handleExpandAll(treeData)}
            >
              {t('expandAll')}
            </Button>,
            <Button
              key="collapseAll"
              icon={<NodeCollapseOutlined />}
              size="small"
              onClick={handleCollapseAll}
            >
              {t('collapseAll')}
            </Button>,
            <Button
              key="create"
              color="primary"
              variant="outlined"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditingGroup(null);
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
          expandable={{
            expandedRowKeys,
            onExpandedRowsChange: (keys) => setExpandedRowKeys(keys),
          }}
          onRow={(record) => ({
            onClick: () => {
              onGroupSelect(record.id);
            },
            className: record.id === currentGroupId ? 'ant-table-row-selected' : undefined,
            style: {
              cursor: 'pointer',
            },
          })}
        />
      </div>

      <PermissionGroupDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={editingGroup}
        onClose={() => {
          setDrawerOpen(false);
          setEditingGroup(null);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default PermissionGroupList;
