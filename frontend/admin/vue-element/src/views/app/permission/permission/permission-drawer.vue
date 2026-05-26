<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    size="700px"
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

      <ElFormItem :label="$t('pages.permission.menuIds')" prop="menuIds">
        <ElTreeSelect
          v-model="formData.menuIds"
          :data="menuTreeData"
          node-key="id"
          multiple
          :render-after-expand="false"
          filterable
          clearable
          :props="{ label: 'meta.title', value: 'id', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.permission.apiIds')" prop="apiIds">
        <ElTreeSelect
          v-model="formData.apiIds"
          :data="apiTreeData"
          node-key="key"
          multiple
          :render-after-expand="false"
          filterable
          clearable
          :props="{ label: 'title', value: 'key', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
        />
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
  fetchListMenus,
  fetchListApis,
  useCreatePermission,
  useUpdatePermission,
  buildPermissionGroupTree,
  buildMenuTree,
  convertApiToTree,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createPermission } = useCreatePermission();
const { mutateAsync: updatePermission } = useUpdatePermission();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
  name: "",
  code: "",
  groupId: undefined as number | undefined,
  status: "ON",
  menuIds: [] as number[],
  apiIds: [] as (number | string)[],
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
    menuTreeData.value = buildMenuTree(result.items || []);
  } catch (error) {
    console.error("Failed to load menu tree:", error);
  }
}

// 加载 API 树
async function loadApiTree() {
  try {
    const result = await fetchListApis(new PaginationQuery({}));
    apiTreeData.value = convertApiToTree(result.items || []);
  } catch (error) {
    console.error("Failed to load API tree:", error);
  }
}

// 过滤数字 ID
function filterNumbers(ids: (number | string)[]): number[] {
  return ids
    .filter((id) => typeof id === "number" || /^\d+$/.test(String(id)))
    .map((id) => Number(id));
}

// 重置表单
function resetForm() {
  Object.assign(formData, {
    name: "",
    code: "",
    groupId: undefined,
    status: "ON",
    menuIds: [],
    apiIds: [],
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

  // 加载树形数据
  await Promise.all([loadPermissionGroupTree(), loadMenuTree(), loadApiTree()]);

  // 编辑时填充数据
  if (data?.row && !isCreate.value) {
    Object.assign(formData, {
      name: data.row.name || "",
      code: data.row.code || "",
      groupId: data.row.groupId,
      status: data.row.status || "ON",
      menuIds: data.row.menuIds || [],
      apiIds: data.row.apiIds || [],
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

    const submitData = {
      ...formData,
      menuIds: formData.menuIds?.length > 0 ? filterNumbers(formData.menuIds) : [],
      apiIds: formData.apiIds?.length > 0 ? filterNumbers(formData.apiIds) : [],
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
</style>
