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
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('common.table.description')" prop="description">
        <ElInput
          v-model="drawer.formData.description"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.api.module')" prop="module">
        <ElInput
          v-model="drawer.formData.module"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.api.moduleDescription')" prop="moduleDescription">
        <ElInput
          v-model="drawer.formData.moduleDescription"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.api.path')" prop="path">
        <ElInput v-model="drawer.formData.path" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.api.method')" prop="method">
        <ElSelect
          v-model="drawer.formData.method"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          style="width: 100%"
        >
          <ElOption
            v-for="item in methodList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
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
import { methodList, useCreateApi, useUpdateApi } from "@/api/composables";
import { $t } from "@/core/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createApi } = useCreateApi();
const { mutateAsync: updateApi } = useUpdateApi();

const formRef = ref();

const drawer = useDrawerForm({
  moduleKey: "pages.api.moduleName",
  defaults: {
    description: "",
    module: "",
    moduleDescription: "",
    path: "",
    method: "",
  },
  createFn: createApi,
  updateFn: (id, values) => updateApi({ id, values }),
});

// 表单验证规则
const formRules = {
  path: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  method: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 暴露方法给父组件
defineExpose({ open: drawer.open });
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
</style>
