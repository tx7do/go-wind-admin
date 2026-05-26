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

      <ElFormItem :label="$t('pages.language.languageName')" prop="languageName">
        <ElInput
          v-model="formData.languageName"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.language.languageCode')" prop="languageCode">
        <ElInput
          v-model="formData.languageCode"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.language.nativeName')" prop="nativeName">
        <ElInput
          v-model="formData.nativeName"
          :placeholder="$t('common.placeholder.input')"
          clearable
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

      <ElFormItem :label="$t('pages.language.isEnabled')" prop="isEnabled">
        <ElSwitch v-model="formData.isEnabled" />
      </ElFormItem>

      <ElFormItem :label="$t('pages.language.isDefault')" prop="isDefault">
        <ElSwitch v-model="formData.isDefault" />
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

import { useCreateLanguage, useUpdateLanguage } from "@/api/composables";
import { $t } from "@/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createLanguage } = useCreateLanguage();
const { mutateAsync: updateLanguage } = useUpdateLanguage();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
  languageName: "",
  languageCode: "",
  nativeName: "",
  sortOrder: 1,
  isEnabled: true,
  isDefault: false,
});

// 表单验证规则
const formRules = {
  languageName: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  languageCode: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  nativeName: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

// 标题
const title = computed(() =>
  isCreate.value ? $t("pages.language.button.create") : $t("pages.language.button.update")
);

// 打开抽屉
function open(row?: any) {
  visible.value = true;

  if (row) {
    // 编辑模式
    isCreate.value = false;
    currentId.value = row.id;
    Object.assign(formData, row);
  } else {
    // 创建模式
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

// 重置表单
function resetForm() {
  formData.languageName = "";
  formData.languageCode = "";
  formData.nativeName = "";
  formData.sortOrder = 1;
  formData.isEnabled = true;
  formData.isDefault = false;

  formRef.value?.clearValidate();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const values = { ...formData };

    if (isCreate.value) {
      await createLanguage(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateLanguage({ id: currentId.value!, values });
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
</style>
