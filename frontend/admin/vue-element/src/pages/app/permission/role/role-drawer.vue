<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :loading="pageLoading"
    :config="{
      component: 'drawer',
      drawer: { size: DRAWER_WIDTH },
    }"
  
    :close-guard="guard.allowClose">
    <ElForm
      ref="formRef"
      v-loading="pageLoading"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.role.name')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.role.code')" prop="code">
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.sortOrder')" prop="sortOrder">
        <ElInputNumber
          v-model="formData.sortOrder"
          :min="1"
          :placeholder="$t('common.placeholder.input')"
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

      <ElFormItem :label="$t('pages.role.dataScope')" prop="dataScope">
        <ElSelect v-model="formData.dataScope" :placeholder="$t('common.placeholder.select')" style="width: 100%">
          <ElOption
            v-for="item in roleDataScopeList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <!-- 权限继承：父角色（可选）。子角色的可选权限与保存校验均受父角色约束 -->
      <ElFormItem :label="$t('pages.role.parentId')" prop="parentId">
        <ElTreeSelect
          v-model="formData.parentId"
          :data="roleTreeOptions"
          :placeholder="$t('pages.role.parentIdPlaceholder')"
          node-key="value"
          :props="{ label: 'label', children: 'children' }"
          filterable
          clearable
          check-strictly
          default-expand-all
          style="width: 100%"
          @change="(v: any) => handleParentChange(typeof v === 'number' ? v : 0)"
        />
      </ElFormItem>

      <!-- SELECTED_UNITS 档位的自定义授权单元集 -->
      <ElFormItem v-if="formData.dataScope === 'SELECTED_UNITS'" :label="$t('pages.role.orgUnits')">
        <template v-if="orgUnitTreeData.length > 0">
          <ElTree
            ref="unitTreeRef"
            :data="orgUnitTreeData"
            node-key="id"
            show-checkbox
            default-expand-all
            :props="{ label: 'name', children: 'children' }"
            style="max-height: 400px; overflow-y: auto"
          />
        </template>
        <div v-else class="org-unit-empty">{{ $t("pages.role.noOrgUnitData") }}</div>
      </ElFormItem>

      <ElFormItem prop="description">
        <template #label>
          <span class="inline-flex items-center">
            {{ $t("common.table.description") }}
            <AiGenerateButton
              scene="DESCRIPTION"
              @generate="(content: string) => (formData.description = content)"
            />
          </span>
        </template>
        <ElInput
          v-model="formData.description"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <!-- 权限配置 -->
      <ElDivider content-position="left">{{ $t("pages.role.permissions") }}</ElDivider>

      <div v-if="permCapIds != null" class="perm-cap-hint">
        {{ $t("pages.role.permCapHint") }}
      </div>

      <ElFormItem prop="permissions">
        <ElTree
          ref="permissionTreeRef"
          :data="permissionTreeData"
          node-key="key"
          show-checkbox
          default-expand-all
          :props="{ label: 'title', children: 'children' }"
          :filter-node-method="filterPermissionNode"
          style="max-height: 400px; overflow-y: auto"
        >
          <template #default="{ node, data }">
            <span class="custom-tree-node">
              <span>{{ node.label }}</span>
              <span v-if="data.key" class="tree-node__key text-xs ml-2">{{ data.key }}</span>
            </span>
          </template>
        </ElTree>
      </ElFormItem>

      <!-- 字段权限：勾选 = 对该角色用户隐藏（黑名单语义，User 资源试点） -->
      <ElFormItem :label="$t('pages.role.fieldPerm.title')">
        <div class="field-perm-box">
          <div class="field-perm-hint">{{ $t("pages.role.fieldPerm.hint") }}</div>
          <ElCheckboxGroup v-model="userHiddenFields">
            <ElCheckbox v-for="item in fieldPermOptions" :key="item.value" :value="item.value">
              {{ item.label }}
            </ElCheckbox>
          </ElCheckboxGroup>
        </div>
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="guard.requestClose(handleClose)">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, nextTick } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { TreeInstance } from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerCloseGuard } from "@/components/Pro/composables/use-drawer-close-guard";
import AiGenerateButton from "@/components/AiGenerateButton/index.vue";

