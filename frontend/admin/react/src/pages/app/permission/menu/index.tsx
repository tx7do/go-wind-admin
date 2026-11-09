import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Popconfirm, Tag, App, Space } from 'antd';
import {
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
  NodeExpandOutlined,
  NodeCollapseOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { permissionservicev1_Menu as Menu } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchListMenus, useDeleteMenu, useSyncMenus, buildSyncMenusRequest } from '@/api/hooks/menu';
import { businessRoutes } from '@/router/business-routes';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getRandomColor } from '@/utils/color';
import {
  getMenuTypeMap,
  getStatusMap,
  MENU_TYPE_STATUS,
  STATUS_TYPE_STATUS,
  normalizeAuthority,
} from './constants';
import MenuDrawer from './components/MenuDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 菜单标题关键词的前端过滤。
 * 标题没法在服务端搜：库里 meta.title 存的是 i18n key 或源语言文本，用户看到的
 * 译文只有前端知道；且后端 go-crud 对 meta.* JSONB 路径的过滤条件值绑定有缺陷
 * （值被丢成空串，CONTAINS 恒真/EQ 恒假）。因此按「所见即所搜」在前端匹配：
 * 译文 + 库里原文 + 路由名三路 contains；命中行连同祖先链一起保留，
 * 才能还原完整树形。
 */
function filterMenusByTitle(
  items: Menu[],
  keyword: string | undefined,
  translate: (title: string) => string,
): Menu[] {
  const kw = (keyword ?? '').trim().toLowerCase();
  if (!kw) return items;
  const byId = new Map<string, Menu>(items.map((item) => [item.id ?? '', item]));
  const keep = new Set<string>();
  for (const item of items) {
    const title = (item.meta?.title as string) ?? '';
    const haystack = [translate(title), title, item.name ?? ''].join('\n').toLowerCase();
    if (!haystack.includes(kw)) continue;
    keep.add(item.id ?? '');
    let parentId = item.parentId;
    while (parentId != null && !keep.has(parentId)) {
      keep.add(parentId);
      parentId = byId.get(parentId)?.parentId;
    }
  }
  return items.filter((item) => keep.has(item.id ?? ''));
}

/**
 * 菜单管理页面
 *
 * 树形表格展示菜单层级结构，支持搜索、CRUD 操作
 */
