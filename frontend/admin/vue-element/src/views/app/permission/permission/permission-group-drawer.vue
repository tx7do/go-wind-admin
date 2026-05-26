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
      <ElFormItem :label="$t('pages.permission_group.name')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.permission_group.module')" prop="module">
        <ElInput
          v-model="formData.module"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.permission_group.parentId')" prop="parentId">
        <ElTreeSelect
          v-model="formData.parentId"
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
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import {
  statusList,
  fetchListPermissionGroups,
  useCreatePermissionGroup,
  useUpdatePermissionGroup,
  buildPermissionGroupTree,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createPermissionGroup } = useCreatePermissionGroup();
const { mutateAsync: updatePermissionGroup } = useUpdatePermissionGroup();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
  name: "",
  module: "",
  parentId: undefined as number | undefined,
  sortOrder: 1,
  status: "ON",
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  module: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.permission_group.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.permission_group.moduleName") })
);

// 权限分组树数据
const permissionGroupTreeData = ref<any[]>([]);

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

// 重置表单
function resetForm() {
  Object.assign(formData, {
    name: "",
    module: "",
    parentId: undefined,
    sortOrder: 1,
    status: "ON",
  });
  formRef.value?.clearValidate();
}

// 打开抽屉
async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;

  // 重置表单
  resetForm();

  // 加载权限分组树
  await loadPermissionGroupTree();

  // 编辑时填充数据
  if (data?.row && !isCreate.value) {
    Object.assign(formData, {
      name: data.row.name || "",
      module: data.row.module || "",
      parentId: data.row.parentId,
      sortOrder: data.row.sortOrder || 1,
      status: data.row.status || "ON",
    });
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

    if (isCreate.value) {
      await createPermissionGroup(formData);
      ElMessage.success($t("common.notification.create_success"));
    } else {
      await updatePermissionGroup({ id: currentId.value!, values: formData });
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
</style>
