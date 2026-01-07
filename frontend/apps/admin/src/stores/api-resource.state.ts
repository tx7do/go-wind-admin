import { defineStore } from 'pinia';

import {
  type permissionservicev1_ApiResource as ApiResource,
  createApiResourceServiceClient,
} from '#/generated/api/admin/service/v1';
import { makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useApiResourceStore = defineStore('api-resource', () => {
  const service = createApiResourceServiceClient(requestClientRequestHandler);

  /**
   * 查询API列表
   */
  async function listApiResource(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await service.List({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: orderBy ?? [],
      query: makeQueryString(formValues ?? null),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取API
   */
  async function getApiResource(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建API
   */
  async function createApiResource(values: object) {
    return await service.Create({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新API
   */
  async function updateApiResource(id: number, values: object) {
    return await service.Update({
      id,
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除API
   */
  async function deleteApiResource(id: number) {
    return await service.Delete({ id });
  }

  async function getWalkRouteData() {
    return await service.GetWalkRouteData({});
  }

  async function syncApiResources() {
    return await service.SyncApiResources({});
  }

  function $reset() {}

  return {
    $reset,
    listApiResource,
    getApiResource,
    createApiResource,
    updateApiResource,
    deleteApiResource,
    getWalkRouteData,
    syncApiResources,
  };
});

interface ApiResourceTreeDataNode {
  key: number | string; // 节点唯一标识（父节点用module，子节点用api.id）
  title: string; // 节点显示文本（父节点用module，子节点用api.name）
  children?: ApiResourceTreeDataNode[]; // 子节点（仅父节点有）
  disabled?: boolean;
  apiInfo?: ApiResource;
}

export function convertApiToTree(
  rawApiList: ApiResource[],
): ApiResourceTreeDataNode[] {
  const moduleMap = new Map<string, ApiResource[]>();
  rawApiList.forEach((api) => {
    const moduleName =
      typeof api.moduleDescription === 'string' ? api.moduleDescription : '';
    if (!moduleMap.has(moduleName)) {
      moduleMap.set(moduleName, []);
    }
    moduleMap.get(moduleName)?.push(api);
  });

  return [...moduleMap.entries()].map(([moduleName, apiList]) => ({
    key: `module-${moduleName}`,
    title: moduleName,
    children: apiList.map((api, index) => ({
      key: api.id ?? `api-default-${index}`,
      title: `${api.description}（${api.method}）`,
      apiInfo: api,
    })),
  }));
}
