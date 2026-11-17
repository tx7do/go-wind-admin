<script lang="ts" setup>
import type { TreeProps } from 'ant-design-vue';
import type { DefaultOptionType } from 'ant-design-vue/es/select';

import { computed, onMounted, ref, watch } from 'vue';

import { LucideEllipsisVertical } from '@vben/icons';
// 包内 iconify 集未导出 Folder —— 宿主自建（同一 createIconifyIcon 工厂，
// 同一 'lucide:folder' 资源名）。
import { createIconifyIcon } from '@vben/icons';

const LucideFolder = createIconifyIcon('lucide:folder');
import { $t } from '@vben/locales';
import { VbenDropdownMenu, VbenIconButton } from '@vben-core/shadcn-ui';
import { mapTree } from '@vben-core/shared/utils';

import { Select } from 'ant-design-vue';

import { type identityservicev1_OrgUnit as OrgUnit } from '#/api';
import { TreeActionEnum } from '#/constants/tree';

import { useUserViewStore } from './user-view.state';

const userViewStore = useUserViewStore();

// 复用面（issue #5）：组织架构页以非 store 形态接入——
//   v-model:selected-keys  回显选中（键 = `${parentId}-${id}`，与树键同构）
//   @org-select            透传选中节点（null = 取消选中）
// 用户页零 props 使用时行为与旧版完全一致（仍走 user-view.store）。
const props = defineProps<{
  selectedKeys?: (number | string)[];
}>();
const emit = defineEmits<{
  (e: 'org-select', node: any): void;
  (e: 'update:selectedKeys', keys: (number | string)[]): void;
}>();
// 双向选中键：外部传入时以外部为准（回显同步），内部选中仍写本地并回抛。
const localSelectedKeys = ref<(number | string)[]>([]);
const effectiveSelectedKeys = computed(
  () => props.selectedKeys ?? localSelectedKeys.value,
);
function commitSelectedKeys(keys: (number | string)[]) {
  localSelectedKeys.value = keys;
  emit('update:selectedKeys', keys);
}

const toolbarList = [
  {
    value: TreeActionEnum.EXPAND_ALL,
    label: $t('ui.tree.expand_all'),
    handler: handleMenuExpandAll,
  },
  {
    value: TreeActionEnum.COLLAPSE_ALL,
    label: $t('ui.tree.collapse_all'),
    handler: handleMenuCollapseAll,
  },
  {
    value: TreeActionEnum.UNSELECT_ALL,
    label: $t('ui.tree.unselect_all'),
    handler: handleMenuUnselectedAll,
  },
];

const expandedKeys = ref<(number | string)[]>([]);
const selectedValue = ref<string>('');
const searchValue = ref<string>('');
const autoExpandParent = ref<boolean>(true);

const treeData = ref<TreeProps['treeData']>([]);
const selectedKeys = ref<(number | string)[]>([]);
// 左栏三段式回显（issue #4）：当前选择 → 搜索框 → 机构树
const selectedTitle = ref<string>('');

const tenantOptions = computed<DefaultOptionType[]>(() =>
  (userViewStore.tenantList.items ?? []).map((t: any) => ({
    label: t.name ?? t.code ?? String(t.id ?? ''),
    value: String(t.id ?? ''),
    raw: t, // 保留原始对象以备需要
  })),
);

const filterOption = (input: string, option: any) => {
  const text = String(option.value ?? option.label ?? '').toLowerCase();
  return text.includes(input.toLowerCase());
};

/**
 * 获取组织单元列表
 */
async function fetchOrgUnits() {
  try {
    const response = await userViewStore.fetchOrgUnitList();

    const newTree = mapTree(response.items ?? [], (node: OrgUnit) => ({
      ...node,
      key: `${node.parentId}-${node.id}`,
      title: node.name,
      isLeaf: !node.children || node.children.length === 0,
    }));
    // @ts-ignore treeData
    treeData.value = newTree ?? [];
  } catch (error) {
    console.error(error);
  }
}

async function fetch() {
  await fetchOrgUnits();
}

/**
 * 展开所有节点
 */
