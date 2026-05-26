import { useState, useEffect, useMemo, useCallback } from 'react';
import { Tree, Select, Input, Space, Button, App } from 'antd';
import {
  ExpandOutlined,
  ShrinkOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { fetchListOrgUnits } from '@/api/hooks/org-unit';
import { fetchListTenants } from '@/api/hooks/tenant';
import { useUserStore } from '@/stores/user';

interface OrgListProps {
  currentOrgUnitId: number | undefined;
  currentTenantId: number | undefined;
  onTenantSelect: (tenantId: number | undefined) => void;
  onOrgSelect: (orgUnitId: number | undefined) => void;
}

/**
 * 左侧组织树列表
 * 包含租户选择器（非租户用户可见）、搜索、树操作
 */
const OrgList: React.FC<OrgListProps> = ({
  currentOrgUnitId,
  currentTenantId,
  onTenantSelect,
  onOrgSelect,
}) => {
  const { t } = useTranslation('user');
  const { message } = App.useApp();
  const isTenantUser = useUserStore((s) => s.isTenantUser());

  // 租户下拉
  const [tenantOptions, setTenantOptions] = useState<{ label: string; value: number }[]>([]);
  const [selectedTenantValue, setSelectedTenantValue] = useState<string>('');

  // 组织树
  const [treeData, setTreeData] = useState<any[]>([]);
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [searchValue, setSearchValue] = useState('');
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // 获取有效租户 ID
  const effectiveTenantId = useMemo(() => {
    if (isTenantUser) {
      const userStore = useUserStore.getState();
      return (userStore as any).tenantId ?? 0;
    }
    return currentTenantId ?? 0;
  }, [isTenantUser, currentTenantId]);

  // 加载租户列表（非租户用户）
  useEffect(() => {
    if (!isTenantUser) {
      fetchListTenants(new PaginationQuery({ formValues: { status: 'ON' } }))
        .then((res) => {
          const opts = (res.items || []).map((item: any) => ({
            label: item.name ?? item.code ?? String(item.id ?? ''),
            value: item.id,
          }));
          setTenantOptions(opts);
        })
        .catch(() => setTenantOptions([]));
    }
  }, [isTenantUser]);

  // 加载组织树（API 已返回树形结构，与 PermissionGroupList 一致）
  const fetchOrgUnits = useCallback(async () => {
    try {
      const res = await fetchListOrgUnits(
        new PaginationQuery({
          formValues: {
            ...(effectiveTenantId ? { tenant_id: effectiveTenantId } : {}),
            status: 'ON',
          },
        }),
      );
      const items = (res.items || []) as any[];
      // API 已返回树形结构，只需清理空 children
      cleanEmptyChildren(items);
      const tree = mapToAntTree(items);
      setTreeData(tree);
      // 默认展开全部
      const keys = collectAllKeys(tree);
      setExpandedKeys(keys);
    } catch (error: any) {
      message.error(error.message || t('fetchFailed'));
      setTreeData([]);
    }
  }, [effectiveTenantId, message, t]);

  useEffect(() => {
    fetchOrgUnits();
  }, [fetchOrgUnits]);

  // 租户切换
  const handleTenantChanged = (value: number | undefined) => {
    setSelectedTenantValue(value != null ? String(value) : '');
    onTenantSelect(value);
    // 清除选中
    setSelectedKeys([]);
    onOrgSelect(undefined);
  };

  // 选中组织节点
  const handleSelect = (keys: React.Key[], info: any) => {
    setSelectedKeys(keys);
    if (keys.length > 0) {
      const node = info.node;
      onOrgSelect(node?.id);
    } else {
      onOrgSelect(undefined);
    }
  };

  // 同步 selectedKeys 与 currentOrgUnitId（树数据加载后重建选中状态）
  useEffect(() => {
    if (currentOrgUnitId == null) {
      setSelectedKeys([]);
      return;
    }
    if (treeData.length === 0) return;
    const findKeyByOrgId = (nodes: any[]): string | null => {
      for (const node of nodes) {
        if (node.id === currentOrgUnitId) return node.key;
        if (node.children?.length) {
          const found = findKeyByOrgId(node.children);
          if (found) return found;
        }
      }
      return null;
    };
    const key = findKeyByOrgId(treeData);
    if (key) {
      setSelectedKeys([key]);
    }
  }, [currentOrgUnitId, treeData]);

  // 搜索自动展开
  useEffect(() => {
    const q = searchValue.trim();
    if (!q) {
      setExpandedKeys([]);
      setAutoExpandParent(false);
      return;
    }
    const parentKeys = new Set<React.Key>();
    collectMatchParents(treeData, q.toLowerCase(), parentKeys);
    setExpandedKeys([...parentKeys]);
    setAutoExpandParent(true);
  }, [searchValue, treeData]);

  // 展开全部
  const handleExpandAll = () => {
    setExpandedKeys(collectAllKeys(treeData));
    setAutoExpandParent(true);
  };

  // 折叠全部
  const handleCollapseAll = () => {
    setExpandedKeys([]);
    setAutoExpandParent(false);
  };

  // 取消选中
  const handleClearSelect = () => {
    setSelectedKeys([]);
    onOrgSelect(undefined);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', padding: '0 4px', overflow: 'hidden' }}>
      {/* 租户选择器（仅非租户用户可见） */}
      {!isTenantUser && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 8px', flex: '0 0 auto' }}>
          <span style={{ fontWeight: 500, whiteSpace: 'nowrap' }}>{t('tenantId')}</span>
          <Select
            showSearch
            allowClear
            style={{ flex: 1 }}
            placeholder={t('tenantIdPlaceholder')}
            value={selectedTenantValue || undefined}
            options={tenantOptions}
            onChange={(value) => handleTenantChanged(value != null ? Number(value) : undefined)}
          />
        </div>
      )}

      {/* 搜索 + 工具栏 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 8px', flex: '0 0 auto' }}>
        <span style={{ fontWeight: 500, whiteSpace: 'nowrap' }}>{t('searchOrg')}</span>
        <Input
          allowClear
          style={{ flex: 1 }}
          placeholder={t('searchOrg')}
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
        />
        <Space size={2}>
          <Button type="text" size="small" icon={<ExpandOutlined />} onClick={handleExpandAll} title={t('expandAll')} />
          <Button type="text" size="small" icon={<ShrinkOutlined />} onClick={handleCollapseAll} title={t('collapseAll')} />
          <Button type="text" size="small" icon={<CloseCircleOutlined />} onClick={handleClearSelect} title={t('clearSelect')} />
        </Space>
      </div>

      {/* 组织树 */}
      <div style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
        <Tree
          blockNode
          expandedKeys={expandedKeys}
          autoExpandParent={autoExpandParent}
          selectedKeys={selectedKeys}
          treeData={treeData}
          onExpand={(keys) => {
            setExpandedKeys(keys);
            setAutoExpandParent(false);
          }}
          onSelect={handleSelect}
          titleRender={(nodeData: any) => {
            const title = nodeData.title as string;
            const q = searchValue.trim();
            if (!q || !title) return title;
            const idx = title.toLowerCase().indexOf(q.toLowerCase());
            if (idx === -1) return title;
            return (
              <span>
                {title.substring(0, idx)}
                <span style={{ color: '#f50' }}>{title.substring(idx, idx + q.length)}</span>
                {title.substring(idx + q.length)}
              </span>
            );
          }}
        />
      </div>
    </div>
  );
};

