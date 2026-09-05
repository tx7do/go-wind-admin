<template>
  <ProSplitter :left-size="40" :left-min-size="30" :left-max-size="50">
    <template #left>
      <DictTypeList />
    </template>
    <template #right>
      <DictEntryList />
    </template>
  </ProSplitter>
</template>

<script lang="ts" setup>
import ProSplitter from "@/components/Pro/ProSplitter/index.vue";

import DictEntryList from "./dict-entry-list.vue";
import DictTypeList from "./dict-type-list.vue";
</script>
