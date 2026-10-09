/**
 * 字段级权限工具
 *
 * 隐藏字段集来自 GetMyPermissionCode 响应的 hiddenFields（"资源.字段" 串数组，
 * 如 "User.email"），由后端按角色配置在登录期聚合。前端用它驱动列/字段显隐；
 * 真正的数据裁剪在后端响应侧完成，这里只管 UI 呈现。
 */

/** 从 "资源.字段" 串数组解析出指定资源的隐藏字段名集合 */
export function parseResourceHiddenFields(
  entries: string[] | undefined | null,
  resource: string,
): Set<string> {
  const result = new Set<string>();
  for (const entry of entries ?? []) {
    const dotIndex = entry.indexOf('.');
    if (dotIndex <= 0) continue;
    if (entry.slice(0, dotIndex) !== resource) continue;
    const field = entry.slice(dotIndex + 1);
    if (field) result.add(field);
  }
  return result;
}

/** 判断指定资源的某字段是否被隐藏 */
export function isFieldHidden(
  entries: string[] | undefined | null,
  resource: string,
  field: string,
): boolean {
  return parseResourceHiddenFields(entries, resource).has(field);
}
