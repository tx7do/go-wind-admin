<template>
  <div class="dept-container">
    <ElCard class="card-flat" shadow="never" :body-style="{ padding: '12px' }">
      <div class="toolbar-container">
        <!-- 租户选择器 -->
        <div v-if="!userViewStore.isTenantUser()" class="toolbar-item">
          <span class="toolbar-label">{{ $t("routes.tenant.member") }}</span>
          <ElSelect
            v-model="selectedValue"
            filterable
            clearable
            class="toolbar-select"
            :placeholder="$t('common.input-search.placeholder')"
            @change="handleTenantChanged"
          >
            <ElOption
              v-for="option in tenantOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
              :filter-method="filterOption"
            />
          </ElSelect>
        </div>

        <!-- 搜索框和工具栏 -->
        <div class="toolbar-item">
          <span class="toolbar-label">{{ $t("pages.org_unit.moduleName") }}</span>
          <div class="search-wrapper">
            <ElInput
              v-model="searchValue"
              class="search-input"
              clearable
              :placeholder="$t('common.input-search.placeholder')"
              :prefix-icon="Search"
            />
            <ElDropdown trigger="click" @command="handleToolbarClick">
              <ElLink :underline="false" class="more-btn">
                <ElIcon :size="18">
                  <More />
                </ElIcon>
              </ElLink>
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem v-for="item in toolbarList" :key="item.value" :command="item">
                    <ElIcon :size="14" class="dropdown-icon">
                      <Expand v-if="item.value === 'EXPAND_ALL'" />
                      <Fold v-else-if="item.value === 'COLLAPSE_ALL'" />
                      <CircleClose v-else />
                    </ElIcon>
                    {{ item.label }}
                  </ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
          </div>
        </div>
      </div>
    </ElCard>

    <ElTree
      ref="treeRef"
      :data="treeData"
      :props="{ label: 'label', children: 'children' }"
      :expand-on-click-node="false"
      :default-expanded-keys="expandedKeys"
      :default-checked-keys="selectedKeys"
      node-key="key"
      highlight-current
      class="tree-container"
      @current-change="handleSelectNode"
    >
      <template #default="{ node }">
        <span class="tree-node-label">
          <span v-if="searchValue && String(node.label).indexOf(searchValue) > -1">
            {{ String(node.label).substring(0, String(node.label).indexOf(searchValue)) }}
            <span class="highlight-text">{{ searchValue }}</span>
            {{
              String(node.label).substring(
                String(node.label).indexOf(searchValue) + searchValue.length
              )
            }}
          </span>
          <span v-else>{{ node.label }}</span>
        </span>
      </template>
    </ElTree>
  </div>
</template>

<script lang="ts" setup>
import type { TreeNode } from "element-plus";

import { computed, onMounted, ref, watch } from "vue";

import {
  ElTree,
  ElCard,
  ElInput,
  ElSelect,
  ElOption,
  ElLink,
  ElIcon,
  ElDropdown,
  ElDropdownMenu,
  ElDropdownItem,
} from "element-plus";
import { More, Search, Expand, Fold, CircleClose } from "@element-plus/icons-vue";
import { $t } from "@/i18n";

import { type identityservicev1_OrgUnit as OrgUnit } from "@/api/generated/admin/service/v1";
import { useUserViewStore } from "@/views/app/opm/user/user-view.state";

const userViewStore = useUserViewStore();

const toolbarList = [
  {
    value: "EXPAND_ALL",
    label: $t("common.tree.expand_all"),
    handler: handleMenuExpandAll,
    icon: Expand,
  },
  {
    value: "COLLAPSE_ALL",
    label: $t("common.tree.collapse_all"),
    handler: handleMenuCollapseAll,
    icon: Fold,
  },
  {
    value: "UNSELECT_ALL",
    label: $t("common.tree.unselect_all"),
    handler: handleMenuUnselectedAll,
    icon: CircleClose,
  },
];

const expandedKeys = ref<(number | string)[]>([]);
const selectedValue = ref<string>("");
const searchValue = ref<string>("");
const autoExpandParent = ref<boolean>(true);

const treeData = ref<any[]>([]);
const selectedKeys = ref<(number | string)[]>([]);

const tenantOptions = computed(() =>
  (userViewStore.tenantList.items ?? []).map((t: any) => ({
    label: t.name ?? t.code ?? String(t.id ?? ""),
    value: String(t.id ?? ""),
    raw: t,
  }))
);

const filterOption = (input: string, option: any) => {
  if (!option) return false;
  const text = String(option.value ?? option.label ?? "").toLowerCase();
  return text.includes(input.toLowerCase());
};

/**
 * 递归转换树形数据（替代 mapTree）
 */
function mapTreeData(nodes: OrgUnit[]): any[] {
  return nodes.map((node) => ({
    ...node,
    key: `${node.parentId}-${node.id}`,
    label: node.name,
    id: node.id,
    children: node.children && node.children.length > 0 ? mapTreeData(node.children) : [],
  }));
}

