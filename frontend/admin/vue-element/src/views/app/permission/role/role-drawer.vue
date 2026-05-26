<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    size="600px"
    :close-on-click-modal="false"
    :append-to-body="true"
    :destroy-on-close="true"
    @close="handleClose"
  >
    <ElForm
      ref="formRef"
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

      <ElFormItem :label="$t('common.table.description')" prop="description">
        <ElInput
          v-model="formData.description"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <!-- 权限配置 -->
      <ElDivider content-position="left">{{ $t("pages.role.permissions") }}</ElDivider>

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
              <span v-if="data.key" class="text-xs text-gray-400 ml-2">{{ data.key }}</span>
            </span>
          </template>
        </ElTree>
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
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, nextTick } from "vue";
import { ElMessage } from "element-plus";
import type { TreeInstance } from "element-plus";

import {
  useCreateRole,
  useUpdateRole,
  fetchListPermissionGroups,
  fetchListPermissions,
  statusList,
  buildPermissionTree,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createRole } = useCreateRole();
const { mutateAsync: updateRole } = useUpdateRole();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();
const permissionTreeRef = ref<TreeInstance>();

// 表单数据
const formData = reactive({
  name: "",
  code: "",
  sortOrder: 1,
  status: "ON",
  description: "",
  permissions: [] as number[],
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.role.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.role.moduleName") })
);

// 权限树数据
const permissionTreeData = ref<any[]>([]);

// 加载权限树
async function loadPermissionTree() {
  try {
    const groupData = await fetchListPermissionGroups(
      new PaginationQuery({ formValues: { status: "ON" } })
    );
    const groups = groupData.items ?? [];

    const permissionData = await fetchListPermissions(
      new PaginationQuery({ formValues: { status: "ON" } })
    );

    permissionTreeData.value = buildPermissionTree(groups, permissionData.items || []);
  } catch (error) {
    console.error("Failed to load permission tree:", error);
  }
}

// 过滤权限节点
function filterPermissionNode(value: string, data: any) {
  if (!value) return true;
  return data.title?.toLowerCase().includes(value.toLowerCase());
}

// 打开抽屉
async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;

  // 重置表单
  resetForm();

  // 加载权限树
  await loadPermissionTree();

  // 如果是编辑模式，填充数据
  if (!isCreate.value && data?.row) {
    Object.assign(formData, data.row);

    // 设置权限树的选中状态
    await nextTick();
    if (data.row.permissions && Array.isArray(data.row.permissions)) {
      permissionTreeRef.value?.setCheckedKeys(data.row.permissions);
    }
  }
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 重置表单
function resetForm() {
  formData.name = "";
  formData.code = "";
  formData.sortOrder = 1;
  formData.status = "ON";
  formData.description = "";
  formData.permissions = [];

  formRef.value?.clearValidate();
  permissionTreeRef.value?.setCheckedKeys([]);
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    // 获取选中的权限ID（只保留数字类型的叶子节点）
    const checkedKeys = permissionTreeRef.value?.getCheckedKeys(false) || [];
    const permissions = checkedKeys.filter((key: any) => typeof key === "number");

    const values = {
      ...formData,
      permissions,
    };

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
}
</style>
