<template>
  <ProSplitter :left-size="40" :left-min-size="30" :left-max-size="50">
    <template #left>
      <PlanList />
    </template>
    <template #right>
      <PlanQuotaList />
    </template>
  </ProSplitter>
</template>

<script lang="ts" setup>
import ProSplitter from "@/components/Pro/ProSplitter/index.vue";

import PlanList from "./plan-list.vue";
import PlanQuotaList from "./plan-quota-list.vue";
</script>
