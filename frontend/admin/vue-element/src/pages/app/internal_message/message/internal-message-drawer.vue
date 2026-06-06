<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: '800px', closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <ElRow :gutter="20">
        <ElCol :span="12">
          <ElFormItem :label="$t('pages.internal_message.status')" prop="status">
            <ElSelect
              v-model="formData.status"
              :placeholder="$t('common.placeholder.select')"
              style="width: 100%"
            >
              <ElOption
                v-for="item in statusOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem :label="$t('pages.internal_message.type')" prop="type">
            <ElSelect
              v-model="formData.type"
              :placeholder="$t('common.placeholder.select')"
              style="width: 100%"
            >
              <ElOption
                v-for="item in typeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElFormItem :label="$t('pages.internal_message.categoryId')" prop="categoryId">
        <ElTreeSelect
          v-model="formData.categoryId"
          :data="categoryTreeData"
          :props="{ label: 'name', children: 'children' }"
          :placeholder="$t('common.placeholder.select')"
          node-key="id"
          check-strictly
          default-expand-all
          filterable
          clearable
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.internal_message.title')" prop="title">
        <ElInput v-model="formData.title" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.internal_message.content')" prop="content">
        <Editor v-model="formData.content" :editor-type="EditorType.RICH_TEXT" :height="350" />
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
  </ProModal>
</template>

<script lang="ts" setup>
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { ref, reactive, computed, watch } from "vue";

import { Editor, EditorType } from "@/components/Editor";

import {
  internalMessageStatusList,
  internalMessageTypeList,
  fetchListMessageCategories,
  useSendMessage,
  useUpdateInternalMessage,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
import ProModal from "@/components/Pro/ProModal/index.vue";

const emit = defineEmits(["success"]);

const { mutateAsync: sendMessage } = useSendMessage();
const { mutateAsync: updateMessage } = useUpdateInternalMessage();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();
const isCreate = ref(true);
const currentId = ref<number | undefined>();

// 分类树数据
const categoryTreeData = ref<any[]>([]);

// 表单数据
const formData = reactive({
  status: "DRAFT",
  type: "NOTIFICATION",
  categoryId: undefined as number | undefined,
  title: "",
  content: "",
});

// 表单验证规则
const formRules: FormRules = {
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  type: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  categoryId: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
  title: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

// 状态选项
const statusOptions = computed(() => internalMessageStatusList.value);

// 类型选项
const typeOptions = computed(() => internalMessageTypeList.value);

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("pages.internal_message.drawer.create")
    : $t("pages.internal_message.drawer.update")
);

// 加载分类树
async function loadCategoryTree() {
  try {
    const result = await fetchListMessageCategories(
      new PaginationQuery({ formValues: { is_enabled: "true" } })
    );
    categoryTreeData.value = result.items || [];
  } catch (error) {
    console.error("加载分类树失败", error);
  }
}

// 重置表单
function resetForm() {
  formData.status = "DRAFT";
  formData.type = "NOTIFICATION";
  formData.categoryId = undefined;
  formData.title = "";
  formData.content = "";
  formRef.value?.clearValidate();
}

// 打开抽屉
async function open(row?: any) {
  visible.value = true;
  await loadCategoryTree();

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
      await sendMessage({
        targetUserIds: undefined,
        ...formData,
        targetAll: true,
      } as any);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateMessage({ id: currentId.value!, values: formData });
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

// ProModal 关闭时自动重置表单
watch(visible, (val) => {
  if (!val) resetForm();
});

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