export default OrgList;

// ========== 工具函数 ==========

/**
 * 将 API 返回的树形 OrgUnit 数据映射为 antd Tree 所需的格式
 * API 已返回 children 树形结构，只需添加 key/title 并递归处理
 */
function mapToAntTree(items: any[]): any[] {
  if (!items || items.length === 0) return [];
  return items.map((item) => {
    const node: any = {
      key: item.id,
      id: item.id,
      title: item.name || '',
    };
    if (item.children && item.children.length > 0) {
      node.children = mapToAntTree(item.children);
    }
    return node;
  });
}

/**
 * 清理 API 返回树形数据中的空 children 数组
 */
function cleanEmptyChildren(nodes: any[]): void {
  nodes.forEach((n) => {
    if (n.children) {
      if (n.children.length === 0) {
        delete n.children;
      } else {
        cleanEmptyChildren(n.children);
      }
    }
  });
}

/**
 * 收集树所有节点 key
 */
function collectAllKeys(tree: any[]): React.Key[] {
  const keys: React.Key[] = [];
  const walk = (nodes: any[]) => {
    nodes.forEach((n) => {
      keys.push(n.key);
      if (n.children?.length) walk(n.children);
    });
  };
  walk(tree);
  return keys;
}

/**
 * 收集匹配搜索文字的父节点 key
 */
function collectMatchParents(tree: any[], query: string, parentKeys: Set<React.Key>, parents: React.Key[] = []) {
  tree.forEach((node) => {
    const title = (node.title || '').toLowerCase();
    if (title.includes(query)) {
      parents.forEach((p) => parentKeys.add(p));
    }
    if (node.children?.length) {
      collectMatchParents(node.children, query, parentKeys, [...parents, node.key]);
    }
  });
}