import {
  useCreateRole,
  useUpdateRole,
  fetchGetRole,
  fetchListRoles,
  fetchListPermissionGroups,
  fetchListPermissions,
  fetchListOrgUnits,
  statusList,
  roleDataScopeList,
  buildPermissionTree,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createRole } = useCreateRole();
const { mutateAsync: updateRole } = useUpdateRole();

const visible = ref(false);
const submitLoading = ref(false);
const pageLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
// 编辑模式的行数据快照（提交时的越权收敛确认需要比对此处保存的存量权限）
const editRow = ref<any>(undefined);
const formRef = ref();
const permissionTreeRef = ref<TreeInstance>();
const unitTreeRef = ref<TreeInstance>();

// 表单数据
const formData = reactive({
  name: "",
  code: "",
  sortOrder: 1,
  status: "ON",
  dataScope: "ALL",
  description: "",
  // 父角色ID，0=根角色。清空选择时 @change 归一为 0，保证显式 0 进 updateMask
  parentId: 0,
  permissions: [] as number[],
});

// ========== 权限继承（父子角色） ==========

// 父角色候选树（排除自身及后代防环，后端另有环检测兜底）
const roleTreeOptions = ref<any[]>([]);

// 父角色权限上限集（null=无父角色，不收窄）
const permCapIds = ref<number[] | null>(null);

// 权限树全量数据源，父角色切换时按 cap 过滤重建
const permissionTreeSource = ref<{ groups: any[]; permissions: any[] } | null>(null);

/** 收集自身及全部后代角色 ID（按 parentId 嵌套） */
function collectSelfAndDescendantIds(items: any[], selfId?: number): number[] {
  if (!selfId) return [];
  const byParent = new Map<number, number[]>();
  for (const it of items || []) {
    const pid = Number(it.parentId || 0);
    if (!byParent.has(pid)) byParent.set(pid, []);
    byParent.get(pid)!.push(Number(it.id));
  }
  const result = new Set<number>([Number(selfId)]);
  const stack = [Number(selfId)];
  while (stack.length > 0) {
    const cur = stack.pop()!;
    for (const child of byParent.get(cur) || []) {
      if (!result.has(child)) {
        result.add(child);
        stack.push(child);
      }
    }
  }
  return [...result];
}

