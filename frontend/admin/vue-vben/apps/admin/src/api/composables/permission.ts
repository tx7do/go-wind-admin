import type {
  permissionservicev1_Permission as Permission,
  permissionservicev1_DeletePermissionRequest,
  permissionservicev1_GetPermissionRequest,
  permissionservicev1_ListPermissionResponse,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 权限点管理
// ==============================

export function useListPermissions(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListPermissionResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPermissions', query],
    queryFn: () => apiClient.permissionService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPermissions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPermissions', params],
    queryFn: () => apiClient.permissionService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetPermission(
  req: permissionservicev1_GetPermissionRequest,
  options?: UseQueryOptions<Permission, Error>,
) {
  return useQuery({
    queryKey: ['getPermission', req],
    queryFn: () => apiClient.permissionService.Get(req),
    ...options,
  });
}

export function useCreatePermission(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.permissionService.Create({ data: { ...values } as Permission }),
    ...options,
  });
}

export function useUpdatePermission(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.permissionService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePermission(
  options?: UseMutationOptions<
    object,
    Error,
    permissionservicev1_DeletePermissionRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.permissionService.Delete(req),
    ...options,
  });
}

export function useSyncPermissions(
  options?: UseMutationOptions<object, Error>,
) {
  return useMutation({
    mutationFn: () => apiClient.permissionService.SyncPermissions({}),
    ...options,
  });
}

// ==============================
// 权限枚举与工具函数
// ==============================

export const roleDataScopeList = computed(() => [
  { label: t('enum.role.dataScope.ALL'), value: 'ALL' },
  { label: t('enum.role.dataScope.UNIT_AND_CHILD'), value: 'UNIT_AND_CHILD' },
  { label: t('enum.role.dataScope.UNIT_ONLY'), value: 'UNIT_ONLY' },
  { label: t('enum.role.dataScope.SELECTED_UNITS'), value: 'SELECTED_UNITS' },
  { label: t('enum.role.dataScope.SELF'), value: 'SELF' },
]);

const DATA_SCOPE_COLOR_MAP: Record<string, string> = {
  ALL: '#F53F3F',
  UNIT_AND_CHILD: '#165DFF',
  UNIT_ONLY: '#FF7D00',
  SELECTED_UNITS: '#722ED1',
  SELF: '#86909C',
  DEFAULT: '#C9CDD4',
};

export function dataScopeToColor(dataScope: any): string {
  return (
    DATA_SCOPE_COLOR_MAP[dataScope as string] ??
    DATA_SCOPE_COLOR_MAP.DEFAULT ??
    '#C9CDD4'
  );
}

export function roleDataScopeToName(dataScope: any) {
  const values = roleDataScopeList.value;
  const matchedItem = values.find((item) => item.value === dataScope);
  return matchedItem ? matchedItem.label : '';
}

interface PermissionTreeDataNode {
  key: number | string;
  title: string;
  children?: PermissionTreeDataNode[];
  permission?: Permission;
}

export function buildPermissionTree(
  permissionGroups: any[],
  permissions: Permission[],
): PermissionTreeDataNode[] {
  const groupNodes: Map<string, PermissionTreeDataNode> = new Map();
  const idToKey: Map<number | string, string> = new Map();
  const nameToKey: Map<string, string> = new Map();
  const defaultKey = 'group-default';
  groupNodes.set(defaultKey, { key: defaultKey, title: '', children: [] });

  function processGroup(g: any, idxPath: string[]): PermissionTreeDataNode {
    const idPart = g.id ?? `idx-${idxPath.join('-')}`;
    const key = `group-${idPart}`;
    const title = typeof g.name === 'string' ? g.name : String(g.id ?? '');
    const node: PermissionTreeDataNode = { key, title, children: [] };
    groupNodes.set(key, node);
    if (g.id !== undefined && g.id !== null) idToKey.set(g.id as any, key);
    if (typeof g.name === 'string' && g.name) nameToKey.set(g.name, key);
    const children = g.children;
    if (Array.isArray(children) && children.length > 0) {
      children.forEach((child: any, idx: number) => {
        const childNode = processGroup(child, [...idxPath, String(idx)]);
        node.children = node.children ?? [];
        node.children.push(childNode);
      });
    }
    return node;
  }

  permissionGroups.forEach((g, idx) => processGroup(g, [String(idx)]));

  permissions.forEach((perm, idx) => {
    let targetKey: string | undefined;
    if (
      perm &&
      (perm as any).groupId !== undefined &&
      (perm as any).groupId !== null
    )
      targetKey =
        idToKey.get((perm as any).groupId) ?? `group-${(perm as any).groupId}`;
    if (!targetKey && typeof (perm as any).groupName === 'string')
      targetKey =
        nameToKey.get((perm as any).groupName) ??
        `group-${(perm as any).groupName}`;
    if (!targetKey || !groupNodes.has(targetKey)) targetKey = defaultKey;
    const childNode: PermissionTreeDataNode = {
      key: perm.id ?? `perm-${idx}`,
      title:
        typeof (perm as any).name === 'string'
          ? `${(perm as any).name} (${(perm as any).code})`
          : '',
      permission: perm,
    };
    const parent = groupNodes.get(targetKey);
    if (parent) {
      parent.children = parent.children ?? [];
      parent.children.push(childNode);
    }
  });

  const result: PermissionTreeDataNode[] = permissionGroups.map((g, idx) => {
    const idPart = g.id ?? `idx-${idx}`;
    const key = `group-${idPart}`;
    return (
      groupNodes.get(key) ?? {
        key,
        title: typeof g.name === 'string' ? g.name : '',
        children: [],
      }
    );
  });
  const defaultNode = groupNodes.get(defaultKey);
  if (defaultNode && (defaultNode.children?.length ?? 0) > 0)
    result.push(defaultNode);
  return result;
}
