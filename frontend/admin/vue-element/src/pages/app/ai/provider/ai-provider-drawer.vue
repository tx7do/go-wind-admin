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
      label-width="140px"
      class="drawer-form"
    >
      <ElFormItem :label="t('pages.ai_provider.name')" prop="name">
        <ElInput v-model="drawer.formData.name" :placeholder="t('pages.ai_provider.namePlaceholder')" clearable />
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_provider.modelType')" prop="modelType">
        <ElRadioGroup v-model="drawer.formData.modelType">
          <ElRadio value="CLOUD">{{ t("pages.ai_provider.modelTypeCloud") }}</ElRadio>
          <ElRadio value="LOCAL">{{ t("pages.ai_provider.modelTypeLocal") }}</ElRadio>
        </ElRadioGroup>
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_provider.modelName')" prop="modelName">
        <ElInput v-model="drawer.formData.modelName" :placeholder="t('pages.ai_provider.modelNamePlaceholder')" clearable />
      </ElFormItem>

      <!-- 云端 / 本地各自的连接配置，与 react 基准同构 -->
      <template v-if="drawer.formData.modelType === 'LOCAL'">
        <ElFormItem :label="t('pages.ai_provider.localHost')" prop="localHost">
          <ElInput v-model="drawer.formData.localHost" :placeholder="t('pages.ai_provider.localHostPlaceholder')" clearable />
        </ElFormItem>
        <ElFormItem :label="t('pages.ai_provider.localPort')" prop="localPort">
          <ElInputNumber v-model="drawer.formData.localPort" :min="1" :max="65535" :controls="false" class="!w-full" />
        </ElFormItem>
      </template>
      <template v-else>
        <ElFormItem :label="t('pages.ai_provider.baseUrl')" prop="baseUrl">
          <ElInput v-model="drawer.formData.baseUrl" :placeholder="t('pages.ai_provider.baseUrlPlaceholder')" clearable />
        </ElFormItem>
        <ElFormItem :label="t('pages.ai_provider.organization')" prop="organization">
          <ElInput v-model="drawer.formData.organization" clearable />
        </ElFormItem>
        <!-- 密钥请求级字段：明文只在本次请求内存在，后端加密落库、读取视图只有脱敏 hint -->
        <ElFormItem :label="t('pages.ai_provider.apiKey')" prop="apiKey">
          <ElInput
            v-model="drawer.formData.apiKey"
            type="password"
            show-password
            :placeholder="isCreate ? t('pages.ai_provider.apiKeyPlaceholder') : t('pages.ai_provider.apiKeyKeepExisting')"
          />
          <div v-if="!isCreate && currentHint" class="field-tip">
            {{ t("pages.ai_provider.apiKeyHintLabel") }}: {{ currentHint }}
          </div>
        </ElFormItem>
      </template>

      <ElFormItem :label="t('pages.ai_provider.timeoutSeconds')" prop="timeoutSeconds">
        <ElInputNumber v-model="drawer.formData.timeoutSeconds" :min="1" :max="600" :controls="false" class="!w-full" />
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_provider.systemPrompt')" prop="systemPrompt">
        <ElInput
          v-model="drawer.formData.systemPrompt"
          type="textarea"
          :rows="3"
          :placeholder="t('pages.ai_provider.systemPromptPlaceholder')"
        />
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_provider.isDefault')" prop="isDefault">
        <ElSwitch v-model="drawer.formData.isDefault" />
        <div class="field-tip">{{ t("pages.ai_provider.isDefaultTooltip") }}</div>
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_provider.isEnabled')" prop="isEnabled">
        <ElSwitch v-model="drawer.formData.isEnabled" />
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_provider.remark')" prop="remark">
        <ElInput v-model="drawer.formData.remark" type="textarea" :rows="2" />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <ElButton @click="drawer.guard.requestClose(drawer.close)">{{ $t("common.button.cancel") }}</ElButton>
      <ElButton
        type="primary"
        :loading="drawer.submitLoading.value"
        @click="drawer.handleSubmit(formRef, () => emit('success'))"
      >
        {{ $t("common.button.confirm") }}
      </ElButton>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElRadio,
  ElRadioGroup,
  ElSwitch,
  type FormInstance,
  type FormRules,
} from "element-plus";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";
import { useI18n } from "@/core/i18n";
import { createAiProvider, updateAiProvider } from "@/api/composables";

const emit = defineEmits<{ success: [] }>();
const { t } = useI18n();
const formRef = ref<FormInstance>();

const currentHint = ref("");

const isCreate = computed(() => drawer.isCreate.value);

const drawer = useDrawerForm({
  moduleKey: "pages.ai_provider.moduleName",
  defaults: {
    name: "",
    modelType: "CLOUD",
    modelName: "",
    baseUrl: "",
    organization: "",
    apiKey: "",
    localHost: "",
    localPort: 11434,
    timeoutSeconds: 60,
    systemPrompt: "",
    isDefault: false,
    isEnabled: true,
    remark: "",
  },
  createFn: async (values: Record<string, any>) => {
    const { apiKeyHint: _hint, ...data } = values;
    return createAiProvider(data);
  },
  updateFn: (id: string, values: Record<string, any>) => {
    const { apiKeyHint: _hint, ...data } = values;
    return updateAiProvider(id, data);
  },
});

const formRules: FormRules = {
  name: [{ required: true, message: t("pages.ai_provider.requiredName"), trigger: "blur" }],
  modelName: [{ required: true, message: t("pages.ai_provider.requiredModelName"), trigger: "blur" }],
  baseUrl: [
    {
      validator: (_rule: any, value: string, callback: (err?: Error) => void) => {
        if (drawer.formData.modelType === "CLOUD" && !value) {
          callback(new Error(t("pages.ai_provider.requiredBaseUrl")));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
};

defineExpose({
  open: (opts: { create: boolean; row?: any }) => {
    currentHint.value = opts.row?.apiKeyHint || "";
    // 编辑模式下显式回填表单声明字段（避免 useDrawerForm 默认不回填导致编辑态空白）
    drawer.open(opts, (row: any) => {
      if (!row) return;
      // 仅回填 defaults 声明的字段；apiKey 为机密不回显，保持空值
      drawer.formData.name = row.name ?? "";
      drawer.formData.modelType = row.modelType ?? "CLOUD";
      drawer.formData.modelName = row.modelName ?? "";
      drawer.formData.baseUrl = row.baseUrl ?? "";
      drawer.formData.organization = row.organization ?? "";
      drawer.formData.localHost = row.localHost ?? "";
      drawer.formData.localPort = row.localPort ?? 11434;
      drawer.formData.timeoutSeconds = row.timeoutSeconds ?? 60;
      drawer.formData.systemPrompt = row.systemPrompt ?? "";
      drawer.formData.isDefault = !!row.isDefault;
      drawer.formData.isEnabled = row.isEnabled ?? true;
      drawer.formData.remark = row.remark ?? "";
    });
  },
});
</script>

<style lang="scss" scoped>
.field-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  margin-top: 4px;
  width: 100%;
}
</style>
