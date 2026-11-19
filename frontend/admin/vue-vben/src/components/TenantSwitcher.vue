<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';

import { Select } from 'ant-design-vue';
import { useUserStore } from '@vben/stores';
import { $t } from '@vben/locales';

import { fetchListTenants, PaginationQuery } from '#/api';
import {
  clearImpersonatedTenant,
  getImpersonatedTenant,
  setImpersonatedTenant,
} from '#/utils/tenant-impersonate';

/**
 * 代租户操作切换器（issue #13）：平台管理员选择生效租户——选中的租户 ID
 * 经请求拦截器走 X-Tenant-Id，后端把数据面与租户闸门切到该租户；"平台全域"
 * 即不携带头，回到现状的全域视图。非平台管理员不渲染。
 */
const userStore = useUserStore();
const isPlatformAdmin = computed(() =>
  (userStore.userInfo?.roles ?? []).includes('platform:admin'),
);

const current = ref<{ id: string; name: string }>(
  getImpersonatedTenant() ?? { id: '', name: '' },
);
const options = ref<{ value: string; label: string }[]>([]);

onMounted(async () => {
  if (!isPlatformAdmin.value) return;
  try {
    const result = await fetchListTenants(
      new PaginationQuery({ paging: { page: 1, pageSize: 200 } }),
    );
    options.value = (result.items ?? []).map((it: any) => ({
      value: it.id != null ? String(it.id) : '',
      label: it.name || it.code || String(it.id),
    }));
  } catch {
    options.value = [];
  }
});

// 持久化值与列表对齐：租户改名/删除后旧缓存自动回落全域
watch(options, () => {
  if (current.value.id && !options.value.some((o) => o.value === current.value.id)) {
    current.value = { id: '', name: '' };
    clearImpersonatedTenant();
  }
});

function handleChange(v: unknown) {
  const hit = options.value.find((o) => o.value === v);
  if (v && hit) {
    current.value = { id: hit.value, name: String(hit.label) };
    setImpersonatedTenant(current.value);
  } else {
    current.value = { id: '', name: '' };
    clearImpersonatedTenant();
  }
}
</script>

<template>
  <div v-if="isPlatformAdmin" class="tenant-switcher mr-2 flex items-center">
    <Select
      :value="current.id || undefined"
      :placeholder="$t('page.role.tenantSwitcherAllTenants')"
      show-search
      allow-clear
      :filter-option="
        (input: string, option: any) =>
          String(option?.label ?? '').toLowerCase().includes(input.toLowerCase())
      "
      style="min-width: 160px"
      @change="handleChange"
    >
      <Select.Option v-for="o in options" :key="o.value" :value="o.value">
        {{ o.label }}
      </Select.Option>
    </Select>
  </div>
</template>
