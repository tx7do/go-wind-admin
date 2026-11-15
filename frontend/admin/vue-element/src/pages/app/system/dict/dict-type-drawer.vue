<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :close-guard="drawer.closeGuard"
    :title="drawer.title.value"
    :config="{
      component: 'drawer',
      drawer: { size: drawer.drawerWidth },
    }"
  >
    <ElForm
      ref="formRef"
      :model="drawer.formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <ElFormItem :label="$t('pages.dict.typeName')" prop="typeName">
        <ElInput
          v-model="drawer.formData.typeName"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.dict.typeCode')" prop="typeCode">
        <ElInput
          v-model="drawer.formData.typeCode"
          :placeholder="$t('common.placeholder.input')"
          clearable
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

      <ElFormItem :label="$t('common.table.status')" prop="isEnabled">
        <ElRadioGroup v-model="drawer.formData.isEnabled">
          <ElRadioButton
            v-for="item in enableBoolList"
            :key="String(item.value)"
            :value="item.value"
          >
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="drawer.guard.requestClose(drawer.close)">{{ $t("common.button.cancel") }}</ElButton>
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
import { enableBoolList, useCreateDictType, useUpdateDictType } from "@/api/composables";
import { $t } from "@go-wind/admin-element";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createDictType } = useCreateDictType();
const { mutateAsync: updateDictType } = useUpdateDictType();

const formRef = ref();

const drawer = useDrawerForm({
  moduleKey: "pages.dict.dictType",
  defaults: {
    typeName: "",
    typeCode: "",
    sortOrder: 1,
    isEnabled: true,
  },
  createFn: createDictType,
  updateFn: (id, values) => updateDictType({ id, values }),
});

// 表单验证规则
const formRules = {
  typeName: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  typeCode: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  isEnabled: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
};

// 包装 open：编辑模式下显式回填表单声明字段（避免 useDrawerForm 默认不回填导致编辑态空白）
function open(data?: { create?: boolean; row?: any }) {
  drawer.open(data, (row: any) => {
    if (!row) return;
    // 仅回填 defaults 声明的字段，不拷贝 id/createdAt 等不可变字段
    drawer.formData.typeName = row.typeName ?? "";
    drawer.formData.typeCode = row.typeCode ?? "";
    drawer.formData.sortOrder = row.sortOrder ?? 1;
    drawer.formData.isEnabled = row.isEnabled ?? true;
  });
}

// 暴露方法
defineExpose({ open });
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
