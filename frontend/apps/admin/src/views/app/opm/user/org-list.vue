<script lang="ts" setup>
import type { TreeProps } from 'ant-design-vue';
import type { DefaultOptionType } from 'ant-design-vue/es/select';

import { computed, onMounted, ref, watch } from 'vue';

import { LucideEllipsisVertical } from '@vben/icons';
import { $t } from '@vben/locales';
import { VbenDropdownMenu, VbenIconButton } from '@vben-core/shadcn-ui';
import { mapTree } from '@vben-core/shared/utils';

import { Select } from 'ant-design-vue';

import { TreeActionEnum } from '#/constants/tree';
import { type identityservicev1_OrgUnit as OrgUnit } from '#/generated/api/admin/service/v1';
import { useUserViewStore } from '#/views/app/opm/user/user-view.state';

const userViewStore = useUserViewStore();

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
  console.log('expand all -> expandedKeys:', expandedKeys.value);
}

/**
 * 折叠所有节点
 */
function handleMenuCollapseAll(_data: any) {
  expandedKeys.value = [];
  autoExpandParent.value = false;
  console.log('collapse all -> expandedKeys cleared');
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
  console.log('selected node:', node);
  userViewStore.setCurrentOrgUnitId(node ? node.id || null : null);
}

/**
 * 选中单个节点
 */
function handleSelectNode(keys: (number | string)[], e?: any) {
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
  handleSelectOrgUnit(node ?? key);
}

function handleTenantChanged(value: any) {
  console.log('Selected tenant ID:', selectedValue.value);
  userViewStore.setCurrentTenantId(value);
  fetchOrgUnits();
}

function clearSelection() {
  selectedKeys.value = [];
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
  <div class="dept-container m-4 mb-0 ml-0 mt-0 h-full">
    <a-card class="card-flat">
      <a-space direction="vertical" class="space-full">
        <a-space-compact
          block
          class="input-row"
          v-if="!userViewStore.isTenantUser()"
        >
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
        </a-space-compact>
        <a-space-compact block class="input-row">
          <div class="input-label">{{ $t('page.orgUnit.moduleName') }}</div>
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
        </a-space-compact>
      </a-space>
    </a-card>

    <a-tree
      :expanded-keys="expandedKeys"
      :auto-expand-parent="autoExpandParent"
      :tree-data="treeData"
      :block-node="true"
      :selected-keys="selectedKeys"
      @expand="handleExpandNode"
      @select="handleSelectNode"
      class="h-full w-full"
    >
      <template #title="{ title }">
        <span v-if="title.indexOf(searchValue) > -1">
          {{ title.substring(0, title.indexOf(searchValue)) }}
          <span style="color: #f50">{{ searchValue }}</span>
          {{ title.substring(title.indexOf(searchValue) + searchValue.length) }}
        </span>
        <span v-else>{{ title }}</span>
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
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 8px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
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
