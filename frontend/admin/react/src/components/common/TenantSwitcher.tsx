import { useEffect, useMemo, useState } from 'react';
import { Select } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { PaginationQuery, useUserStore } from '@go-wind/admin-react';
import { fetchListTenants } from '@/api/hooks/tenant';
import { useTenantImpersonateStore } from '@/stores/tenant-impersonate';

/**
 * 代租户操作切换器（issue #13）：平台管理员在头部选择生效租户——选中的
 * 租户 ID 经请求拦截器走 X-Tenant-Id，后端把数据面与租户闸门切到该租户；
 * "平台全域"即不携带头，回到现状的全域视图。非平台管理员不渲染。
 */
export const TenantSwitcher = () => {
  const { t } = useTranslation('tenant-switcher');
  const isPlatformAdmin =
    useUserStore((s) => s.userRoles?.includes('platform:admin')) ?? false;
  const { tenantId, tenantName, setTenant, clearTenant } = useTenantImpersonateStore();
  const [loaded, setLoaded] = useState(false);

  const { data } = useQuery({
    queryKey: ['listTenants', 'impersonate'],
    enabled: isPlatformAdmin && loaded,
    queryFn: () =>
      fetchListTenants(new PaginationQuery({ paging: { page: 1, pageSize: 200 } })),
  });

  useEffect(() => {
    if (isPlatformAdmin) {
      setLoaded(true);
    }
  }, [isPlatformAdmin]);

  const options = useMemo(
    () =>
      (data?.items ?? []).map((it: any) => ({
        value: it.id != null ? String(it.id) : '',
        label: it.name || it.code || String(it.id),
      })),
    [data],
  );

  if (!isPlatformAdmin) return null;

  return (
    <Select
      showSearch
      allowClear
      value={tenantId || undefined}
      style={{ minWidth: 160 }}
      placeholder={t('tenantSwitcher.allTenants')}
      options={options}
      filterOption={(input, option) =>
        String(option?.label ?? '').toLowerCase().includes(input.toLowerCase())
      }
      onChange={(value: unknown) => {
        const picked = options.find((o) => o.value === value);
        if (value && picked) {
          setTenant(picked.value, String(picked.label));
        } else {
          clearTenant();
        }
      }}
      onDropdownVisibleChange={(open: boolean) => {
        // 缓存名与列表可能脱节（租户改名/删除）：下拉展开时按 id 对齐一次
        if (open && tenantId && !options.some((o) => o.value === tenantId)) {
          clearTenant();
        } else if (open && tenantId) {
          const hit = options.find((o) => o.value === tenantId);
          if (hit && hit.label !== tenantName) {
            setTenant(tenantId, String(hit.label));
          }
        }
      }}
    />
  );
};
