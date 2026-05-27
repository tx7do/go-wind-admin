/**
 * Global authority directive
 * 用于组件级别的细粒度权限控制
 * @Example v-access="[\"sys:menu:add\"]" 或 v-access="\"sys:menu:add\""
 * 同时检查角色码和权限码（取并集）
 */
import type { App, Directive, DirectiveBinding } from "vue";

import { useAccess } from "./use-access";

function isAccessible(el: Element, binding: DirectiveBinding<string | string[]>) {
  const { hasAccess } = useAccess();

  const value = binding.value;
  if (!value) return;

  const values = Array.isArray(value) ? value : [value];

  if (!hasAccess(values)) {
    el?.remove();
  }
}

const mounted = (el: Element, binding: DirectiveBinding<string | string[]>) => {
  isAccessible(el, binding);
};

const authDirective: Directive = {
  mounted,
};

export function registerAccessDirective(app: App) {
  app.directive("access", authDirective);
}
