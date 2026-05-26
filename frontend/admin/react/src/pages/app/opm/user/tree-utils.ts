/**
 * 组织树 TreeSelect 数据构建工具
 */

interface TreeNode {
  value: number;
  label: string;
  title: string;
  children?: TreeNode[];
  parentId?: number;
}

/**
 * 将扁平组织数据构建为 TreeSelect 所需的树形结构
 */
export function buildOrgTreeSelectData(
  items: Array<{ id?: number; name?: string; parentId?: number | null }>,
): TreeNode[] {
  if (!items || items.length === 0) return [];

  const map = new Map<number, TreeNode>();
  const roots: TreeNode[] = [];

  items.forEach((item) => {
    if (item.id == null) return;
    map.set(item.id, {
      value: item.id,
      label: item.name || '',
      title: item.name || '',
      children: [],
      parentId: item.parentId ?? undefined,
    });
  });

  map.forEach((node) => {
    if (node.parentId && map.has(node.parentId)) {
      map.get(node.parentId)!.children!.push(node);
    } else {
      roots.push(node);
    }
  });

  // 清理空 children
  const cleanEmpty = (nodes: TreeNode[]): TreeNode[] =>
    nodes.map((n) => ({
      value: n.value,
      label: n.label,
      title: n.title,
      ...(n.children && n.children.length > 0 ? { children: cleanEmpty(n.children) } : {}),
    }));

  return cleanEmpty(roots);
}
