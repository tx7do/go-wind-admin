import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { Tree, Select, Input, Button, App, theme } from 'antd';
import {
  ExpandOutlined,
  ShrinkOutlined,
  CloseCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { fetchListOrgUnits } from '@/api/hooks/org-unit';
import { fetchListTenants } from '@/api/hooks/tenant';
import { useUserStore } from '@/stores/user';

interface OrgListProps {
  currentOrgUnitId: string | undefined; // id 均为 uint64，proto3 JSON 下是字符串
  currentTenantId: string | undefined;
  onTenantSelect: (tenantId: string | undefined) => void;
  onOrgSelect: (orgUnitId: string | undefined) => void;
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
  const [tenantOptions, setTenantOptions] = useState<{ label: string; value: string }[]>([]);

  // 组织树
  const [treeData, setTreeData] = useState<any[]>([]);
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [searchValue, setSearchValue] = useState('');
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // 用 ref 持有 t/message/treeData，避免把它们放进 effect 依赖：
  // - t 来自 useTranslation，切换语言时会变化，会导致组织树被重新拉取并重置展开/选中状态；
  // - treeData 进入搜索 effect 依赖会导致每次重拉都清空用户手动展开的节点。
  const tRef = useRef(t);
  tRef.current = t;
  const messageRef = useRef(message);
  messageRef.current = message;
  const treeDataRef = useRef(treeData);
  treeDataRef.current = treeData;

  // 获取有效租户 ID（uint64 字符串，空串表示未指定）
  const effectiveTenantId = useMemo(() => {
    if (isTenantUser) {
      const userStore = useUserStore.getState();
      return (userStore as any).tenantId ?? '';
    }
    return currentTenantId ?? '';
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
      // 默认折叠全部节点，与 Vue 版一致
    } catch (error: any) {
      messageRef.current.error(error.message || tRef.current('fetchFailed'));
      setTreeData([]);
    }
  }, [effectiveTenantId]);

  useEffect(() => {
    fetchOrgUnits();
  }, [fetchOrgUnits]);

  // 租户切换
  const handleTenantChanged = (value: string | undefined) => {
    onTenantSelect(value);
    // 清除选中
    setSelectedKeys([]);
    onOrgSelect(undefined);
    // 清空搜索词：搜索 effect 现在只依赖 [searchValue]，
    // 若不清空，切换租户后 treeData 变成新 key 集合，但 expandedKeys 仍指向旧树
    // 的 key（在新树里不存在），导致展开态错乱。
    setSearchValue('');
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

  // 搜索自动展开：只依赖 searchValue，treeData 经 ref 读取。
  // 避免把 treeData 放入依赖导致每次重拉（租户/语言切换）都清空用户手动展开的节点。
  useEffect(() => {
    const q = searchValue.trim();
    if (!q) {
      setExpandedKeys([]);
      setAutoExpandParent(false);
      return;
    }
    const parentKeys = new Set<React.Key>();
    collectMatchParents(treeDataRef.current, q.toLowerCase(), parentKeys);
    setExpandedKeys([...parentKeys]);
    setAutoExpandParent(true);
  }, [searchValue]);

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

  const { token } = theme.useToken();

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        overflow: 'hidden',
        borderRadius: token.borderRadiusLG,
        border: `1px solid ${token.colorBorderSecondary}`,
        backgroundColor: token.colorBgContainer,
      }}
    >
      {/* 头部标题栏 */}
      <div
        style={{
          flex: '0 0 auto',
          padding: '10px 12px 8px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        <div style={{
          fontSize: 14,
          fontWeight: 600,
          color: token.colorTextHeading,
          marginBottom: 8,
        }}>
          {t('searchOrg')}
        </div>

        {/* 租户选择器（仅非租户用户可见） */}
        {!isTenantUser && (
          <Select
            showSearch
            allowClear
            style={{ width: '100%', marginBottom: 8 }}
            placeholder={t('tenantIdPlaceholder')}
            value={currentTenantId ?? undefined}
            options={tenantOptions}
            onChange={(value) => handleTenantChanged(value != null ? String(value) : undefined)}
          />
        )}

        {/* 搜索栏 + 工具按钮 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <Input
            allowClear
            prefix={<SearchOutlined style={{ color: token.colorTextQuaternary }} />}
            style={{ flex: 1 }}
            placeholder={t('searchOrg')}
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
          />
          <Button type="text" size="small" icon={<ExpandOutlined />} onClick={handleExpandAll} title={t('expandAll')} />
          <Button type="text" size="small" icon={<ShrinkOutlined />} onClick={handleCollapseAll} title={t('collapseAll')} />
          <Button type="text" size="small" icon={<CloseCircleOutlined />} onClick={handleClearSelect} title={t('clearSelect')} />
        </div>
      </div>

      {/* 组织树 */}
      <div style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: '4px 4px 8px' }}>
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
                <span style={{ color: token.colorPrimary }}>{title.substring(idx, idx + q.length)}</span>
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
