import {
  createApiServiceClient,
  type permissionservicev1_CreateApiRequest,
  type permissionservicev1_DeleteApiRequest,
  type permissionservicev1_GetApiRequest,
  type permissionservicev1_UpdateApiRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createApiServiceClient> = null;

export function getApiService() {
  if (!_instance) {
    _instance = createApiServiceClient(requestApi);
  }
  return _instance;
}

export async function listApis(query: PaginationQuery) {
  const params = query.toRawParams();
  return getApiService().List(params);
}

export async function getApi(request: permissionservicev1_GetApiRequest) {
  return getApiService().Get(request);
}

export async function createApi(request: permissionservicev1_CreateApiRequest) {
  return getApiService().Create(request);
}

export async function updateApi(request: permissionservicev1_UpdateApiRequest) {
  return getApiService().Update(request);
}

export async function deleteApi(request: permissionservicev1_DeleteApiRequest) {
  return getApiService().Delete(request);
}

export async function syncApis() {
  return getApiService().SyncApis({});
}
