import { create } from 'zustand';
import { persist } from 'zustand/middleware';

/**
 * 代租户操作上下文（issue #13）：平台管理员选择一个生效租户后，
 * 所有管理请求携带 X-Tenant-Id——后端把数据面/租户闸门切到该租户，
 * token 未动，审计仍记真实操作者。空串 = 平台全域视图（默认）。
 */
interface TenantImpersonateState {
  /** 生效租户 ID（字符串形态的 u64）；空串 = 平台全域视图 */
  tenantId: string;
  /** 生效租户名（下拉展示用，随选择缓存） */
  tenantName: string;
  setTenant: (id: string, name: string) => void;
  clearTenant: () => void;
}

export const useTenantImpersonateStore = create<TenantImpersonateState>()(
  persist(
    (set) => ({
      tenantId: '',
      tenantName: '',
      setTenant: (tenantId, tenantName) => set({ tenantId, tenantName }),
      clearTenant: () => set({ tenantId: '', tenantName: '' }),
    }),
    { name: 'rushwind-tenant-impersonate' },
  ),
);
