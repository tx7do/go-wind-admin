import {
  createApiServiceClient,
  type resourceservicev1_CreateApiRequest,
  type resourceservicev1_DeleteApiRequest,
  type resourceservicev1_GetApiRequest,
  type resourceservicev1_UpdateApiRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createApiServiceClient> | null = null;

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

export async function getApi(request: resourceservicev1_GetApiRequest) {
  return getApiService().Get(request);
}

export async function createApi(request: resourceservicev1_CreateApiRequest) {
  return getApiService().Create(request);
}

export async function updateApi(request: resourceservicev1_UpdateApiRequest) {
  return getApiService().Update(request);
}

export async function deleteApi(request: resourceservicev1_DeleteApiRequest) {
  return getApiService().Delete(request);
}