/** 角色列表按 parentId 嵌套为 TreeSelect 数据（森林，缺父孤行落为根），排除 excludeIds */
function buildRoleTreeOptions(items: any[], excludeIds: number[]): any[] {
  const excluded = new Set(excludeIds.map(Number));
  const nodes = new Map<number, any>();
  for (const it of items || []) {
    const id = Number(it.id);
    if (excluded.has(id)) continue;
    nodes.set(id, {
      value: id,
      label: it.name || it.code || String(id),
      pid: Number(it.parentId || 0),
      children: [] as any[],
    });
  }
  const roots: any[] = [];
  for (const node of nodes.values()) {
    const parent = node.pid > 0 ? nodes.get(node.pid) : undefined;
    if (parent) {
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  }
  const strip = (list: any[]): any[] =>
    list.map((n) => {
      const { pid: _pid, ...rest } = n;
      if (rest.children.length > 0) {
        rest.children = strip(rest.children);
      } else {
        delete rest.children;
      }
      return rest;
    });
  return strip(roots);
}

/**
 * 按父角色权限上限集（cap）过滤权限树数据源——在 buildPermissionTree 之前调用。
 * 只保留 cap 内的权限点，以及（递归）仍含至少一个保留权限点的权限组。
 * cap 为 null/undefined/空数组表示无父角色（根角色），原样返回全量。
 */
function filterPermissionSourceByCap(
  groups: any[],
  permissions: any[],
  capIds: number[] | null | undefined,
): { groups: any[]; permissions: any[] } {
  if (!capIds || capIds.length === 0) return { groups, permissions };
  const cap = new Set(capIds.map(Number));
  const kept = (permissions || []).filter((p) => cap.has(Number(p.id)));
  const filterGroups = (list: any[]): any[] =>
    (list || [])
      .map((g) => ({ ...g, children: filterGroups(g.children || []) }))
      .filter(
        (g) => g.children.length > 0 || kept.some((p) => String(p.groupId) === String(g.id)),
      );
  return { groups: filterGroups(groups), permissions: kept };
}

/** 按父角色上限集重建权限树；返回本次生效的 cap（拉取失败按全量展示，后端保存兜底） */
async function rebuildPermissionTree(parentId: number | null): Promise<number[] | null> {
  const src = permissionTreeSource.value;
  if (!src) return null;
  let cap: number[] | null = null;
  if (parentId) {
    try {
      const parent = await fetchGetRole({ id: parentId });
      cap = ((parent as any)?.permissions ?? [])
        .map((v: any) => Number(v))
        .filter((v: any) => Number.isFinite(v));
    } catch (error) {
      console.error("Failed to load parent role permission cap:", error);
    }
  }
  permCapIds.value = cap;
  const filtered = filterPermissionSourceByCap(src.groups, src.permissions, cap);
  permissionTreeData.value =
    filtered.groups.length === 0 && filtered.permissions.length > 0
      ? filtered.permissions.map((p) => ({
          key: Number(p.id),
          title: p.name || p.code || String(p.id),
        }))
      : buildPermissionTree(filtered.groups, filtered.permissions);
  return cap;
}

/** 切换父角色：重建权限树并把已勾权限收敛到新 cap 内 */
async function handleParentChange(pid: number) {
  formData.parentId = pid;
  const cap = await rebuildPermissionTree(pid || null);
  if (pid && cap && permissionTreeRef.value) {
    const keys = (permissionTreeRef.value.getCheckedKeys(false) || []).filter(
      (k: any) => typeof k === "number" && cap!.includes(Number(k))
    );
    permissionTreeRef.value.setCheckedKeys(keys);
  }
}

/** 加载父角色候选下拉（启用角色全集，排除自身及后代） */
async function loadRoleTreeOptions() {
  try {
    const result = await fetchListRoles(
      new PaginationQuery({ paging: { page: 1, pageSize: 1000 }, formValues: { status: "ON" } })
    );
    roleTreeOptions.value = buildRoleTreeOptions(
      result.items || [],
      collectSelfAndDescendantIds(result.items || [], currentId.value)
    );
  } catch (error) {
    console.error("Failed to load parent role options:", error);
    roleTreeOptions.value = [];
  }
}

// 组织单元树数据（SELECTED_UNITS 档位的自定义授权集）
const orgUnitTreeData = ref<any[]>([]);

// 字段权限：User 资源上勾选隐藏的字段（proto 字段 json_name）
const userHiddenFields = ref<string[]>([]);

// 可勾选字段：value 与后端 identity User proto 字段 json_name 逐字一致，
// 提交后经登录聚合写入令牌，命中字段在响应侧被裁剪。
const fieldPermOptions = [
  { value: "email", label: $t("pages.role.fieldPerm.field.email") },
  { value: "mobile", label: $t("pages.role.fieldPerm.field.mobile") },
  { value: "telephone", label: $t("pages.role.fieldPerm.field.telephone") },
  { value: "address", label: $t("pages.role.fieldPerm.field.address") },
  { value: "region", label: $t("pages.role.fieldPerm.field.region") },
  { value: "lastLoginAt", label: $t("pages.role.fieldPerm.field.lastLoginAt") },
  { value: "lastLoginIp", label: $t("pages.role.fieldPerm.field.lastLoginIp") },
];

// 保留角色码：platform: / template: 命名空间与内置租户管理员码承载登录态
// 身份判定语义（后端按码值精确匹配填管理员标志），禁止用户自建角色占用。
// 前端拦截只是即时反馈，后端 Create/Update 仍有一道校验兜底。
const RESERVED_ROLE_CODE_PATTERN = /^(platform:|template:)/;
const isReservedRoleCode = (v: string) =>
  RESERVED_ROLE_CODE_PATTERN.test(v) || v === "tenant:manager";

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [
    { required: true, message: $t("common.validation.required"), trigger: "blur" },
    {
      validator: (_rule: any, value: string, callback: (err?: Error) => void) => {
        if (value && isReservedRoleCode(value)) {
          callback(new Error($t("pages.role.reservedCode")));
          return;
        }
        callback();
      },
      trigger: "blur",
    },
  ],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  dataScope: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.role.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.role.moduleName") })
);

