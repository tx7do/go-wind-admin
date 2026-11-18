<script lang="ts" setup>
import { reactive } from 'vue';

import SplitPane from '#/components/SplitPane.vue';

import DictEntryList from '#/views/app/system/dict/dict-entry-list.vue';
import DictTypeList from '#/views/app/system/dict/dict-type-list.vue';

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
      <DictTypeList />
    </template>
    <DictEntryList />
  </SplitPane>
</template>
