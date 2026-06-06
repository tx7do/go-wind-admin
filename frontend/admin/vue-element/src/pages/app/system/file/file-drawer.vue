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

      <ElFormItem :label="$t('pages.file.fileName')" prop="fileName">
        <ElInput
          v-model="drawer.formData.fileName"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.remark')" prop="remark">
        <ElInput
          v-model="drawer.formData.remark"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
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
import { useCreateFile, useUpdateFile } from "@/api/composables";
import { $t } from "@/core/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createFile } = useCreateFile();
const { mutateAsync: updateFile } = useUpdateFile();

const formRef = ref();

const drawer = useDrawerForm({
  moduleKey: "pages.file.moduleName",
  defaults: {
    fileName: "",
    remark: "",
  },
  createFn: createFile,
  updateFn: (id, values) => updateFile({ id, values }),
});

// 表单验证规则
const formRules = {
  fileName: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
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
