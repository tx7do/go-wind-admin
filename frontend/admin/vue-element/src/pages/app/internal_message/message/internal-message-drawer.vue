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

      <!-- 发送范围与接收用户：仅创建模式 -->
      <template v-if="isCreate">
        <ElFormItem :label="$t('pages.internal_message.sendScope')" prop="sendScope">
          <ElRadioGroup v-model="formData.sendScope">
            <ElRadio value="ALL">{{ $t("pages.internal_message.sendScopeAll") }}</ElRadio>
            <ElRadio value="USERS">{{ $t("pages.internal_message.sendScopeUsers") }}</ElRadio>
          </ElRadioGroup>
        </ElFormItem>

        <ElFormItem
          v-if="formData.sendScope === 'USERS'"
          :label="$t('pages.internal_message.targetUsers')"
          prop="targetUserIds"
        >
          <ElSelect
            v-model="formData.targetUserIds"
            multiple
            filterable
            clearable
            :placeholder="$t('common.placeholder.select')"
            style="width: 100%"
          >
            <ElOption
              v-for="u in userOptions"
              :key="u.value"
              :label="u.label"
              :value="u.value"
            />
          </ElSelect>
        </ElFormItem>
      </template>

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
  fetchListUsers,
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

// 接收用户选项（切换到指定用户时懒加载）
const userOptions = ref<{ label: string; value: number }[]>([]);

// 表单数据
const formData = reactive({
  status: "DRAFT",
  type: "NOTIFICATION",
  categoryId: undefined as number | undefined,
  title: "",
  content: "",
  sendScope: "ALL" as "ALL" | "USERS",
  targetUserIds: [] as number[],
});

// 表单验证规则
const formRules: FormRules = {
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  type: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  categoryId: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
  title: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  targetUserIds: [
    {
      validator: (_rule, value: number[], callback) => {
        if (formData.sendScope === "USERS" && (!value || value.length === 0)) {
          callback(new Error($t("pages.internal_message.requiredTargetUsers")));
        } else {
          callback();
        }
      },
      trigger: "change",
    },
  ],
};

// 切换到指定用户时按需拉取用户选项（fetchListUsers 不传分页即全量，管理员规模可控）
async function loadUserOptions() {
  if (userOptions.value.length > 0) return;
  try {
    const result = await fetchListUsers(new PaginationQuery({}));
    userOptions.value = (result.items || []).map((u: any) => ({
      label: u.nickname ? `${u.nickname}(${u.username})` : String(u.username),
      value: Number(u.id),
    }));
  } catch (error) {
    console.error("加载用户列表失败", error);
  }
}

watch(
  () => formData.sendScope,
  (val) => {
    if (val === "USERS") loadUserOptions();
  },
);

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
  formData.sendScope = "ALL";
  formData.targetUserIds = [];
  formRef.value?.clearValidate();
}

// 打开抽屉
async function open(row?: any) {
  visible.value = true;
  await loadCategoryTree();

  if (row) {
    isCreate.value = false;
    currentId.value = row.id;
    // 仅回填表单声明的字段，避免把 id/createdAt/tenantId 等不可变字段灌入 formData
    formData.status = row.status ?? "DRAFT";
    formData.type = row.type ?? "NOTIFICATION";
    formData.categoryId = row.categoryId;
    formData.title = row.title ?? "";
    formData.content = row.content ?? "";
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

  // 校验失败时 validate 会 reject 字段错误对象（非 false），用二段式区分校验失败与接口失败
  const valid = await formRef.value.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  try {
    loading.value = true;

    if (isCreate.value) {
      // sendScope/targetUserIds 是前端选择发送范围的控制字段，按范围决定提交形态
      const { sendScope, targetUserIds, ...rest } = formData;
      await sendMessage(
        (sendScope === "USERS"
          ? { ...rest, targetUserIds }
          : { ...rest, targetAll: true }) as any
      );
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      const { sendScope: _scope, targetUserIds: _uids, ...editValues } = formData;
      await updateMessage({ id: currentId.value!, values: editValues });
      ElMessage.success($t("common.notification.updateSuccess"));
    }

    emit("success");
    handleClose();
  } catch {
    // 仅接口失败会进入此分支（校验失败已在上方提前 return）
    ElMessage.error(
      isCreate.value
        ? $t("common.notification.createFailed")
        : $t("common.notification.updateFailed")
    );
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