// 权限树数据
const permissionTreeData = ref<any[]>([]);

// 加载权限树全量数据源（按 cap 过滤重建由 rebuildPermissionTree 负责）
async function loadPermissionTree() {
  try {
    const groupData = await fetchListPermissionGroups(
      new PaginationQuery({ formValues: { status: "ON" } })
    );
    const groups = groupData.items ?? [];

    const permissionData = await fetchListPermissions(
      new PaginationQuery({ formValues: { status: "ON" } })
    );

    permissionTreeSource.value = { groups, permissions: permissionData.items || [] };
  } catch (error) {
    console.error("Failed to load permission tree:", error);
  }
}

// 过滤权限节点
function filterPermissionNode(value: string, data: any) {
  if (!value) return true;
  return data.title?.toLowerCase().includes(value.toLowerCase());
}

// 加载组织单元树（SELECTED_UNITS 档位的自定义授权集配置用）
async function loadOrgUnitTree() {
  try {
    const result = await fetchListOrgUnits(new PaginationQuery({ formValues: { status: "ON" } }));
    orgUnitTreeData.value = result.items || [];
  } catch (error) {
    console.error("Failed to load org unit tree:", error);
  }
}

// 打开抽屉
// 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）；
// 权限树/授权单元树的勾选与字段权限中间态也是编辑态，纳入脏判定
const guard = useDrawerCloseGuard(() => ({
  ...formData,
  permissionKeys: permissionTreeRef.value?.getCheckedKeys(false),
  unitKeys: unitTreeRef.value?.getCheckedKeys(false),
  userHiddenFields: userHiddenFields.value,
}));

