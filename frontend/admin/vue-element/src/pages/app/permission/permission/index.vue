<template>
  <ProSplitter :left-size="40" :left-min-size="30" :left-max-size="50">
    <template #left>
      <PermissionGroupList />
    </template>
    <template #right>
      <PermissionList />
    </template>
  </ProSplitter>
</template>

<script lang="ts" setup>
import ProSplitter from "@/components/Pro/ProSplitter/index.vue";

import PermissionGroupList from "./permission-group-list.vue";
import PermissionList from "./permission-list.vue";
</script>
