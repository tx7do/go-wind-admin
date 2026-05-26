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
      <ElFormItem :label="$t('pages.dict.typeName')" prop="typeName">
        <ElInput
          v-model="formData.typeName"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.dict.typeCode')" prop="typeCode">
        <ElInput
          v-model="formData.typeCode"
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

      <ElFormItem :label="$t('common.table.status')" prop="isEnabled">
        <ElRadioGroup v-model="formData.isEnabled">
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

import { enableBoolList, useCreateDictType, useUpdateDictType } from "@/api/composables";
import { $t } from "@/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createDictType } = useCreateDictType();
const { mutateAsync: updateDictType } = useUpdateDictType();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
  typeName: "",
  typeCode: "",
  sortOrder: 1,
  isEnabled: true,
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

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.dict.dictType") })
    : $t("common.modal.update", { moduleName: $t("pages.dict.dictType") })
);

// 重置表单
function resetForm() {
  Object.assign(formData, {
    typeName: "",
    typeCode: "",
    sortOrder: 1,
    isEnabled: true,
  });
  formRef.value?.clearValidate();
}

// 打开抽屉
function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;

  // 重置表单
  resetForm();

  // 编辑时填充数据
  if (data?.row && !isCreate.value) {
    Object.assign(formData, {
      typeName: data.row.typeName || "",
      typeCode: data.row.typeCode || "",
      sortOrder: data.row.sortOrder || 1,
      isEnabled: data.row.isEnabled ?? true,
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
      await createDictType(formData);
      ElMessage.success($t("common.notification.create_success"));
    } else {
      await updateDictType({ id: currentId.value!, values: formData });
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