async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;
  editRow.value = isCreate.value ? undefined : data?.row;

  // 重置表单
  resetForm();

  // 加载树数据（显示加载状态）
  pageLoading.value = true;
  try {
    await Promise.all([loadPermissionTree(), loadOrgUnitTree(), loadRoleTreeOptions()]);

    // 编辑模式的初始父角色（回填值），据此收窄权限树
    const initialParentId = !isCreate.value ? Number(data?.row?.parentId || 0) : 0;
    await rebuildPermissionTree(initialParentId || null);

    // 如果是编辑模式，填充数据
    if (!isCreate.value && data?.row) {
      // 仅回填表单声明的字段，避免把 id/createdAt/tenantName/tenantId 等
      // 不可变字段灌入 formData，进而被 ...formData 带入提交载荷。
      // （permissions / orgUnits 由树勾选控制，由 setCheckedKeys 回填，不在此赋值）
      formData.name = data.row.name ?? "";
      formData.code = data.row.code ?? "";
      formData.sortOrder = data.row.sortOrder ?? 1;
      formData.status = data.row.status ?? "ON";
      formData.dataScope = data.row.dataScope ?? "ALL";
      formData.description = data.row.description ?? "";
      formData.parentId = Number(data.row.parentId || 0);

      // 设置树的选中状态：超出父角色上限（树上不可见）的存量勾选先收敛，
      // 提交时由 outOfCap 确认流程向操作者说明（编辑时校验语义）。
      await nextTick();
      const storedPerms = Array.isArray(data.row.permissions)
        ? data.row.permissions.filter((v: any) => typeof v === "number")
        : [];
      const effPerms = permCapIds.value
        ? storedPerms.filter((id: number) => permCapIds.value!.includes(Number(id)))
        : storedPerms;
      permissionTreeRef.value?.setCheckedKeys(effPerms);
      if (formData.dataScope === "SELECTED_UNITS" && Array.isArray(data.row.orgUnits)) {
        unitTreeRef.value?.setCheckedKeys(data.row.orgUnits);
      }

      // 回填字段权限（User 资源的隐藏字段集）
      const fpEntry = Array.isArray(data.row.fieldPermissions)
        ? data.row.fieldPermissions.find((e: any) => e?.resource === "User")
        : undefined;
      userHiddenFields.value = Array.isArray(fpEntry?.hiddenFields)
        ? fpEntry.hiddenFields.filter((v: any) => typeof v === "string")
        : [];
    }
  } finally {
    pageLoading.value = false;
  }

  // 填充完成后记录基线快照（回填不算脏）
  guard.syncBaseline();
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 重置表单
function resetForm() {
  // 先删除所有 formData 上可能被历史 Object.assign 残留的多余 key
  // （id/createdAt/tenantName/tenantId 等），再恢复声明字段的默认值，
  // 确保跨抽屉开启无脏数据残留。
  Object.keys(formData).forEach((k) => {
    delete (formData as any)[k];
  });
  formData.name = "";
  formData.code = "";
  formData.sortOrder = 1;
  formData.status = "ON";
  formData.dataScope = "ALL";
  formData.description = "";
  formData.parentId = 0;
  formData.permissions = [];

  formRef.value?.clearValidate();
  permissionTreeRef.value?.setCheckedKeys([]);
  unitTreeRef.value?.setCheckedKeys([]);
  orgUnitTreeData.value = [];
  userHiddenFields.value = [];
  permCapIds.value = null;
  permissionTreeSource.value = null;
  roleTreeOptions.value = [];
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  // 保存收敛确认：编辑模式下超出父角色上限（树上不可见）的存量勾选会被剥离，
  // 先向操作者确认（编辑时校验语义的前端引导；后端亦有子集校验兜底）。
  if (!isCreate.value && permCapIds.value) {
    const storedPerms: number[] = Array.isArray(editRow.value?.permissions)
      ? editRow.value.permissions.filter((v: any) => typeof v === "number")
      : [];
    const outOfCap = storedPerms.filter(
      (id) => !permCapIds.value!.includes(Number(id))
    );
    if (outOfCap.length > 0) {
      try {
        await ElMessageBox.confirm(
          $t("pages.role.permTrimConfirmContent", { count: outOfCap.length }),
          $t("pages.role.permTrimConfirmTitle"),
          {
            type: "warning",
            confirmButtonText: $t("common.button.confirm"),
            cancelButtonText: $t("common.button.cancel"),
          }
        );
      } catch {
        return; // 用户取消收敛，维持现状不提交
      }
    }
  }

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    // 获取选中的权限ID（只保留数字类型的叶子节点）
    const checkedKeys = permissionTreeRef.value?.getCheckedKeys(false) || [];
    const permissions = checkedKeys.filter((key: any) => typeof key === "number");

    const values: Record<string, any> = {
      ...formData,
      permissions,
      // 父角色始终随表单提交：0=根角色（清空父角色须显式 0 进 updateMask，
      // 否则后端三态语义会把它当"未声明"静默跳过）。
      parentId: formData.parentId || 0,
    };

    // 仅 SELECTED_UNITS 档提交授权单元集（含清空场景）；
    // 其余档位不携带该字段，后端维持既有集不替换。
    if (formData.dataScope === "SELECTED_UNITS") {
      const unitCheckedKeys = unitTreeRef.value?.getCheckedKeys(false) || [];
      values.orgUnits = unitCheckedKeys.filter((key: any) => typeof key === "number");
    }

    // 字段权限始终随表单提交（含清空场景）：抽屉所见即保存后的最终态。
    values.fieldPermissions =
      userHiddenFields.value.length > 0
        ? [{ resource: "User", hiddenFields: [...userHiddenFields.value] }]
        : [];

    if (isCreate.value) {
      await createRole(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateRole({ id: currentId.value!, values });
      ElMessage.success($t("common.notification.updateSuccess"));
    }

    emit("success");
    handleClose();
  } catch (error) {
    if (error !== false) {
      // 不是验证错误
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.createFailed")
          : $t("common.notification.updateFailed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法给父组件
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.custom-tree-node {
  display: flex;
  align-items: center;
  flex: 1;

  // 权限码提示文字：tailwind gray-400 (#9CA3AF) 在浅色白底上实测仅 2.54:1，
  // 改走 EP 次要文字 token（§5.2 禁止硬编码色值）
  .tree-node__key {
    color: var(--el-text-color-secondary);
  }
}

.org-unit-empty {
  width: 100%;
  box-sizing: border-box;
  padding: 16px 0;
  text-align: center;
  font-size: 14px;
  border: 1px dashed var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  color: var(--el-text-color-secondary);
}

.field-perm-box {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
}

.field-perm-hint {
  margin-bottom: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.perm-cap-hint {
  width: 100%;
  margin-bottom: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
</style>
