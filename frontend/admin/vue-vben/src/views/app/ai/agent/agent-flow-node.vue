<template>
  <div
    class="agent-flow-node bg-card text-foreground"
    :class="{ selected }"
    :style="{ '--node-color': meta.color }"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="flex items-center gap-1.5 font-semibold">
      <span class="node-icon" :style="{ color: meta.color }">{{ meta.icon }}</span>
      <span>{{ data.label }}</span>
    </div>
    <div class="mt-1 text-xs text-muted-foreground">{{ summary }}</div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';

import { Handle, Position } from '@vue-flow/core';

/**
 * Agent 画布自定义节点（与 react 基准 agent-canvas.tsx 的 AgentFlowNode 同构）：
 * data = { nodeType, label, config }，config 摘要与 react configSummary 逐字对齐。
 */
const props = defineProps<{
  data: { nodeType: string; label: string; config?: Record<string, any> };
  selected?: boolean;
}>();

const NODE_META: Record<string, { color: string; icon: string }> = {
  start: { color: '#52c41a', icon: '▶' },
  knowledge: { color: '#722ed1', icon: '📚' },
  llm: { color: '#1677ff', icon: '✦' },
  http: { color: '#4a8dff', icon: '⇋' },
  condition: { color: '#fa8c16', icon: '⑃' },
  end: { color: '#8c8c8c', icon: '■' },
};

const meta = computed(() => NODE_META[props.data?.nodeType] ?? NODE_META.llm!);

const summary = computed(() => configSummary(props.data?.nodeType, props.data?.config));

function configSummary(nodeType: string, config?: Record<string, any>): string {
  if (!config) return '';
  switch (nodeType) {
    case 'start': {
      return (config.inputVars ?? []).map((v: any) => v.name).join(', ');
    }
    case 'knowledge': {
      return config.baseId ? `kb#${config.baseId} topK=${config.topK ?? 4}` : '';
    }
    case 'llm': {
      return config.model || (config.providerId ? `provider#${config.providerId}` : '');
    }
    case 'http': {
      return config.url ? `${config.method ?? 'GET'} ${String(config.url).slice(0, 36)}` : '';
    }
    case 'condition': {
      return config.expression || '';
    }
    case 'end': {
      return config.output ? String(config.output).slice(0, 40) : '';
    }
    default: {
      return '';
    }
  }
}
</script>

<style lang="scss" scoped>
.agent-flow-node {
  min-width: 150px;
  padding: 8px 12px;
  border: 2px solid transparent;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgb(0 0 0 / 12%);

  &.selected {
    border-color: var(--node-color);
  }

  .node-icon {
    color: var(--node-color);
  }
}
</style>
