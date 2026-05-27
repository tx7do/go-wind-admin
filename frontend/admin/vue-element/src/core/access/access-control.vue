<!--
 Access control component for fine-grained access control.
 同时检查角色码和权限码（取并集）
-->
<template>
  <slot v-if="!codes" />
  <slot v-else-if="hasAuth" />
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { useAccess } from "./use-access";

interface Props {
  /**
   * 所需的权限标识（角色码或权限码均可）
   * 不传或传 undefined 表示不限制
   */
  codes?: string[];
}

defineOptions({
  name: "AccessControl",
});

const props = withDefaults(defineProps<Props>(), {});

const { hasAccess } = useAccess();

const hasAuth = computed(() => {
  return hasAccess(props.codes ?? []);
});
</script>
