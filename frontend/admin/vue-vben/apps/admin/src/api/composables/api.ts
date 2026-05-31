import type {
  permissionservicev1_Api,
  permissionservicev1_DeleteApiRequest,
  permissionservicev1_GetApiRequest,
  permissionservicev1_ListApiResponse,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import {
  createApi,
  deleteApi,
  getApi,
  listApis,
  syncApis,
  updateApi,
} from '#/api/service/api';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

// ==============================
// API 管理
// ==============================

export function useListApis(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListApiResponse, Error>,
) {
  return useQuery({
    queryKey: ['listApis', query],
    queryFn: () => listApis(query),
    ...options,
  });
}

export async function fetchListApis(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listApis', params],
    queryFn: () => listApis(params),
    retry: 0,
  });
}

export function useGetApi(
  req: permissionservicev1_GetApiRequest,
  options?: UseQueryOptions<permissionservicev1_Api, Error>,
) {
  return useQuery({
    queryKey: ['getApi', req],
    queryFn: () => getApi(req),
    ...options,
  });
}

export function useCreateApi(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      createApi({ data: { ...values } as permissionservicev1_Api }),
    ...options,
  });
}

export function useUpdateApi(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateApi({
        id,
        data: {
          ...values,
        },
        updateMask: makeUpdateMask(Object.keys(values ?? [])),
      }),
    ...options,
  });
}

export function useDeleteApi(
  options?: UseMutationOptions<
    object,
    Error,
    permissionservicev1_DeleteApiRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => deleteApi(data),
    ...options,
  });
}

export function useSyncApisApi(options?: UseMutationOptions<object, Error>) {
  return useMutation({
    mutationFn: () => syncApis(),
    ...options,
  });
}

// ==============================
// API 枚举与工具函数
// ==============================

export function convertApiToTree(apis: any[]): any[] {
  const tree: any[] = [];
  for (const api of apis) {
    if (!api) continue;
    if (api.parentId !== 0 && api.parentId !== undefined) continue;
    tree.push(api);
  }
  for (const api of apis) {
    if (!api) continue;
    if (api.parentId === 0 || api.parentId === undefined) continue;
    function findParent(nodes: any[]): boolean {
      for (const node of nodes) {
        if (node.id === api.parentId) {
          if (node.children !== undefined) node.children.push(api);
          return true;
        }
        if (node.children && findParent(node.children)) return true;
      }
      return false;
    }
    if (findParent(tree)) continue;
    tree.push(api);
  }
  return tree;
}
