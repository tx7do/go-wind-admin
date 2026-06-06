<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :title="drawer.title.value"
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
import { enableBoolList, useCreateDictType, useUpdateDictType } from "@/api/composables";
import { $t } from "@/core/i18n";

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
