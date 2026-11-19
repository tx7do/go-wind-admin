/**
 * 代租户操作上下文（issue #13）：平台管理员选择生效租户后，所有管理请求
 * 携带 X-Tenant-Id——后端把数据面与租户闸门切到该租户，token 未动，审计
 * 仍记真实操作者。空 = 平台全域视图（默认）。localStorage 持久化，跨页签
 * 与刷新生效；登出/会话失效由选择器在非平台管理员形态下不再注入而失效。
 */
const KEY = "rushwind-tenant-impersonate";

export interface ImpersonatedTenant {
  id: string;
  name: string;
}

export function getImpersonatedTenant(): ImpersonatedTenant | null {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<ImpersonatedTenant>;
    if (parsed && typeof parsed.id === "string" && parsed.id !== "") {
      return { id: parsed.id, name: typeof parsed.name === "string" ? parsed.name : "" };
    }
  } catch {
    // 损坏的持久化按未配置处理
  }
  return null;
}

export function setImpersonatedTenant(t: ImpersonatedTenant): void {
  localStorage.setItem(KEY, JSON.stringify(t));
}

export function clearImpersonatedTenant(): void {
  localStorage.removeItem(KEY);
}
