<script lang="ts" setup>
import { reactive } from 'vue';

import { ColPage } from '@vben/common-ui';

import OrgList from './org-list.vue';
import UserList from './user-list.vue';

const props = reactive({
  leftCollapsedWidth: 0,
  leftCollapsible: false,
  leftMaxWidth: 25,
  leftMinWidth: 75,
  leftWidth: 25,
  rightWidth: 75,
  resizable: true,
  splitHandle: true,
  splitLine: true,
});
</script>

<template>
  <ColPage auto-content-height v-bind="props">
    <template #left>
      <OrgList />
    </template>
    <UserList />
  </ColPage>
</template>