/**
 * 获取组织单元列表
 */
async function fetchOrgUnits() {
  try {
    const response = await userViewStore.fetchOrgUnitList();
    treeData.value = mapTreeData(response.items ?? []);
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
function handleMenuExpandAll() {
  const keys: (number | string)[] = [];

  const traverse = (nodes: any[] | undefined) => {
    if (!nodes || nodes.length === 0) return;
    for (const node of nodes) {
      if (node.children && node.children.length > 0) {
        if (node.key !== undefined && node.key !== null) {
          keys.push(node.key);
        } else if (node.id !== undefined || node.parentId !== undefined) {
          keys.push(`${node.parentId ?? ""}-${node.id ?? ""}`);
        }
        traverse(node.children);
      }
    }
  };

  traverse(treeData.value);
  expandedKeys.value = keys;
  autoExpandParent.value = true;
}

/**
 * 折叠所有节点
 */
function handleMenuCollapseAll() {
  expandedKeys.value = [];
  autoExpandParent.value = false;
}

/**
 * 取消选中所有节点
 */
function handleMenuUnselectedAll() {
  clearSelection();
}

/**
 * 选中组织单元
 */
function handleSelectOrgUnit(node: any) {
  userViewStore.setCurrentOrgUnitId(node ? node.id || null : null);
}

/**
 * 选中单个节点
 */
function handleSelectNode(node: TreeNode) {
  selectedKeys.value = [(node as any).key];
  handleSelectOrgUnit(node);
}

function handleTenantChanged(value: any) {
  userViewStore.setCurrentTenantId(value);
  fetchOrgUnits();
}

function clearSelection() {
  selectedKeys.value = [];
  handleSelectOrgUnit(null);
}

function handleToolbarClick(action: any) {
  action.handler();
}

watch(searchValue, (val) => {
  const q = String(val ?? "").trim();
  if (!q) {
    expandedKeys.value = [];
    autoExpandParent.value = false;
    return;
  }

  const parentKeys = new Set<number | string>();
  const collect = (nodes: any[] | undefined, parents: (number | string)[] = []) => {
    if (!nodes || nodes.length === 0) return;
    for (const node of nodes) {
      const title = String(node.label ?? "");
      const key = node.key;
      if (title.toLowerCase().includes(q.toLowerCase())) {
        parents.forEach((p) => {
          if (p !== undefined && p !== null) parentKeys.add(p);
        });
      }
      collect(node.children, [...parents, key as number | string]);
    }
  };

  collect(treeData.value);
  expandedKeys.value = [...parentKeys];
  autoExpandParent.value = true;
});

onMounted(async () => {
  if (!userViewStore.isTenantUser()) {
    await userViewStore.fetchTenantList({ status: "ON" });
  }

  await fetch();
});
</script>

<style lang="scss" scoped>
.dept-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  padding: 12px;
  background: var(--el-bg-color-page);

  .card-flat {
    flex: 0 0 auto;
    margin-bottom: 12px;
    border-radius: 8px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);

    :deep(.el-card__body) {
      padding: 12px;
    }
  }

  .toolbar-container {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .toolbar-item {
    display: flex;
    align-items: center;
    gap: 12px;

    .toolbar-label {
      flex-shrink: 0;
      font-size: 14px;
      font-weight: 500;
      color: var(--el-text-color-regular);
      white-space: nowrap;
      min-width: 70px;
    }

    .toolbar-select {
      flex: 1;
      min-width: 200px;
    }

    .search-wrapper {
      flex: 1;
      display: flex;
      align-items: center;
      gap: 8px;

      .search-input {
        flex: 1;
      }

      .more-btn {
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 32px;
        height: 32px;
        border-radius: 6px;
        transition: all 0.2s;

        &:hover {
          background: var(--el-fill-color);
          color: var(--el-color-primary);
        }
      }
    }
  }

  .tree-container {
    flex: 1;
    overflow-y: auto;
    min-height: 0;
    background: var(--el-bg-color);
    border-radius: 8px;
    padding: 8px;
    border: 1px solid var(--el-border-color-lighter);

    // 树节点样式优化
    :deep(.el-tree-node__content) {
      height: 36px;
      border-radius: 6px;
      margin-bottom: 2px;
      transition: all 0.2s;

      &:hover {
        background: var(--el-fill-color-light);
      }

      &.is-current {
        background: var(--el-color-primary-light-9);
        font-weight: 500;
      }
    }

    :deep(.el-tree-node__label) {
      font-size: 14px;
    }
  }

  .tree-node-label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    padding-right: 8px;
  }

  .highlight-text {
    color: var(--el-color-danger);
    font-weight: 600;
    background: var(--el-color-danger-light-9);
    padding: 0 2px;
    border-radius: 2px;
  }
}
</style>
