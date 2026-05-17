<script lang="ts" setup>
import { reactive } from 'vue';

import { ColPage } from '@vben/common-ui';

import PermissionGroupList from '#/views/app/permission/permission/permission-group-list.vue';
import PermissionList from '#/views/app/permission/permission/permission-list.vue';

const props = reactive({
  leftCollapsedWidth: 0,
  leftCollapsible: false,
  leftMaxWidth: 50,
  leftMinWidth: 40,
  leftWidth: 40,
  rightWidth: 60,
  resizable: true,
  splitHandle: true,
  splitLine: true,
});
</script>

<template>
  <ColPage auto-content-height v-bind="props">
    <template #left>
      <PermissionGroupList />
    </template>
    <PermissionList />
  </ColPage>
</template>
