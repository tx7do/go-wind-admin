<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    size="600px"
    append-to-body
    destroy-on-close
    @close="handleClose"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <ElFormItem :label="$t('pages.internal_message_category.name')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.internal_message_category.code')" prop="code">
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.sortOrder')" prop="sortOrder">
        <ElInputNumber
          v-model="formData.sortOrder"
          :min="1"
          :max="9999"
          controls-position="right"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="isEnabled">
        <ElRadioGroup v-model="formData.isEnabled">
          <ElRadio :value="true">{{ $t("common.switch.active") }}</ElRadio>
          <ElRadio :value="false">{{ $t("common.switch.inactive") }}</ElRadio>
        </ElRadioGroup>
      </ElFormItem>

      <ElFormItem :label="$t('common.table.remark')" prop="remark">
        <ElInput
          v-model="formData.remark"
          type="textarea"
          :placeholder="$t('common.placeholder.input')"
          :rows="4"
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="loading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { ref, reactive, computed } from "vue";

import { useCreateMessageCategory, useUpdateMessageCategory } from "@/api/composables";
import { $t } from "@/i18n";

const emit = defineEmits(["success"]);

const { mutateAsync: createMessageCategory } = useCreateMessageCategory();
const { mutateAsync: updateMessageCategory } = useUpdateMessageCategory();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();
const isCreate = ref(true);
const currentId = ref<number | undefined>();

// 表单数据
const formData = reactive({
  name: "",
  code: "",
  sortOrder: 1,
  isEnabled: true,
  remark: "",
});

// 表单验证规则
const formRules: FormRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  isEnabled: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.internal_message_category.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.internal_message_category.moduleName") })
);

// 重置表单
function resetForm() {
  formData.name = "";
  formData.code = "";
  formData.sortOrder = 1;
  formData.isEnabled = true;
  formData.remark = "";
  formRef.value?.clearValidate();
}

// 打开抽屉
function open(row?: any) {
  visible.value = true;

  if (row) {
    isCreate.value = false;
    currentId.value = row.id;
    Object.assign(formData, row);
  } else {
    isCreate.value = true;
    currentId.value = undefined;
    resetForm();
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
    loading.value = true;

    if (isCreate.value) {
      await createMessageCategory({ data: formData });
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateMessageCategory({ id: currentId.value!, values: formData });
      ElMessage.success($t("common.notification.updateSuccess"));
    }

    emit("success");
    handleClose();
  } catch (error) {
    if (error !== false) {
      // 非表单验证错误
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.createFailed")
          : $t("common.notification.updateFailed")
      );
    }
  } finally {
    loading.value = false;
  }
}

// 暴露方法
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
