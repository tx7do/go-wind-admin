import {
  createMenuServiceClient,
  type resourceservicev1_CreateMenuRequest,
  type resourceservicev1_DeleteMenuRequest,
  type resourceservicev1_UpdateMenuRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createMenuServiceClient> | null = null;

export function getMenuService() {
  if (!_instance) {
    _instance = createMenuServiceClient(requestApi);
  }
  return _instance;
}

export async function listMenus(query: PaginationQuery) {
  const params = query.toRawParams();
  return getMenuService().List(params);
}

export async function getMenu(id: number) {
  return getMenuService().Get({ id });
}

export async function createMenu(request: resourceservicev1_CreateMenuRequest) {
  return getMenuService().Create(request);
}

export async function updateMenu(request: resourceservicev1_UpdateMenuRequest) {
  return getMenuService().Update(request);
}

export async function deleteMenu(request: resourceservicev1_DeleteMenuRequest) {
  return getMenuService().Delete(request);
}
