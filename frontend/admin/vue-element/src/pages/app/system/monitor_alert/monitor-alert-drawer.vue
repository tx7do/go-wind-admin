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
      label-width="120px"
      class="drawer-form"
    >
      <ElFormItem :label="t('pages.monitor_alert.name')" prop="name">
        <ElInput
          v-model="formData.name"
          :placeholder="t('common.placeholder.input')"
          maxlength="100"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="t('pages.monitor_alert.metric')" prop="metric">
        <ElSelect
          v-model="formData.metric"
          :placeholder="t('common.placeholder.select')"
          filterable
          style="width: 100%"
          @change="handleMetricChange"
        >
          <ElOption
            v-for="item in metricOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <!-- DB_PING_FAIL 是布尔指标：ping 失败即触发，op/threshold 无意义 -->
      <template v-if="formData.metric !== 'DB_PING_FAIL'">
        <ElFormItem :label="t('pages.monitor_alert.op')" prop="op">
          <ElSelect v-model="formData.op" style="width: 100%">
            <ElOption
              v-for="item in opOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem :label="t('pages.monitor_alert.threshold')" prop="threshold">
          <ElInputNumber
            v-model="formData.threshold"
            :precision="2"
            :step="1"
            style="width: 100%"
          />
        </ElFormItem>
      </template>

      <ElFormItem prop="cooldownMinutes">
        <template #label>
          <span class="label-with-tip">
            {{ t("pages.monitor_alert.cooldownMinutes") }}
            <ElTooltip :content="t('pages.monitor_alert.cooldownHint')" placement="top">
              <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
            </ElTooltip>
          </span>
        </template>
        <ElInputNumber
          v-model="formData.cooldownMinutes"
          :min="1"
          :max="1440"
          :precision="0"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="t('pages.monitor_alert.channel')" prop="channel">
        <ElSelect v-model="formData.channel" style="width: 100%">
          <ElOption
            v-for="item in channelOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem prop="target">
        <template #label>
          <span class="label-with-tip">
            {{ t("pages.monitor_alert.target") }}
            <ElTooltip :content="t('pages.monitor_alert.targetHint')" placement="top">
              <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
            </ElTooltip>
          </span>
        </template>
        <ElInput
          v-model="formData.target"
          :placeholder="t('common.placeholder.input')"
          maxlength="500"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="t('pages.monitor_alert.isEnabled')">
        <ElSwitch v-model="formData.isEnabled" />
      </ElFormItem>

      <ElFormItem :label="t('pages.monitor_alert.remark')" prop="remark">
        <ElInput
          v-model="formData.remark"
          type="textarea"
          :rows="2"
          maxlength="500"
          :placeholder="t('common.placeholder.input')"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="guard.requestClose(handleClose)">{{ $t("common.button.cancel") }}</ElButton>
      <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
        {{ $t("common.button.confirm") }}
      </ElButton>
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
  ElInputNumber,
  ElMessage,
  ElOption,
  ElSelect,
  ElSwitch,
  ElTooltip,
} from "element-plus";
import { QuestionFilled } from "@element-plus/icons-vue";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerCloseGuard } from "@/components/Pro/composables/use-drawer-close-guard";
import type {
  monitor_alertservicev1_MonitorAlertRule,
  monitor_alertservicev1_MonitorMetric,
} from "@/api/generated/admin/service/v1";
import { useI18n } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import {
  useCreateMonitorAlertRule,
  useUpdateMonitorAlertRule,
} from "@/api/composables";

const emit = defineEmits<{
  success: [];
}>();

const { t } = useI18n();

const { mutateAsync: createRule } = useCreateMonitorAlertRule();
const { mutateAsync: updateRule } = useUpdateMonitorAlertRule();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<string>();
const formRef = ref();

const formData = reactive({
  name: "",
  metric: "" as monitor_alertservicev1_MonitorMetric | "",
  op: "GE",
  threshold: undefined as number | undefined,
  cooldownMinutes: 30,
  channel: "EMAIL",
  target: "",
  isEnabled: true,
  remark: "",
});

const metricOptions = computed(() =>
  (["GO_GOROUTINES", "GO_MEM_ALLOC_MB", "DB_OPEN_CONNECTIONS", "DB_PING_FAIL", "REDIS_DB_SIZE"] as const).map(
    (value) => ({ value, label: t(`pages.monitor_alert.metricMap.${value}`) })
  )
);

