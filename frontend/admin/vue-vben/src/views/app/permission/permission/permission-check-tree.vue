<script setup lang="ts">
import type { PropType } from 'vue';

import { ref, watch } from 'vue';

import {
  SquareCheck,
  FoldVertical,
  Square,
  UnfoldVertical,
} from '@vben/icons';
import { $t } from '@vben/locales';

import { Button, Space, Spin, Tree } from 'ant-design-vue';

/**
 * 权限勾选树（创建/更新权限抽屉的「关联菜单 / 关联API」字段）。
 * 对齐 react 基准（PermissionDrawer）：标题在上通栏 + 展开/折叠/全选/取选
 * 四个链接钮工具栏 + 300px 圆角边框盒内滚动 + 默认展开全部，数据为空时
 * 显示虚线占位。树节点为 { key, title, children } 形态。
 */
interface Props {
  emptyText?: string;
  hint?: string;
  loading?: boolean;
  title?: string;
  treeData?: any[];
}

const props = withDefaults(defineProps<Props>(), {
  emptyText: '',
  hint: '',
  loading: false,
  title: '',
  treeData: () => [],
});

const checked = defineModel({ type: Array as PropType<any[]>, default: () => [] });

const expandedKeys = ref<any[]>([]);

function collectAllKeys(nodes: any[]): any[] {
  const keys: any[] = [];
  const walk = (list: any[]) => {
    for (const node of list) {
      keys.push(node.key);
      if (node.children?.length) walk(node.children);
    }
  };
  walk(nodes);
  return keys;
}

watch(
  () => props.treeData,
  (data) => {
    // 默认展开全部（对齐 react 基准）
    expandedKeys.value = collectAllKeys(data ?? []);
  },
  { immediate: true },
);

function expandAll() {
  expandedKeys.value = collectAllKeys(props.treeData);
}

function collapseAll() {
  expandedKeys.value = [];
}

function checkAll() {
  checked.value = collectAllKeys(props.treeData);
}

function uncheckAll() {
  checked.value = [];
}
</script>

<template>
  <div class="w-full">
    <div class="mb-1 text-sm font-medium">{{ title }}</div>
    <div v-if="hint" class="mb-1 text-xs text-muted-foreground">{{ hint }}</div>
    <Spin :spinning="!!loading">
      <template v-if="treeData.length > 0">
        <Space :size="4" class="mb-1">
          <Button
            :style="{ padding: '0 4px', fontSize: '12px' }"
            size="small"
            type="link"
            @click="expandAll"
          >
            <template #icon>
              <UnfoldVertical class="size-3.5" />
            </template>
            {{ $t('page.permission.treeExpandAll') }}
          </Button>
          <Button
            :style="{ padding: '0 4px', fontSize: '12px' }"
            size="small"
            type="link"
            @click="collapseAll"
          >
            <template #icon>
              <FoldVertical class="size-3.5" />
            </template>
            {{ $t('page.permission.treeCollapseAll') }}
          </Button>
          <Button
            :style="{ padding: '0 4px', fontSize: '12px' }"
            size="small"
            type="link"
            @click="checkAll"
          >
            <template #icon>
              <SquareCheck class="size-3.5" />
            </template>
            {{ $t('page.permission.selectAll') }}
          </Button>
          <Button
            :style="{ padding: '0 4px', fontSize: '12px' }"
            size="small"
            type="link"
            @click="uncheckAll"
          >
            <template #icon>
              <Square class="size-3.5" />
            </template>
            {{ $t('page.permission.deselectAll') }}
          </Button>
        </Space>
        <Tree
          v-model:checked-keys="checked"
          v-model:expanded-keys="expandedKeys"
          checkable
          :tree-data="treeData"
          class="max-h-[300px] overflow-auto rounded-md border p-2"
        />
      </template>
      <div
        v-else
        class="rounded-md border border-dashed p-4 text-center text-muted-foreground"
      >
        {{ loading ? '' : emptyText }}
      </div>
    </Spin>
  </div>
</template>
