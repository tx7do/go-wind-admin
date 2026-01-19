import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  type permissionservicev1_Api as Api,
  createApiServiceClient,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useApiStore = defineStore('api', () => {
  const service = createApiServiceClient(requestClientRequestHandler);

  const userStore = useUserStore();

  /**
   * 查询API列表
   */
  async function listApi(
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
      orderBy: makeOrderBy(orderBy),
      query: makeQueryString(formValues, userStore.isTenantUser()),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取API
   */
  async function getApi(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建API
   */
  async function createApi(values: object) {
    return await service.Create({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新API
   */
  async function updateApi(id: number, values: object) {
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
  async function deleteApi(id: number) {
    return await service.Delete({ id });
  }

  async function getWalkRouteData() {
    return await service.GetWalkRouteData({});
  }

  async function syncApis() {
    return await service.SyncApis({});
  }

  function $reset() {}

  return {
    $reset,
    listApi,
    getApi,
    createApi,
    updateApi,
    deleteApi,
    getWalkRouteData,
    syncApis,
  };
});

interface ApiTreeDataNode {
  key: number | string; // 节点唯一标识（父节点用module，子节点用api.id）
  title: string; // 节点显示文本（父节点用module，子节点用api.name）
  children?: ApiTreeDataNode[]; // 子节点（仅父节点有）
  disabled?: boolean;
  apiInfo?: Api;
}

export function convertApiToTree(rawApiList: Api[]): ApiTreeDataNode[] {
  const moduleMap = new Map<string, Api[]>();
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
