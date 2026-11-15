<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :close-guard="drawer.closeGuard"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
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
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.config.name')" prop="name">
        <ElInput
          v-model="drawer.formData.name"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.config.key')" prop="key">
        <ElInput
          v-model="drawer.formData.key"
          placeholder="e.g. sys.login.captchaEnabled"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.config.value')" prop="value">
        <ElInput
          v-model="drawer.formData.value"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.config.valueType')" prop="valueType">
        <ElSelect
          v-model="drawer.formData.valueType"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
        >
          <ElOption
            v-for="item in configValueTypeList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.config.isBuiltIn')" prop="isBuiltIn">
        <ElSwitch v-model="drawer.formData.isBuiltIn" />
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
import {
  ElButton,
  ElDivider,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElSelect,
  ElSwitch,
} from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";
import { configValueTypeList, useCreateConfig, useUpdateConfig } from "@/api/composables";
import { $t } from "@go-wind/admin-element";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createConfig } = useCreateConfig();
const { mutateAsync: updateConfig } = useUpdateConfig();

const formRef = ref();

const drawer = useDrawerForm({
  moduleKey: "pages.config.moduleName",
  defaults: {
    name: "",
    key: "",
    value: "",
    valueType: "STRING",
    isBuiltIn: false,
  },
  createFn: createConfig,
  updateFn: (id, values) => updateConfig({ id, values }),
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  key: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  value: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  valueType: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
};

// 包装 open：编辑模式下显式回填表单声明字段（避免 useDrawerForm 默认不回填导致编辑态空白）
function open(data?: { create?: boolean; row?: any }) {
  drawer.open(data, (row: any) => {
    if (!row) return;
    // 仅回填 defaults 声明的字段，不拷贝 id/createdAt 等不可变字段
    drawer.formData.name = row.name ?? "";
    drawer.formData.key = row.key ?? "";
    drawer.formData.value = row.value ?? "";
    drawer.formData.valueType = row.valueType ?? "STRING";
    drawer.formData.isBuiltIn = !!row.isBuiltIn;
  });
}

// 暴露方法给父组件
defineExpose({ open });
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
