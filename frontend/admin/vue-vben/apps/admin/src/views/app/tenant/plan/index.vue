<script lang="ts" setup>
import { reactive } from 'vue';

import { ColPage } from '@vben/common-ui';

import PlanQuotaList from '#/views/app/tenant/plan/plan-quota-list.vue';
import PlanList from '#/views/app/tenant/plan/plan-list.vue';

const props = reactive({
  leftCollapsedWidth: 0,
  leftCollapsible: false,
  leftMaxWidth: 75,
  leftMinWidth: 25,
  leftWidth: 45,
  rightWidth: 60,
  resizable: true,
  splitHandle: true,
  splitLine: true,
});
</script>

<template>
  <ColPage auto-content-height v-bind="props">
    <template #left>
      <PlanList />
    </template>
    <PlanQuotaList />
  </ColPage>
</template>
