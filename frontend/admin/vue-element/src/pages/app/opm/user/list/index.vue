<template>
  <!-- 组织树较窄，分栏在更窄的视口仍可用，堆叠阈值相应降低 -->
  <ProSplitter :left-size="25" :left-min-size="20" :left-max-size="25" :right-size="75" :stack-below="1200">
    <template #left>
      <OrgList />
    </template>
    <template #right>
      <UserList />
    </template>
  </ProSplitter>
</template>

<script lang="ts" setup>
import ProSplitter from "@/components/Pro/ProSplitter/index.vue";

import OrgList from "./org-list.vue";
import UserList from "./user-list.vue";
</script>
