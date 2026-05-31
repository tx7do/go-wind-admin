import { createAdminPortalServiceClient } from '#/api/generated/admin/service/v1';
import { requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createAdminPortalServiceClient> = null;

export function getAdminPortalService() {
  if (!_instance) {
    _instance = createAdminPortalServiceClient(requestApi);
  }
  return _instance;
}

export async function getNavigation() {
  return getAdminPortalService().GetNavigation({});
}

export async function getMyPermissionCode() {
  return getAdminPortalService().GetMyPermissionCode({});
}

export async function getInitialContext() {
  return getAdminPortalService().GetInitialContext({});
}
