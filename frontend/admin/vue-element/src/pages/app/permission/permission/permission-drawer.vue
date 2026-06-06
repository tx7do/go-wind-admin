<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :loading="pageLoading"
    :config="{
      component: 'drawer',
      drawer: { size: '700px', closeOnClickModal: false },
    }"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
      v-loading="pageLoading"
    >
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.permission.name')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.permission.code')" prop="code">
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.permission.groupId')" prop="groupId">
        <ElTreeSelect
          v-model="formData.groupId"
          :data="permissionGroupTreeData"
          node-key="id"
          check-strictly
          :render-after-expand="false"
          default-expand-all
          filterable
          clearable
          :props="{ label: 'name', value: 'id', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="status">
        <ElRadioGroup v-model="formData.status">
          <ElRadioButton v-for="item in statusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <!-- 关联配置 -->
      <ElDivider content-position="left">{{ $t("common.section.configuration") }}</ElDivider>

      <ElFormItem :label="$t('pages.permission.menuIds')">
        <!-- 树工具栏 -->
        <div v-if="menuTreeData.length > 0" class="tree-toolbar">
          <ElButton text size="small" @click="menuExpandAll">
            {{ $t("common.tree.expand_all") }}
          </ElButton>
          <ElButton text size="small" @click="menuCollapseAll">
            {{ $t("common.tree.collapse_all") }}
          </ElButton>
          <ElButton text size="small" @click="menuSelectAll">
            {{ $t("common.tree.select_all") }}
          </ElButton>
          <ElButton text size="small" @click="menuUnselectAll">
            {{ $t("common.tree.unselect_all") }}
          </ElButton>
        </div>
        <ElTree
          v-if="menuTreeData.length > 0"
          ref="menuTreeRef"
          :data="menuTreeData"
          node-key="key"
          show-checkbox
          check-strictly
          :default-expanded-keys="menuExpandedKeys"
          :props="{ label: 'title', children: 'children' }"
          class="drawer-tree"
        />
        <div v-else class="tree-empty">
          {{ $t("common.tree.no_data") }}
        </div>
      </ElFormItem>

      <ElFormItem :label="$t('pages.permission.apiIds')">
        <!-- 树工具栏 -->
        <div v-if="apiTreeData.length > 0" class="tree-toolbar">
          <ElButton text size="small" @click="apiExpandAll">
            {{ $t("common.tree.expand_all") }}
          </ElButton>
          <ElButton text size="small" @click="apiCollapseAll">
            {{ $t("common.tree.collapse_all") }}
          </ElButton>
          <ElButton text size="small" @click="apiSelectAll">
            {{ $t("common.tree.select_all") }}
          </ElButton>
          <ElButton text size="small" @click="apiUnselectAll">
            {{ $t("common.tree.unselect_all") }}
          </ElButton>
        </div>
        <ElTree
          v-if="apiTreeData.length > 0"
          ref="apiTreeRef"
          :data="apiTreeData"
          node-key="key"
          show-checkbox
          :default-expanded-keys="apiExpandedKeys"
          :props="{ label: 'title', children: 'children' }"
          class="drawer-tree"
        />
        <div v-else class="tree-empty">
          {{ $t("common.tree.no_data") }}
        </div>
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, nextTick, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import {
  statusList,
  fetchListPermissionGroups,
  fetchListMenus,
  fetchListApis,
  useCreatePermission,
  useUpdatePermission,
  buildPermissionGroupTree,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createPermission } = useCreatePermission();
const { mutateAsync: updatePermission } = useUpdatePermission();

const visible = ref(false);
const submitLoading = ref(false);
const pageLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();
const menuTreeRef = ref();
const apiTreeRef = ref();