function handleMenuExpandAll(_data: any) {
  const keys: (number | string)[] = [];

  const traverse = (nodes: any[] | undefined) => {
    if (!nodes || nodes.length === 0) return;
    for (const node of nodes) {
      if (node.children && node.children.length > 0) {
        if (node.key !== undefined && node.key !== null) {
          keys.push(node.key);
        } else if (node.id !== undefined || node.parentId !== undefined) {
          keys.push(`${node.parentId ?? ''}-${node.id ?? ''}`);
        }
        traverse(node.children);
      }
    }
  };

  traverse(treeData.value as any[]);
  expandedKeys.value = keys;
  autoExpandParent.value = true;
}

/**
 * 折叠所有节点
 */
function handleMenuCollapseAll(_data: any) {
  expandedKeys.value = [];
  autoExpandParent.value = false;
}

/**
 * 取消选中所有节点
 */
function handleMenuUnselectedAll(_data: any) {
  clearSelection();
}

/**
 * 展开单个节点
 * @param keys
 */
const handleExpandNode = (keys: string[]) => {
  expandedKeys.value = keys;
  autoExpandParent.value = false;
};

/**
 * 选中组织单元
 * @param node
 */
function handleSelectOrgUnit(node: any) {
  // 非 store 接入形态（组织架构页）：只发事件，不触碰用户页 store。
  if (props.selectedKeys !== undefined) {
    emit('org-select', node);
    return;
  }
  userViewStore.setCurrentOrgUnitId(node ? node.id || null : null);
}

/**
 * 选中单个节点
 */
function handleSelectNode(keys: (number | string)[], e?: any) {
  commitSelectedKeys(keys as (number | string)[]);
  selectedKeys.value = keys as (number | string)[];

  const key = keys && keys.length > 0 ? keys[0] : undefined;
  if (key === undefined) {
    handleSelectOrgUnit(null);
    return;
  }

  // 优先使用事件中提供的节点信息
  const nodeFromEvent =
    e?.selectedNodes?.[0] ?? e?.node?.dataRef ?? e?.node?.origin ?? null;
  if (nodeFromEvent) {
    selectedTitle.value = String(
      (nodeFromEvent as any).title ?? (nodeFromEvent as any).name ?? '',
    );
    handleSelectOrgUnit(nodeFromEvent);
    return;
  }

  // 回退：在 treeData 中递归查找匹配 key 的节点
  const findNode = (nodes: any[] | undefined): any => {
    if (!nodes || nodes.length === 0) return null;
    for (const n of nodes) {
      if (
        n.key === key ||
        n.id === key ||
        `${n.parentId ?? ''}-${n.id ?? ''}` === String(key)
      ) {
        return n;
      }
      const found = findNode(n.children);
      if (found) return found;
    }
    return null;
  };

  const node = findNode(treeData.value as any[]);
  selectedTitle.value = node ? String(node.title ?? node.name ?? '') : '';
  handleSelectOrgUnit(node ?? key);
}

function handleTenantChanged(value: any) {
  userViewStore.setCurrentTenantId(value);
  fetchOrgUnits();
}

function clearSelection() {
  commitSelectedKeys([]);
  selectedKeys.value = [];
  selectedTitle.value = '';
  handleSelectOrgUnit(null);
}

watch(searchValue, (val) => {
  const q = String(val ?? '').trim();
  if (!q) {
    // 搜索为空时清除展开（按需可改为保留原 expandedKeys）
    expandedKeys.value = [];
    autoExpandParent.value = false;
    return;
  }

  const parentKeys = new Set<number | string>();
  const collect = (
    nodes: any[] | undefined,
    parents: (number | string)[] = [],
  ) => {
    if (!nodes || nodes.length === 0) return;
    for (const node of nodes) {
      const title = String(node.title ?? '');
      const key =
        node.key ??
        (node.id === undefined
          ? undefined
          : `${node.parentId ?? ''}-${node.id}`);
      if (title.toLowerCase().includes(q.toLowerCase())) {
        // 将所有祖先 key 收集起来以便展开
        parents.forEach((p) => {
          if (p !== undefined && p !== null) parentKeys.add(p);
        });
      }
      collect(node.children, [...parents, key as number | string]);
    }
  };

  collect(treeData.value as any[]);
  expandedKeys.value = [...parentKeys];
  autoExpandParent.value = true;
});

