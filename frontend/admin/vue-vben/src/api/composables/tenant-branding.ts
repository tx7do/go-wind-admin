import { apiClient } from '#/api/client';

/**
 * 租户白标（登录前）：按租户编号取名称/Logo 等展示信息。
 * 未命中（found=false）与未设置同等对待：保持默认品牌，不区分提示（防枚举）。
 */
export async function fetchTenantBranding(code: string): Promise<{
  found: boolean;
  name: string;
  logoUrl: string;
}> {
  const resp = await apiClient.authenticationService.GetTenantBranding({
    code,
  });
  return {
    found: !!resp.found,
    name: resp.name ?? '',
    logoUrl: resp.logoUrl ?? '',
  };
}
