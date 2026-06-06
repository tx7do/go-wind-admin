<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
    :config="{
      component: 'drawer',
      drawer: { size: drawer.drawerWidth, closeOnClickModal: false },
    }"
  >
    <ElForm
      ref="formRef"
      :model="drawer.formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <ElFormItem :label="$t('pages.permission_group.name')" prop="name">
        <ElInput v-model="drawer.formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.permission_group.module')" prop="module">
        <ElInput
          v-model="drawer.formData.module"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.permission_group.parentId')" prop="parentId">
        <ElTreeSelect
          v-model="drawer.formData.parentId"
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
          v-model="drawer.formData.sortOrder"
          :min="1"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="status">
        <ElRadioGroup v-model="drawer.formData.status">
          <ElRadioButton v-for="item in statusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="drawer.close">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton
          type="primary"
          :loading="drawer.submitLoading.value"
          @click="drawer.handleSubmit(formRef, () => emit('success'))"
        >
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";

import {
  statusList,
  fetchListPermissionGroups,
  useCreatePermissionGroup,
  useUpdatePermissionGroup,
  buildPermissionGroupTree,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createPermissionGroup } = useCreatePermissionGroup();
const { mutateAsync: updatePermissionGroup } = useUpdatePermissionGroup();

const formRef = ref();
const permissionGroupTreeData = ref<any[]>([]);

const drawer = useDrawerForm({
  moduleKey: "pages.permission_group.moduleName",
  defaults: {
    name: "",
    module: "",
    parentId: undefined as number | undefined,
    sortOrder: 1,
    status: "ON",
  },
  createFn: createPermissionGroup,
  updateFn: (id, values) => updatePermissionGroup({ id, values }),
  asyncSetup: async () => {
    const result = await fetchListPermissionGroups(
      new PaginationQuery({ formValues: { status: "ON" } })
    );
    permissionGroupTreeData.value = buildPermissionGroupTree(result.items || []);
  },
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  module: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 暴露方法
defineExpose({ open: drawer.open });
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
