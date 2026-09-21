<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
    :config="{
      component: 'drawer',
      drawer: { size: drawer.drawerWidth, closeOnClickModal: false },
    }"
  >
    <ElForm
      ref="formRef"
      :model="drawer.formData"
      :rules="formRules"
      label-width="140px"
      class="drawer-form"
    >
      <ElFormItem :label="t('pages.notification_channel.name')" prop="name">
        <ElInput v-model="drawer.formData.name" :placeholder="t('pages.notification_channel.requiredName')" clearable />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_channel.type')" prop="type">
        <ElSelect v-model="drawer.formData.type" :disabled="!isCreate">
          <ElOption :label="t('pages.notification_channel.typeEmail')" value="EMAIL" />
          <ElOption :label="t('pages.notification_channel.typeWebhook')" value="WEBHOOK" />
        </ElSelect>
      </ElFormItem>

      <!-- 两类渠道各自的配置项分组显示：表单里同时摆两组，管理员要自己猜哪一组生效。
           编辑态 type 不可改（改类型等于换一条渠道实现），所以两组不会在同一个会话里来回切。 -->
      <template v-if="drawer.formData.type === 'WEBHOOK'">
        <ElFormItem :label="t('pages.notification_channel.webhookUrl')" prop="webhookUrl">
          <ElInput
            v-model="drawer.formData.webhookUrl"
            placeholder="https://example.com/hooks/notification"
            clearable
          />
          <div class="field-tip">{{ t("pages.notification_channel.webhookUrlHint") }}</div>
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.webhookSecret')" prop="webhookSecret">
          <ElInput
            v-model="drawer.formData.webhookSecret"
            type="password"
            show-password
            :placeholder="
              isCreate
                ? t('pages.notification_channel.webhookSecretPlaceholder')
                : t('pages.notification_channel.webhookSecretKeepHint')
            "
          />
          <div class="field-tip">{{ t("pages.notification_channel.webhookSecretHint") }}</div>
        </ElFormItem>
      </template>

      <template v-else>
        <ElFormItem :label="t('pages.notification_channel.smtpHost')" prop="smtpHost">
          <ElInput v-model="drawer.formData.smtpHost" placeholder="smtp.example.com" clearable />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smtpPort')" prop="smtpPort">
          <ElInputNumber
            v-model="drawer.formData.smtpPort"
            :min="1"
            :max="65535"
            :precision="0"
            controls-position="right"
          />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smtpUsername')" prop="smtpUsername">
          <ElInput v-model="drawer.formData.smtpUsername" clearable />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.password')" prop="password">
          <ElInput
            v-model="drawer.formData.password"
            type="password"
            show-password
            :placeholder="isCreate ? t('pages.notification_channel.passwordPlaceholder') : t('pages.notification_channel.passwordKeepHint')"
          />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smtpFrom')" prop="smtpFrom">
          <ElInput v-model="drawer.formData.smtpFrom" placeholder="noreply@example.com" clearable />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smtpTls')" prop="smtpTls">
          <ElSelect v-model="drawer.formData.smtpTls">
            <ElOption :label="t('pages.notification_channel.tlsNone')" value="NONE" />
            <ElOption :label="t('pages.notification_channel.tlsStartTls')" value="START_TLS" />
            <ElOption :label="t('pages.notification_channel.tlsSsl')" value="SSL" />
          </ElSelect>
        </ElFormItem>
      </template>

      <ElFormItem :label="t('pages.notification_channel.enabled')" prop="enabled">
        <ElSwitch v-model="drawer.formData.enabled" />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_channel.remark')" prop="remark">
        <ElInput v-model="drawer.formData.remark" type="textarea" :rows="2" />
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
import { computed, ref } from "vue";
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
  ElSwitch,
} from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";
import { useI18n } from "@/core/i18n";
import {
  createNotificationChannel,
  updateNotificationChannel,
} from "@/api/composables";

const emit = defineEmits<{
  success: [];
}>();

const { t } = useI18n();

const formRef = ref();
const isCreate = ref(true);

const drawer = useDrawerForm({
  moduleKey: "pages.notification_channel.moduleName",
  defaults: {
    name: "",
    type: "EMAIL",
    smtpHost: "",
    smtpPort: 587,
    smtpUsername: "",
    password: "",
    smtpFrom: "",
    smtpTls: "START_TLS",
    webhookUrl: "",
    webhookSecret: "",
    enabled: true,
    remark: "",
  },
  // password / webhookSecret 是请求级字段，与 data 并列发出去：读视图只有 hasPassword /
  // hasWebhookSecret 两个布尔，所以它们既不进 data、也不进 updateMask。
  createFn: async (values: Record<string, any>) => {
    const { password, webhookSecret, ...data } = values;
    return createNotificationChannel(data, password, webhookSecret);
  },
  updateFn: (id: number, values: Record<string, any>) => {
    const { password, webhookSecret, ...data } = values;
    return updateNotificationChannel(id, data, password || undefined, webhookSecret || undefined);
  },
});

const formRules = computed(() => ({
  name: [{ required: true, message: t("pages.notification_channel.requiredName"), trigger: "blur" }],
  // SMTP 密码不填等于建了个发不出信的渠道，故创建时必填；签名密钥始终可选（不签名是合法
  // 配置），所以 WEBHOOK 组里只有 webhookUrl 在创建时必填。两类规则各按当前 type 生效。
  password:
    isCreate.value && drawer.formData.type === "EMAIL"
      ? [{ required: true, message: t("pages.notification_channel.requiredPassword"), trigger: "blur" }]
      : [],
  webhookUrl:
    isCreate.value && drawer.formData.type === "WEBHOOK"
      ? [
          {
            required: true,
            message: t("pages.notification_channel.requiredWebhookUrl"),
            trigger: "blur",
          },
        ]
      : [],
}));

// 包装 open：追踪创建/编辑模式（切换类型字段禁用态与密钥必填规则），
// 并在编辑时显式回填行数据（useDrawerForm 不做默认填充；hasPassword /
// hasWebhookSecret 为服务端计算字段、两处密钥均不回显，故不进表单）。
function open(options: { create: boolean; row?: any }) {
  isCreate.value = options.create;
  drawer.open(options, (row: any) => {
    Object.assign(drawer.formData, {
      name: row.name || "",
      type: row.type || "EMAIL",
      smtpHost: row.smtpHost || "",
      smtpPort: row.smtpPort ?? 587,
      smtpUsername: row.smtpUsername || "",
      smtpFrom: row.smtpFrom || "",
      smtpTls: row.smtpTls || "START_TLS",
      webhookUrl: row.webhookUrl || "",
      enabled: !!row.enabled,
      remark: row.remark || "",
    });
  });
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.field-tip {
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