// 表单数据
const formData = reactive({
  name: "",
  code: "",
  groupId: null as number | null,
  status: "ON",
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.permission.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.permission.moduleName") })
);

// 树形数据
const permissionGroupTreeData = ref<any[]>([]);
const menuTreeData = ref<any[]>([]);
const apiTreeData = ref<any[]>([]);

// 受控展开 keys
const menuExpandedKeys = ref<(string | number)[]>([]);
const apiExpandedKeys = ref<(string | number)[]>([]);

// === 工具函数 ===

// 收集树所有节点的 key
function collectAllKeys(data: any[]): (string | number)[] {
  const keys: (string | number)[] = [];
  const walk = (nodes: any[]) => {
    for (const n of nodes) {
      keys.push(n.key);
      if (n.children?.length) walk(n.children);
    }
  };
  walk(data);
  return keys;
}

// 收集叶子节点 key（排除分组父节点）
function collectLeafKeys(data: any[]): (string | number)[] {
  const keys: (string | number)[] = [];
  const walk = (nodes: any[]) => {
    for (const n of nodes) {
      if (!n.children?.length) {
        keys.push(n.key);
      } else {
        walk(n.children);
      }
    }
  };
  walk(data);
  return keys;
}

// 过滤数字 ID（排除模块分组等非数字 key）
function filterNumbers(values: any[]): number[] {
  if (!Array.isArray(values)) return [];
  return values.filter((v) => typeof v === "number" && !isNaN(v)).map((v) => Number(v));
}

// === 树构建函数（参考 React buildMenuTree / buildApiTree） ===

// 构建菜单勾选树节点
function buildMenuCheckTree(
  items: Array<{
    id?: number | string;
    name?: string;
    parentId?: number | string | null;
    meta?: { title?: string };
  }>
): any[] {
  if (!items || items.length === 0) return [];

  const map = new Map<number | string, any>();
  items.forEach((item) => {
    if (item.id == null) return;
    map.set(item.id, {
      key: Number(item.id),
      title: item.meta?.title || item.name || "",
      children: [],
      parentId: item.parentId,
    });
  });

  const tree: any[] = [];
  map.forEach((node) => {
    if (node.parentId && map.has(node.parentId)) {
      map.get(node.parentId)!.children.push(node);
    } else {
      tree.push(node);
    }
  });

  // 清理空 children
  const cleanEmpty = (nodes: any[]) => {
    for (const n of nodes) {
      if (n.children && n.children.length === 0) {
        delete n.children;
      } else if (n.children) {
        cleanEmpty(n.children);
      }
    }
  };
  cleanEmpty(tree);
  return tree;
}

// 构建 API 勾选树节点（按模块分组）
function buildApiCheckTree(
  items: Array<{
    id?: number | string;
    method?: string;
    path?: string;
    description?: string;
    name?: string;
    module?: string;
  }>
): any[] {
  if (!items || items.length === 0) return [];

  const moduleMap = new Map<string, any[]>();
  items.forEach((item) => {
    const mod = item.module || $t("common.text.none");
    if (!moduleMap.has(mod)) moduleMap.set(mod, []);
    moduleMap.get(mod)!.push(item);
  });

  return Array.from(moduleMap.entries()).map(([mod, apis]) => ({
    key: `module_${mod}`,
    title: mod,
    children: apis.map((api) => ({
      key: Number(api.id),
      title: `${api.method || ""} ${api.path || ""}${
        api.description || api.name ? ` - ${api.description || api.name}` : ""
      }`,
    })),
  }));
}

function setAllExpanded(treeRef: any, expanded: boolean) {
  if (!treeRef.value?.store?.nodesMap) return;
  Object.values(treeRef.value.store.nodesMap).forEach((node: any) => {
    node.expanded = expanded;
  });
}

function menuExpandAll() {
  menuExpandedKeys.value = collectAllKeys(menuTreeData.value);
  setAllExpanded(menuTreeRef, true);
}
function menuCollapseAll() {
  menuExpandedKeys.value = [];
  setAllExpanded(menuTreeRef, false);
}
function menuSelectAll() {
  const keys = collectAllKeys(menuTreeData.value);
  menuTreeRef.value?.setCheckedKeys(keys);
}
function menuUnselectAll() {
  menuTreeRef.value?.setCheckedKeys([]);
}

// === API 树操作 ===

function apiExpandAll() {
  apiExpandedKeys.value = collectAllKeys(apiTreeData.value);
  setAllExpanded(apiTreeRef, true);
}
function apiCollapseAll() {
  apiExpandedKeys.value = [];
  setAllExpanded(apiTreeRef, false);
}
function apiSelectAll() {
  // 只收集叶子节点 key（排除模块分组节点）
  const keys = collectLeafKeys(apiTreeData.value);
  apiTreeRef.value?.setCheckedKeys(keys);
}
function apiUnselectAll() {
  apiTreeRef.value?.setCheckedKeys([]);
}

// === 数据加载 ===

// 加载权限分组树
async function loadPermissionGroupTree() {
  try {
    const result = await fetchListPermissionGroups(
      new PaginationQuery({ formValues: { status: "ON" } })
    );
    permissionGroupTreeData.value = buildPermissionGroupTree(result.items || []);
  } catch (error) {
    console.error("Failed to load permission group tree:", error);
  }
}

// 加载菜单树
async function loadMenuTree() {
  try {
    const result = await fetchListMenus(new PaginationQuery({ formValues: { status: "ON" } }));
    const tree = buildMenuCheckTree(result.items || []);
    menuTreeData.value = tree;
    menuExpandedKeys.value = collectAllKeys(tree);
  } catch (error) {
    console.error("Failed to load menu tree:", error);
  }
}

// 加载 API 树
async function loadApiTree() {
  try {
    const result = await fetchListApis(new PaginationQuery({}));
    const tree = buildApiCheckTree(result.items || []);
    apiTreeData.value = tree;
    apiExpandedKeys.value = collectAllKeys(tree);
  } catch (error) {
    console.error("Failed to load API tree:", error);
  }
}

// === 表单操作 ===

// 重置表单
function resetForm() {
  Object.assign(formData, {
    name: "",
    code: "",
    groupId: null,
    status: "ON",
  });
  formRef.value?.clearValidate();
  menuTreeRef.value?.setCheckedKeys([]);
  apiTreeRef.value?.setCheckedKeys([]);
}

// 打开抽屉
async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;

  // 重置表单
  resetForm();

  // 加载树形数据（显示加载状态）
  pageLoading.value = true;
  try {
    await Promise.all([loadPermissionGroupTree(), loadMenuTree(), loadApiTree()]);

    // 编辑时填充数据
    if (data?.row && !isCreate.value) {
      Object.assign(formData, {
        name: data.row.name || "",
        code: data.row.code || "",
        groupId: data.row.groupId ?? null,
        status: data.row.status || "ON",
      });
      // 等待树渲染后设置选中节点
      nextTick(() => {
        menuTreeRef.value?.setCheckedKeys(data.row.menuIds || []);
        apiTreeRef.value?.setCheckedKeys(data.row.apiIds || []);
      });
    }
  } finally {
    pageLoading.value = false;
  }
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const submitData = {
      ...formData,
      menuIds: filterNumbers(menuTreeRef.value?.getCheckedKeys() || []),
      apiIds: filterNumbers(apiTreeRef.value?.getCheckedKeys() || []),
    };

    if (isCreate.value) {
      await createPermission(submitData);
      ElMessage.success($t("common.notification.create_success"));
    } else {
      await updatePermission({ id: currentId.value!, values: submitData });
      ElMessage.success($t("common.notification.update_success"));
    }

    handleClose();
    emit("success");
  } catch (error) {
    if (error !== false) {
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.create_failed")
          : $t("common.notification.update_failed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 12px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.drawer-tree {
  width: 100%;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  padding: 4px;
  max-height: 280px;
  overflow-y: auto;
}

.tree-toolbar {
  display: flex;
  gap: 4px;
  margin-bottom: 4px;
}

.tree-empty {
  color: var(--el-text-color-placeholder);
  padding: 16px;
  text-align: center;
  border: 1px dashed var(--el-border-color);
  border-radius: var(--el-border-radius-base);
}
</style>
