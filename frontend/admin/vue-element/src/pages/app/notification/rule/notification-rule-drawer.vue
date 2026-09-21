<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <ElFormItem :label="t('pages.notification_rule.eventType')" prop="eventType">
        <!-- 编辑态 eventType 不可改：改事件类型等于换一条路由，删旧建新才说得清 -->
        <ElSelect
          v-model="formData.eventType"
          :disabled="!isCreate"
          :placeholder="t('pages.notification_rule.requiredEventType')"
          filterable
          style="width: 100%"
        >
          <ElOption
            v-for="item in notificationRuleEventTypeList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_rule.channel')" prop="channel">
        <ElSelect
          v-model="formData.channel"
          :placeholder="t('pages.notification_rule.requiredChannel')"
          style="width: 100%"
        >
          <ElOption
            v-for="item in notificationRuleChannelOptionList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem prop="isAsync">
        <template #label>
          <span class="label-with-tip">
            {{ t("pages.notification_rule.isAsync") }}
            <ElTooltip :content="t('pages.notification_rule.isAsyncHint')" placement="top">
              <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
            </ElTooltip>
          </span>
        </template>
        <ElSwitch
          v-model="formData.isAsync"
          :active-text="t('pages.notification_rule.isAsyncOn')"
          :inactive-text="t('pages.notification_rule.isAsyncOff')"
        />
      </ElFormItem>

      <ElFormItem prop="isEnabled">
        <template #label>
          <span class="label-with-tip">
            {{ t("pages.notification_rule.isEnabled") }}
            <ElTooltip :content="t('pages.notification_rule.isEnabledHint')" placement="top">
              <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
            </ElTooltip>
          </span>
        </template>
        <ElSwitch v-model="formData.isEnabled" />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_rule.remark')" prop="remark">
        <ElInput v-model="formData.remark" type="textarea" :rows="2" />
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
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElMessage,
  ElOption,
  ElSelect,
  ElSwitch,
  ElTooltip,
} from "element-plus";
import { QuestionFilled } from "@element-plus/icons-vue";

import ProModal from "@/components/Pro/ProModal/index.vue";
import type {
  notificationservicev1_Channel,
  notificationservicev1_EventType,
  notificationservicev1_NotificationRule,
} from "@/api/generated/admin/service/v1";
import { useI18n } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import {
  notificationRuleChannelOptionList,
  notificationRuleEventTypeList,
  useCreateNotificationRule,
  useUpdateNotificationRule,
} from "@/api/composables";

const emit = defineEmits<{
  success: [];
}>();

const { t } = useI18n();

const { mutateAsync: createNotificationRule } = useCreateNotificationRule();
const { mutateAsync: updateNotificationRule } = useUpdateNotificationRule();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

const formData = reactive({
  eventType: "" as notificationservicev1_EventType | "",
  channel: "EMAIL" as notificationservicev1_Channel,
  isAsync: true,
  isEnabled: true,
  remark: "",
});

const formRules = {
  eventType: [
    { required: true, message: t("pages.notification_rule.requiredEventType"), trigger: "change" },
  ],
  channel: [
    { required: true, message: t("pages.notification_rule.requiredChannel"), trigger: "change" },
  ],
};

const title = computed(() =>
  isCreate.value ? t("pages.notification_rule.create") : t("pages.notification_rule.edit")
);

function resetForm() {
  formData.eventType = "";
  formData.channel = "EMAIL";
  formData.isAsync = true;
  formData.isEnabled = true;
  formData.remark = "";
  formRef.value?.clearValidate();
}

function open(options: { create: boolean; row?: notificationservicev1_NotificationRule }) {
  visible.value = true;
  isCreate.value = options.create;
  currentId.value = options.row?.id;
  resetForm();

  if (!options.create && options.row) {
    // 只回填表单声明的字段：id/createdAt/updatedAt 是服务端字段，灌进 formData 会被当作 data 发出去
    formData.eventType = options.row.eventType ?? "";
    formData.channel = options.row.channel ?? "EMAIL";
    formData.isAsync = !!options.row.isAsync;
    formData.isEnabled = !!options.row.isEnabled;
    formData.remark = options.row.remark ?? "";
  }
}

function handleClose() {
  visible.value = false;
  resetForm();
}

async function handleSubmit() {
  if (!formRef.value) return;

  // 二段式校验：validate 失败时 reject 的是字段错误对象，先单独校验、失败即返回，
  // 免得把「必填项为空」误报成「保存失败」
  const valid = await formRef.value.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  // 只发可写字段：eventType 由服务端按 id 定位，编辑态它不可改（掩码里也没有）
  const data: notificationservicev1_NotificationRule = {
    eventType: formData.eventType || undefined,
    channel: formData.channel,
    isAsync: !!formData.isAsync,
    isEnabled: !!formData.isEnabled,
    remark: formData.remark,
  };

  try {
    submitLoading.value = true;
    if (isCreate.value) {
      await createNotificationRule({ data });
      ElMessage.success(t("pages.notification_rule.createSuccess"));
    } else if (currentId.value !== undefined) {
      await updateNotificationRule({ id: currentId.value, data });
      ElMessage.success(t("pages.notification_rule.updateSuccess"));
    }
    emit("success");
    handleClose();
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error("save notification rule failed", error);
    ElMessage.error(error?.message || t("pages.notification_rule.saveFailed"));
  } finally {
    submitLoading.value = false;
  }
}

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

.label-with-tip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.label-tip-icon {
  color: var(--el-text-color-secondary);
  cursor: help;
}
</style>