const MenuManagement = () => {
  const { t } = useTranslation('menu');
  // routes 命名空间：后端菜单 meta.title 用 vue 端种子键位（menu.* / page.*），
  // 翻译兜底走 routes.json 的 menu/page 段（与侧边栏 translateLabel 同一套）
  const { t: tRoutes } = useTranslation('routes');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedMenu, setSelectedMenu] = useState<Menu | undefined>();

  // 树数据
  const [treeData, setTreeData] = useState<Menu[]>([]);

  const menuTypeMap = getMenuTypeMap(t);
  const statusMap = getStatusMap(t);

  /**
   * 翻译菜单标题：i18n 直查 → routes 命名空间兜底 → 原文兜底。
   * 与侧边栏 translateLabel 同语义，让菜单管理页展示的也是用户看到的译文。
   */
  const translateMenuTitle = (label: string | undefined): string => {
    if (!label) return '';
    const commonHit = t(label);
    if (commonHit !== label) return commonHit;
    return tRoutes(label, { defaultValue: label });
  };

  // 删除操作
  const deleteMutation = useDeleteMenu({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listMenus'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 同步菜单：将本地业务路由全量推送到后端（后端会清空重建，角色授权需重做）
  const [syncing, setSyncing] = useState(false);
  const syncMutation = useSyncMenus({
    onSuccess: () => {
      message.success(t('syncSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listMenus'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('syncFailed'));
    },
  });

  const handleSyncMenus = async () => {
    setSyncing(true);
    try {
      const request = await buildSyncMenusRequest(businessRoutes, t);
      await syncMutation.mutateAsync(request);
    } finally {
      setSyncing(false);
    }
  };

  // 构建菜单树
  const buildMenuTree = (items: Menu[]): Menu[] => {
    const map = new Map<string, Menu & { children?: Menu[] }>();
    const roots: (Menu & { children?: Menu[] })[] = [];

    // 先创建所有节点的映射
    items.forEach((item) => {
      map.set(item.id ?? '', { ...item, children: [] });
    });

    // 构建树
    map.forEach((node) => {
      if (node.parentId && map.has(node.parentId)) {
        const parent = map.get(node.parentId)!;
        if (!parent.children) parent.children = [];
        parent.children.push(node);
      } else {
        roots.push(node);
      }
    });

    // 清理叶子节点的空 children，避免 ProTable 显示无效展开箭头
    const cleanEmpty = (nodes: (Menu & { children?: Menu[] })[]) => {
      nodes.forEach((n) => {
        if (n.children && n.children.length === 0) {
          delete n.children;
        } else if (n.children) {
          cleanEmpty(n.children);
        }
      });
    };
    cleanEmpty(roots);

    return roots;
  };

  // 展开全部/折叠全部
  const [expandedRowKeys, setExpandedRowKeys] = useState<readonly React.Key[]>([]);

  const handleExpandAll = (data: Menu[]) => {
    const keys: React.Key[] = [];
    const collectKeys = (items: Menu[]) => {
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
  const columns: ProColumns<Menu>[] = [
    {
      // 搜索专用列：标题列的 dataIndex 是 meta.title（ProTable 会把值嵌进
      // params.meta.title），独立出 menuName 扁平键让 request 能直接取到
      title: t('menuName'),
      dataIndex: 'menuName',
      hideInTable: true,
    },
    {
      title: t('menuName'),
      dataIndex: 'meta.title',
      width: 220,
      fixed: 'left',
      hideInSearch: true,
      render: (_, record) => {
        const meta = (record as any).meta || {};
        // 译文展示（对齐 vben/ele 的 $t）：库里的 key / 源语言文本翻成当前语言；
        // meta 缺失（如直接 API 创建的菜单）时回退展示 name，避免整列显示 "-"
        const titleText =
          translateMenuTitle(meta.title) || (record as any).name || '-';
        return (
          <Space>
            <span>{titleText}</span>
          </Space>
        );
      },
    },
    {
      title: t('type'),
      dataIndex: 'type',
      width: 95,
      hideInSearch: true,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(MENU_TYPE_STATUS).map(([key, status]) => [
          key,
          { text: menuTypeMap[key as keyof typeof menuTypeMap]?.text || key, status },
        ]),
      ),
      render: (_, record) => {
        const type = record.type as keyof typeof menuTypeMap;
        const config = menuTypeMap[type] || { text: type, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('authority'),
      dataIndex: 'meta.authority',
      hideInSearch: true,
      render: (_, record) => {
        const meta = (record as any).meta || {};
        const authorities = normalizeAuthority(meta.authority);
        if (authorities.length === 0) return '-';
        return (
          <Space wrap size={[4, 4]}>
            {authorities.map((auth) => (
              <Tag
                key={auth}
                style={{
                  backgroundColor: getRandomColor(auth),
                  color: '#333',
                  border: 'none',
                }}
              >
                {auth}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: t('path'),
      dataIndex: 'path',
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('component'),
      dataIndex: 'component',
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('status'),
      dataIndex: 'status',
      width: 95,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(STATUS_TYPE_STATUS).map(([key, status]) => [
          key,
          { text: statusMap[key as keyof typeof statusMap]?.text || key, status },
        ]),
      ),
      render: (_, record) => {
        const status = record.status as keyof typeof statusMap;
        const config = statusMap[status] || { text: status, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('sortOrder'),
      dataIndex: 'meta.order',
      width: 70,
      hideInSearch: true,
      render: (_, record) => {
        const meta = (record as any).meta || {};
        return meta.order ?? '-';
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
                setDrawerMode('edit');
                setSelectedMenu(record);
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
                description: t('deleteConfirmDesc', {
                  name: (record as any).meta?.title || record.path || '',
                }),
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
          <ListTable<Menu>
            actionRef={actionRef}
            columns={columns}
            request={async (params, _sorter, _filter) => {
              // 标题走前端过滤（见 filterMenusByTitle），服务端只收 status；
              // pageSize 拉满确保全量行都在前端过滤范围内
              const query = new PaginationQuery({
                paging: {
                  page: 1,
                  pageSize: 1000,
                },
                formValues: {
                  status: params.status,
                },
              });

              const response = await fetchListMenus(query);
              const items = filterMenusByTitle(
                (response.items || []) as Menu[],
                params.menuName,
                translateMenuTitle,
              );
              const treeData = buildMenuTree(items);
              setTreeData(treeData);

              return {
                data: treeData,
                total: items.length,
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
              <TableExportButton key="export" fetcher={fetchListMenus} columns={columns} filename="menus" />,
              <Button
                key="expandAll"
                icon={<NodeExpandOutlined />}
                onClick={() => handleExpandAll(treeData)}
              >
                {t('expandAll')}
              </Button>,
              <Button
                key="collapseAll"
                icon={<NodeCollapseOutlined />}
                onClick={handleCollapseAll}
              >
                {t('collapseAll')}
              </Button>,
              <Popconfirm
                key="syncMenus"
                title={t('syncConfirmTitle')}
                description={t('syncConfirmDesc')}
                onConfirm={handleSyncMenus}
                okText={t('common:button.ok')}
                cancelText={t('common:button.cancel')}
              >
                <Button color="orange" variant="outlined" icon={<SyncOutlined />} loading={syncing || syncMutation.isPending}>
                  {t('sync')}
                </Button>
              </Popconfirm>,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedMenu(undefined);
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
            expandable={{
              expandedRowKeys,
              onExpandedRowsChange: (keys) => setExpandedRowKeys(keys),
            }}
          />
        </div>
      </ContentContainer>

      {/* 菜单编辑/创建 Drawer */}
      <MenuDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedMenu}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedMenu(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default MenuManagement;
