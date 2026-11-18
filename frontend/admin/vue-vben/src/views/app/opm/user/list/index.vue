<script lang="ts" setup>
import { reactive } from 'vue';

import SplitPane from '#/components/SplitPane.vue';

import OrgList from './org-list.vue';
import UserList from './user-list.vue';

const props = reactive({
  leftCollapsedWidth: 0,
  leftCollapsible: false,
  leftMaxWidth: 40,
  leftMinWidth: 20,
  leftWidth: 25,
  rightWidth: 75,
  resizable: true,
  splitHandle: true,
  splitLine: true,
});
</script>

<template>
  <SplitPane auto-content-height v-bind="props">
    <template #left>
      <OrgList />
    </template>
    <!-- 分割线右侧同距（issue #4）：左栏 m-3 与右侧 ml-3 对称 -->
    <div class="ml-3 h-full">
      <UserList />
    </div>
  </SplitPane>
</template>