onMounted(async () => {
  if (!userViewStore.isTenantUser()) {
    await userViewStore.fetchTenantList({ status: 'ON' });
  }

  await fetch();
});
</script>

<template>
  <!-- 分割线两侧等间距（issue #4）：m-3 恢复左栏与分割线的间距 -->
  <div class="dept-container m-3 mb-0 h-full">
    <a-card class="card-flat">
      <a-space direction="vertical" class="space-full">
        <div class="current-node">
          <span class="current-label">
            {{ $t('page.orgUnit.currentSelection') }}
          </span>
          <span class="current-value" :title="selectedTitle">
            {{ selectedTitle || $t('page.orgUnit.allUnits') }}
          </span>
        </div>
        <div class="input-row" v-if="!userViewStore.isTenantUser()">
          <div class="input-label">{{ $t('menu.tenant.member') }}</div>
          <Select
            show-search
            allow-clear
            class="search-input"
            :placeholder="$t('ui.input-search.placeholder')"
            v-model:value="selectedValue"
            :options="tenantOptions"
            :filter-option="filterOption"
            @change="handleTenantChanged"
          />
        </div>
        <!-- 搜索框独立一行（issue #4）：不再与机构标签挤 compact 一行 -->
        <div class="input-row">
          <a-input-search
            class="search-input"
            allow-clear
            size="middle"
            v-model:value="searchValue"
            :placeholder="$t('ui.input-search.placeholder')"
          />
          <VbenDropdownMenu
            :modal="false"
            :menus="toolbarList"
            class="dropdown-right"
          >
            <VbenIconButton>
              <LucideEllipsisVertical />
            </VbenIconButton>
          </VbenDropdownMenu>
        </div>
      </a-space>
    </a-card>

    <a-tree
      :expanded-keys="expandedKeys"
      :auto-expand-parent="autoExpandParent"
      :tree-data="treeData"
      :block-node="true"
      :selected-keys="effectiveSelectedKeys"
      @expand="handleExpandNode"
      @select="handleSelectNode"
      class="h-full w-full"
    >
      <template #title="{ title }">
        <span class="node-title">
          <LucideFolder class="node-icon" />
          <span v-if="title.indexOf(searchValue) > -1">
            {{ title.substring(0, title.indexOf(searchValue)) }}
            <span class="text-primary">{{ searchValue }}</span>
            {{
              title.substring(title.indexOf(searchValue) + searchValue.length)
            }}
          </span>
          <span v-else>{{ title }}</span>
        </span>
      </template>
    </a-tree>
  </div>
</template>

<style lang="less" scoped>
.dept-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  min-height: 0;
}

.input-row {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 8px;
  padding: 6px 8px; /* 控制左右空隙 */
  box-sizing: border-box;
}

.current-node {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 8px;
  min-width: 0;
  font-size: 13px;

  .current-label {
    color: hsl(var(--muted-foreground));
    white-space: nowrap;
  }

  .current-value {
    color: hsl(var(--foreground));
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.node-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;

  .node-icon {
    flex: 0 0 auto;
    width: 14px;
    height: 14px;
    color: hsl(var(--muted-foreground));
  }
}

.input-label {
  font-weight: 500;
  white-space: nowrap;
  text-align: left;
}

.input-area {
  flex: 1 1 0;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
}

.card-flat {
  display: flex;
  flex-direction: column;
  width: 100%;
  flex: 0 0 auto;
  box-sizing: border-box;
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  box-shadow: none;
  overflow: hidden;
  padding: 8px;
}

.card-flat ::v-deep(.ant-card-body) {
  padding: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: stretch;
  width: 100%;
  box-sizing: border-box;
}

.space-full {
  flex: 0 1 auto;
  min-width: 0;
  width: 100%;
}

.search-input {
  flex: 1 1 0;
  min-width: 0;
  width: 100%;
}

.dropdown-right {
  flex: 0 0 auto;
  margin-left: 8px;
}

.enter-button-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  margin: 0;
}

.dept-container > a-tree,
.dept-container > .ant-tree,
.dept-container > .h-full {
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
}
</style>
