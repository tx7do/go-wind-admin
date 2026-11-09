/**
 * 组织树 TreeSelect 数据构建工具
 *
 * 注意：fetchListOrgUnits 返回的 items 已经是**嵌套树形结构**（含 children），
 * 因此这里只需递归映射字段（id→value, name→title）并保留层级，
 * 不能再用 parentId 重建——否则只遍历顶层数组会丢失所有子树。
 */

interface TreeNode {
  value: string; // id 为 uint64，proto3 JSON 下是字符串
  title: string;
  children?: TreeNode[];
}

/**
 * 将已嵌套的组织树映射为 TreeSelect 所需的 {value,title,children} 结构。
 * @param items API 返回的已嵌套组织树
 * @param excludeId 可选，需从候选中排除的节点 id（防止把自身设为父级形成自环）
 */
export function buildOrgTreeSelectData(
  items: Array<{ id?: number | string; name?: string; children?: any[] }>,
  excludeId?: number | string,
): TreeNode[] {
  if (!items || items.length === 0) return [];

  const build = (nodes: Array<{ id?: number | string; name?: string; children?: any[] }>): TreeNode[] => {
    const result: TreeNode[] = [];
    nodes.forEach((item) => {
      if (item.id == null) return;
      // 排除指定节点（防自环）：跳过自身，且不递归其子树
      if (excludeId !== undefined && item.id === excludeId) return;
      const node: TreeNode = {
        value: String(item.id),
        title: item.name || '',
      };
      if (item.children && item.children.length > 0) {
        const children = build(item.children);
        if (children.length > 0) {
          node.children = children;
        }
      }
      result.push(node);
    });
    return result;
  };

  return build(items);
}
