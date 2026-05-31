import {
  createMenuServiceClient,
  type permissionservicev1_CreateMenuRequest,
  type permissionservicev1_DeleteMenuRequest,
  type permissionservicev1_GetMenuRequest,
  type permissionservicev1_UpdateMenuRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createMenuServiceClient> = null;

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

export async function getMenu(req: permissionservicev1_GetMenuRequest) {
  return getMenuService().Get(req);
}

export async function createMenu(
  request: permissionservicev1_CreateMenuRequest,
) {
  return getMenuService().Create(request);
}

export async function updateMenu(
  request: permissionservicev1_UpdateMenuRequest,
) {
  return getMenuService().Update(request);
}

export async function deleteMenu(
  request: permissionservicev1_DeleteMenuRequest,
) {
  return getMenuService().Delete(request);
}