const opOptions = computed(() =>
  (["GE", "LE"] as const).map((value) => ({ value, label: t(`pages.monitor_alert.opMap.${value}`) }))
);

const channelOptions = computed(() =>
  (["EMAIL", "WEBHOOK"] as const).map((value) => ({ value, label: t(`pages.monitor_alert.channelMap.${value}`) }))
);

const formRules = computed(() => ({
  name: [{ required: true, message: t("pages.monitor_alert.requiredName"), trigger: "blur" }],
  metric: [{ required: true, message: t("pages.monitor_alert.requiredMetric"), trigger: "change" }],
  op: [{ required: true, message: t("pages.monitor_alert.requiredOp"), trigger: "change" }],
  threshold: [{ required: true, message: t("pages.monitor_alert.requiredThreshold"), trigger: "blur" }],
  channel: [{ required: true, message: t("pages.monitor_alert.requiredChannel"), trigger: "change" }],
  target: [{ required: true, message: t("pages.monitor_alert.requiredTarget"), trigger: "blur" }],
}));

const title = computed(() =>
  isCreate.value ? t("pages.monitor_alert.create") : t("pages.monitor_alert.edit")
);

function handleMetricChange() {
  // 切到布尔指标时清掉无意义的比较条件，避免把残留值提交给服务端
  if (formData.metric === "DB_PING_FAIL") {
    formData.op = "GE";
    formData.threshold = undefined;
  }
  formRef.value?.clearValidate(["op", "threshold"]);
}

function resetForm() {
  formData.name = "";
  formData.metric = "";
  formData.op = "GE";
  formData.threshold = undefined;
  formData.cooldownMinutes = 30;
  formData.channel = "EMAIL";
  formData.target = "";
  formData.isEnabled = true;
  formData.remark = "";
  formRef.value?.clearValidate();
}

// 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
const guard = useDrawerCloseGuard(() => ({ ...formData }));
function open(options: { create: boolean; row?: monitor_alertservicev1_MonitorAlertRule }) {
  visible.value = true;
  isCreate.value = options.create;
  currentId.value = options.row?.id;
  resetForm();

  if (!options.create && options.row) {
    formData.name = options.row.name ?? "";
    formData.metric = options.row.metric ?? "";
    formData.op = options.row.op ?? "GE";
    formData.threshold = options.row.threshold;
    formData.cooldownMinutes = options.row.cooldownMinutes ?? 30;
    formData.channel = options.row.channel ?? "EMAIL";
    formData.target = options.row.target ?? "";
    formData.isEnabled = !!options.row.isEnabled;
    formData.remark = options.row.remark ?? "";
  }

  // 填充完成后记录基线快照（回填不算脏）
  guard.syncBaseline();
}

function handleClose() {
  visible.value = false;
  resetForm();
}

async function handleSubmit() {
  if (!formRef.value) return;

  const valid = await formRef.value.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  const isPingFail = formData.metric === "DB_PING_FAIL";
  const data: monitor_alertservicev1_MonitorAlertRule = {
    name: formData.name,
    metric: (formData.metric || undefined) as monitor_alertservicev1_MonitorAlertRule["metric"],
    // DB_PING_FAIL 是布尔指标，op/threshold 不给（服务端忽略）
    op: isPingFail ? undefined : (formData.op as monitor_alertservicev1_MonitorAlertRule["op"]),
    threshold: isPingFail ? undefined : formData.threshold,
    cooldownMinutes: formData.cooldownMinutes,
    channel: formData.channel as monitor_alertservicev1_MonitorAlertRule["channel"],
    target: formData.target,
    isEnabled: !!formData.isEnabled,
    remark: formData.remark,
  };

  try {
    submitLoading.value = true;
    if (isCreate.value) {
      await createRule({ data });
      ElMessage.success(t("pages.monitor_alert.createSuccess"));
    } else if (currentId.value !== undefined) {
      await updateRule({
        id: currentId.value,
        data,
        updateMask: "name,metric,op,threshold,cooldownMinutes,channel,target,isEnabled,remark",
      });
      ElMessage.success(t("pages.monitor_alert.updateSuccess"));
    }
    emit("success");
    handleClose();
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error("save monitor alert rule failed", error);
    ElMessage.error(error?.message || t("pages.monitor_alert.saveFailed"));
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
