import type {
  permissionservicev1_PermissionGroup as PermissionGroup,
  permissionservicev1_DeletePermissionGroupRequest,
  permissionservicev1_GetPermissionGroupRequest,
  permissionservicev1_ListPermissionGroupResponse,
} from '#/api/generated/admin/service/v1';

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
// 权限分组管理
// ==============================

export function useListPermissionGroups(
  query: PaginationQuery,
  options?: UseQueryOptions<
    permissionservicev1_ListPermissionGroupResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listPermissionGroups', query],
    queryFn: () => apiClient.permissionGroupService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPermissionGroups(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPermissionGroups', params],
    queryFn: () => apiClient.permissionGroupService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPermissionGroup(
  req: permissionservicev1_GetPermissionGroupRequest,
  options?: UseQueryOptions<PermissionGroup, Error>,
) {
  return useQuery({
    queryKey: ['getPermissionGroup', req],
    queryFn: () => apiClient.permissionGroupService.Get(req),
    ...options,
  });
}

export function useCreatePermissionGroup(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.permissionGroupService.Create({
        data: { ...values } as PermissionGroup,
      }),
    ...options,
  });
}

export function useUpdatePermissionGroup(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.permissionGroupService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePermissionGroup(
  options?: UseMutationOptions<
    object,
    Error,
    permissionservicev1_DeletePermissionGroupRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.permissionGroupService.Delete(req),
    ...options,
  });
}

// ==============================
// 权限分组枚举与工具函数
// ==============================

export function travelPermissionGroupChild(
  nodes: PermissionGroup[] | undefined,
  parent: PermissionGroup,
): boolean {
  if (nodes === undefined) return false;
  if (parent.parentId === 0 || parent.parentId === undefined) {
    if (parent?.name) parent.name = t(parent?.name ?? '');
    nodes.push(parent);
    return true;
  }
  for (const node of nodes) {
    if (node === undefined) continue;
    if (node.id === parent.parentId) {
      if (parent?.name) parent.name = t(parent?.name ?? '');
      if (node.children !== undefined) node.children.push(parent);
      return true;
    }
    if (travelPermissionGroupChild(node.children, parent)) return true;
  }
  return false;
}

export function buildPermissionGroupTree(
  groups: PermissionGroup[],
): PermissionGroup[] {
  const tree: PermissionGroup[] = [];
  for (const group of groups) {
    if (!group) continue;
    if (group.parentId !== 0 && group.parentId !== undefined) continue;
    if (group?.name) group.name = t(group?.name ?? '');
    tree.push(group);
  }
  for (const group of groups) {
    if (!group) continue;
    if (group.parentId === 0 || group.parentId === undefined) continue;
    if (travelPermissionGroupChild(tree, group)) continue;
    if (group?.name) group.name = t(group?.name ?? '');
    tree.push(group);
  }
  return tree;
}
