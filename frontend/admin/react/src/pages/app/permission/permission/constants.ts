/**
 * 权限点/权限分组模块通用常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// 通用 fallback：在非组件环境下的默认翻译
const defaultT = (key: string) => key;

// ========== 状态映射 ==========

export const STATUS_COLORS: Record<string, string> = {
  ON: 'success',
  OFF: 'error',
};

export function getStatusMap(t: TFn) {
  return {
    ON: { text: t('statusMap.ON'), color: STATUS_COLORS.ON },
    OFF: { text: t('statusMap.OFF'), color: STATUS_COLORS.OFF },
  };
}

export function getStatusOptions(t: TFn) {
  return [
    { label: t('statusMap.ON'), value: 'ON' },
    { label: t('statusMap.OFF'), value: 'OFF' },
  ];
}

// ========== TreeSelect 数据构建（用于 Drawer 的父级选择） ==========

interface TreeSelectNode {
  value: string;
  title: string;
  children?: TreeSelectNode[];
}

/**
 * 将扁平数据构建为 TreeSelect 所需的树形结构
 */
export function buildTreeSelectData(
  items: Array<{ id?: number | string; name?: string; parentId?: number | string | null }>,
  excludeId?: number | string,
): TreeSelectNode[] {
  if (!items || items.length === 0) return [];

  const map = new Map<string, TreeSelectNode & { parentId?: string | null }>();

  items.forEach((item) => {
    if (item.id === undefined || item.id === null) return;
    if (excludeId !== undefined && item.id === excludeId) return;
    map.set(String(item.id), { value: String(item.id), title: item.name || '', children: [], parentId: item.parentId == null ? undefined : String(item.parentId) });
  });

  const tree: TreeSelectNode[] = [];

  map.forEach((node) => {
    if (node.parentId && map.has(node.parentId)) {
      map.get(node.parentId)!.children!.push(node);
    } else {
      tree.push(node);
    }
  });

  const cleanEmpty = (nodes: TreeSelectNode[]): TreeSelectNode[] =>
    nodes.map((n) => ({
      value: n.value,
      title: n.title,
      ...(n.children && n.children.length > 0 ? { children: cleanEmpty(n.children) } : {}),
    }));

  return cleanEmpty(tree);
}

// ========== ProTable 树形数据构建（用于分组列表） ==========

/**
 * 将扁平权限分组数据构建为 ProTable 需要的树形结构（children 嵌套）
 */
export function buildGroupTreeData(items: any[]): any[] {
  if (!items || items.length === 0) return [];

  const map = new Map<string, any>();
  const roots: any[] = [];

  items.forEach((item: any) => {
    if (item.id == null) return;
    map.set(String(item.id), { ...item, children: [] });
  });

  map.forEach((node) => {
    const parentId = node.parentId ? String(node.parentId) : null;
    if (parentId && map.has(parentId)) {
      map.get(parentId)!.children.push(node);
    } else {
      roots.push(node);
    }
  });

  // 清理空的 children 数组
  const cleanEmpty = (nodes: any[]) => {
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
}

// ========== Tree 组件数据构建（用于 Drawer 的勾选树） ==========

interface TreeNode {
  key: string;
  title: string;
  children?: TreeNode[];
}

/**
 * 构建菜单树形数据（用于 Tree 组件勾选）
 */
export function buildMenuTree(
  items?: Array<{
    id?: number | string;
    name?: string;
    parentId?: number | string | null;
    meta?: { title?: string };
  }> | null,
): TreeNode[] {
  if (!items || items.length === 0) return [];

  const map = new Map<string, TreeNode & { parentId?: string | null }>();

  items.forEach((item) => {
    if (item.id === undefined || item.id === null) return;
    map.set(String(item.id), {
      key: String(item.id),
      title: item.meta?.title || item.name || '',
      children: [],
      parentId: item.parentId == null ? undefined : String(item.parentId),
    });
  });

  const tree: TreeNode[] = [];

  map.forEach((node) => {
    if (node.parentId && map.has(node.parentId)) {
      map.get(node.parentId)!.children!.push(node);
    } else {
      tree.push(node);
    }
  });

  const cleanEmpty = (nodes: TreeNode[]): TreeNode[] =>
    nodes.map((n) => ({
      key: n.key,
      title: n.title,
      ...(n.children && n.children.length > 0 ? { children: cleanEmpty(n.children) } : {}),
    }));

  return cleanEmpty(tree);
}

/**
 * 构建 API 树形数据（按模块分组，用于 Tree 组件勾选）
 */
export function buildApiTree(
  items?: Array<{
    id?: number | string;
    method?: string;
    path?: string;
    description?: string;
    name?: string;
    module?: string;
  }> | null,
  t?: TFn,
): TreeNode[] {
  if (!items || items.length === 0) return [];

  const moduleMap = new Map<string, any[]>();

  items.forEach((item) => {
    const mod = item.module || (t || defaultT)('permission-group:moduleOther');
    if (!moduleMap.has(mod)) {
      moduleMap.set(mod, []);
    }
    moduleMap.get(mod)!.push(item);
  });

  return Array.from(moduleMap.entries()).map(([mod, apis]) => ({
    key: `module_${mod}`,
    title: mod,
    selectable: false,
    children: apis.map((api) => ({
      key: String(api.id),
      title: `${api.method || ''} ${api.path || ''} - ${api.description || api.name}`,
    })),
  }));
}

// ========== 工具函数 ==========

/**
 * 从勾选值中提取所有实体 ID 字符串（过滤掉模块分组等合成 key）
 * 注：ID 已迁移为 uint64 → proto3 JSON 字符串，勾选 key 全部是字符串，
 * 以 'module_' 前缀识别并剥离 API 树的分组节点 key。
 */
export function filterNumbers(values: any[]): string[] {
  if (!Array.isArray(values)) return [];
  return values
    .flat(Infinity)
    .filter((v) => typeof v === 'string' && v !== '' && !v.startsWith('module_'));
}

/**
 * 递归收集树中所有「叶子节点」（无 children）的 key。
 * 用于提交勾选值时剥离父节点 key：
 * 菜单树默认开启父子联动，勾选某父菜单下全部子菜单时父菜单会被自动勾选，
 * 其 key（父菜单 ID）会混入 checkedKeys。
 * 这里用叶子集合对 checkedKeys 求交集，只保留真正的叶子菜单 ID。
 */
export function extractLeafIds(checkedKeys: any[], treeData: any[]): string[] {
  if (!Array.isArray(checkedKeys) || !Array.isArray(treeData)) return [];
  const leafIds = new Set<string>();
  const collect = (nodes: any[]) => {
    for (const node of nodes) {
      if (node.children && node.children.length > 0) {
        collect(node.children);
      } else if (typeof node.key === 'string' && node.key !== '') {
        leafIds.add(node.key);
      }
    }
  };
  collect(treeData);
  return checkedKeys
    .flat(Infinity)
    .filter((v): v is string => typeof v === 'string' && v !== '' && leafIds.has(v));
}
