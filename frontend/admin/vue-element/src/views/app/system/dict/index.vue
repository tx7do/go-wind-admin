<script lang="ts" setup>
import { reactive } from 'vue';

import { ColPage } from '@vben/common-ui';

import DictEntryList from '#/views/app/system/dict/dict-entry-list.vue';
import DictTypeList from '#/views/app/system/dict/dict-type-list.vue';

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
      <DictTypeList />
    </template>
    <DictEntryList />
  </ColPage>
</template>
