<script lang="ts" setup>
import { reactive } from 'vue';

import SplitPane from '#/components/SplitPane.vue';

import PlanQuotaList from '#/views/app/tenant/plan/plan-quota-list.vue';
import PlanList from '#/views/app/tenant/plan/plan-list.vue';

const props = reactive({
  leftCollapsedWidth: 0,
  leftCollapsible: false,
  leftMaxWidth: 75,
  leftMinWidth: 25,
  leftWidth: 45,
  rightWidth: 55,
  resizable: true,
  splitHandle: true,
  splitLine: true,
});
</script>

<template>
  <SplitPane auto-content-height v-bind="props">
    <template #left>
      <PlanList />
    </template>
    <PlanQuotaList />
  </SplitPane>
</template>
