<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useQuery } from "@tanstack/vue-query";
import { ElSelect, ElOption } from "element-plus";
import { useAppUserStore } from "@go-wind/admin-element";
import { fetchListTenants } from "@/api/composables/tenant";
import {
  getImpersonatedTenant,
  setImpersonatedTenant,
  clearImpersonatedTenant,
} from "@/utils/tenant-impersonate";

/**
 * 代租户操作切换器（issue #13）：平台管理员选择生效租户——选中的租户 ID
 * 经请求拦截器走 X-Tenant-Id，后端把数据面与租户闸门切到该租户；"平台全域"
 * 即不携带头，回到现状的全域视图。非平台管理员不渲染。
 */
const userStore = useAppUserStore();
const isPlatformAdmin = computed(() =>
  (userStore.userInfo?.roles ?? []).includes("platform:admin"),
);

const current = ref<{ id: string; name: string }>(
  getImpersonatedTenant() ?? { id: "", name: "" },
);

const { data } = useQuery({
  queryKey: ["listTenants", "impersonate"],
  enabled: isPlatformAdmin,
  queryFn: () => fetchListTenants({ paging: { page: 1, pageSize: 200 } } as any),
});

// 显式给出元素类型：getter 里 `data.value as any` 会把整条链污染成 any，
// 下方 some/find 的回调参数随之失去上下文类型（noImplicitAny 报 TS7006）。
const options = computed<{ value: string; label: string }[]>(() =>
  ((data.value as any)?.items ?? []).map((it: any) => ({
    value: it.id != null ? String(it.id) : "",
    label: String(it.name || it.code || String(it.id)),
  })),
);

// 持久化值与列表对齐：租户改名/删除后旧缓存自动回落全域
watch(
  [options, () => current.value.id],
  () => {
    if (current.value.id && !options.value.some((o) => o.value === current.value.id)) {
      current.value = { id: "", name: "" };
      clearImpersonatedTenant();
    }
  },
  { immediate: true },
);

function handleChange(v: unknown) {
  const hit = options.value.find((o) => o.value === v);
  if (v && hit) {
    current.value = { id: hit.value, name: String(hit.label) };
    setImpersonatedTenant(current.value);
  } else {
    current.value = { id: "", name: "" };
    clearImpersonatedTenant();
  }
}
</script>

<template>
  <div v-if="isPlatformAdmin" class="tenant-switcher">
    <ElSelect
      :model-value="current.id || undefined"
      :placeholder="$t('pages.tenant-switcher.allTenants')"
      filterable
      clearable
      style="min-width: 160px"
      @change="handleChange"
    >
      <ElOption
        v-for="o in options"
        :key="o.value"
        :label="o.label"
        :value="o.value"
      />
    </ElSelect>
  </div>
</template>

<style lang="scss" scoped>
.tenant-switcher {
  display: flex;
  align-items: center;
  margin-right: 4px;
}
</style>
