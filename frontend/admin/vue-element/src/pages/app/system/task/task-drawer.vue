<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH } }"
    :close-guard="guard.allowClose"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="140px"
      class="drawer-form"
    >
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.task.type')" prop="type">
        <ElSelect
          v-model="formData.type"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          style="width: 100%"
        >
          <ElOption
            v-for="item in taskTypeList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.task.typeName')" prop="typeName">
        <ElSelect
          v-model="formData.typeName"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          style="width: 100%"
        >
          <ElOption
            v-for="item in typeNameOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.task.taskPayload')" prop="taskPayload">
        <ElInput
          v-model="formData.taskPayload"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem
        v-if="formData.type === 'PERIODIC'"
        :label="$t('pages.task.cronSpec')"
        prop="cronSpec"
      >
        <ElInput
          v-model="formData.cronSpec"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.task.taskOptionsMaxRetry')" prop="taskOptions.maxRetry">
        <ElInputNumber
          v-model="formData.taskOptions.maxRetry"
          :min="0"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem
        v-if="['DELAY', 'WAIT_RESULT'].includes(formData.type)"
        :label="$t('pages.task.taskOptionsTimeout')"
        prop="taskOptions.timeout"
      >
        <ElInputNumber
          v-model="formData.taskOptions.timeout"
          :min="0"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem
        v-if="formData.type === 'DELAY'"
        :label="$t('pages.task.taskOptionsDeadline')"
        prop="taskOptions.deadline"
      >
        <ElDatePicker
          v-model="formData.taskOptions.deadline"
          type="datetime"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem
        v-if="formData.type === 'DELAY'"
        :label="$t('pages.task.taskOptionsProcessIn')"
        prop="taskOptions.processIn"
      >
        <ElDatePicker
          v-model="formData.taskOptions.processIn"
          type="datetime"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem
        v-if="formData.type === 'DELAY'"
        :label="$t('pages.task.taskOptionsProcessAt')"
        prop="taskOptions.processAt"
      >
        <ElDatePicker
          v-model="formData.taskOptions.processAt"
          type="datetime"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.task.enable')" prop="enable">
        <ElRadioGroup v-model="formData.enable">
          <ElRadioButton :value="true">{{ $t("enum.enable.true") }}</ElRadioButton>
          <ElRadioButton :value="false">{{ $t("enum.enable.false") }}</ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <ElFormItem :label="$t('common.table.remark')" prop="remark">
        <ElInput
          v-model="formData.remark"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="guard.requestClose(handleClose)">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watch } from "vue";
import { ElMessage } from "element-plus";

import {
  taskTypeList,
  useCreateTask,
  useUpdateTask,
  fetchListTaskTypeNames,
} from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerCloseGuard } from "@/components/Pro/composables/use-drawer-close-guard";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createTask } = useCreateTask();
const { mutateAsync: updateTask } = useUpdateTask();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<string>();
const formRef = ref();
const typeNameOptions = ref<Array<{ label: string; value: string }>>([]);

// 表单数据
const formData = reactive({
  type: "PERIODIC",
  typeName: "",
  taskPayload: "",
  cronSpec: "",
  taskOptions: {
    maxRetry: 3,
    timeout: undefined as number | undefined,
    deadline: undefined as Date | undefined,
    processIn: undefined as Date | undefined,
    processAt: undefined as Date | undefined,
  },
  enable: true,
  remark: "",
});

// 表单验证规则
const formRules = {
  type: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  typeName: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  enable: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 标题
const title = computed(() =>
  isCreate.value ? $t("pages.task.button.create") : $t("pages.task.button.update")
);

// 加载任务类型名称选项
async function loadTypeNameOptions() {
  try {
    const result = await fetchListTaskTypeNames();
    typeNameOptions.value = (result.typeNames ?? []).map((item: any) => ({
      label: item,
      value: item,
    }));
  } catch (error) {
    console.error("加载任务类型名称失败", error);
  }
}

// 打开抽屉
// 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
const guard = useDrawerCloseGuard(() => ({ ...formData }));
async function open(row?: any) {
  visible.value = true;

  // 加载任务类型名称选项
  await loadTypeNameOptions();

  if (row) {
    // 编辑模式
    isCreate.value = false;
    currentId.value = row.id;
    // 仅回填表单声明的字段，避免把 id/createdAt 等不可变字段灌入 formData
    formData.type = row.type ?? "PERIODIC";
    formData.typeName = row.typeName ?? "";
    formData.taskPayload = row.taskPayload ?? "";
    formData.cronSpec = row.cronSpec ?? "";
    formData.enable = row.enable ?? true;
    formData.remark = row.remark ?? "";
    // 仅回填 taskOptions 声明的子字段
    if (row.taskOptions) {
      formData.taskOptions.maxRetry = row.taskOptions.maxRetry ?? 3;
      formData.taskOptions.timeout = row.taskOptions.timeout;
      formData.taskOptions.deadline = row.taskOptions.deadline;
      formData.taskOptions.processIn = row.taskOptions.processIn;
      formData.taskOptions.processAt = row.taskOptions.processAt;
    }
  } else {
    // 创建模式
    isCreate.value = true;
    currentId.value = undefined;
    resetForm();
  }

  // 填充完成后记录基线快照（回填不算脏）
  guard.syncBaseline();
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 重置表单
function resetForm() {
  formData.type = "PERIODIC";
  formData.typeName = "";
  formData.taskPayload = "";
  formData.cronSpec = "";
  formData.taskOptions = {
    maxRetry: 3,
    timeout: undefined,
    deadline: undefined,
    processIn: undefined,
    processAt: undefined,
  };
  formData.enable = true;
  formData.remark = "";

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
      await createTask(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateTask({ id: currentId.value!, values });
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

// ProModal 关闭时自动重置表单
watch(visible, (val) => {
  if (!val) resetForm();
});

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
