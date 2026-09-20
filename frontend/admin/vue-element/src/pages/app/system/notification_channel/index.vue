<template>
  <div class="app-container h-full flex flex-1 flex-col">
  <ProPage
    ref="pageRef"
    :config="pageConfig"
    @add="handleAdd"
    @edit="handleEdit"
    @operate="handleOperate"
  >
    <!-- 渠道类型：按 row.type 渲染，不写死成邮件 -->
    <template #type="scope: any">
      <ElTag v-if="scope.row.type" size="small" :type="typeTagType(scope.row.type)">
        {{ typeLabel(scope.row.type) }}
      </ElTag>
      <span v-else>-</span>
    </template>

    <!-- 密码配置状态 -->
    <template #hasPassword="scope: any">
      <ElTag v-if="scope.row.hasPassword" type="success" size="small">
        {{ t("pages.notification_channel.passwordSet") }}
      </ElTag>
      <ElTag v-else size="small">{{ t("pages.notification_channel.passwordNotSet") }}</ElTag>
    </template>

    <!-- 签名密钥配置状态：与密码同一条约定，读侧只有布尔、密钥永不回显 -->
    <template #hasWebhookSecret="scope: any">
      <ElTag v-if="scope.row.hasWebhookSecret" type="success" size="small">
        {{ t("pages.notification_channel.passwordSet") }}
      </ElTag>
      <ElTag v-else size="small">{{ t("pages.notification_channel.passwordNotSet") }}</ElTag>
    </template>

    <!-- 启用状态 -->
    <template #enabled="scope: any">
      <ElTag v-if="scope.row.enabled" type="success" size="small">
        {{ t("pages.notification_channel.enabledOn") }}
      </ElTag>
      <ElTag v-else size="small">{{ t("pages.notification_channel.enabledOff") }}</ElTag>
    </template>
  </ProPage>

  <!-- 创建/编辑抽屉 -->
  <NotificationChannelDrawer ref="drawerRef" @success="handleSuccess" />

  <!-- 测试发送对话框 -->
  <ElDialog
    v-model="testVisible"
    :title="t('pages.notification_channel.testSendTitle', { name: testTarget?.name || '' })"
    width="480px"
    align-center
    :close-on-click-modal="false"
    @closed="testTarget = undefined"
  >
    <ElForm ref="testFormRef" :model="testForm" :rules="testRules" label-width="120px">
      <ElFormItem :label="t('pages.notification_channel.testRecipient')" prop="recipient">
        <ElInput v-model="testForm.recipient" placeholder="you@example.com" clearable />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="testVisible = false">{{ $t("common.button.cancel") }}</ElButton>
      <ElButton type="primary" :loading="testing" @click="handleTestSend">
        {{ t("pages.notification_channel.testSend") }}
      </ElButton>
    </template>
  </ElDialog>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElTag,
} from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import NotificationChannelDrawer from "./notification-channel-drawer.vue";
import { useI18n } from "@/core/i18n";
import { PaginationQuery } from "@/core/transport/rest";
import type { notification_channelservicev1_NotificationChannel as NotificationChannel } from "@/api/generated/admin/service/v1";
import {
  createPagedExportAction,
  deleteNotificationChannel,
  fetchListNotificationChannels,
  sendTestEmail,
} from "@/api/composables";

const { t } = useI18n();

const pageRef = ref();
const drawerRef = ref();

const tlsLabels: Record<string, string> = {
  NONE: t("pages.notification_channel.tlsNone"),
  START_TLS: t("pages.notification_channel.tlsStartTls"),
  SSL: t("pages.notification_channel.tlsSsl"),
};

function tlsLabel(mode?: string): string {
  return (mode && tlsLabels[mode]) || mode || "-";
}

// 渠道类型按枚举查表渲染：一个类型对应一列自己的配置（SMTP 那几列对 WEBHOOK 没有意义，
// 反之 webhookUrl 对邮件渠道为空），混着显示会读成"没配"。
const typeLabels: Record<string, string> = {
  EMAIL: t("pages.notification_channel.typeEmail"),
  WEBHOOK: t("pages.notification_channel.typeWebhook"),
};

const typeTagTypes: Record<string, "primary" | "info"> = {
  EMAIL: "primary",
  WEBHOOK: "info",
};

function typeLabel(type?: string): string {
  return (type && typeLabels[type]) || type || "-";
}

function typeTagType(type?: string): "primary" | "info" {
  return (type && typeTagTypes[type]) || "info";
}

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  exportFilename: "notification-channels",

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...rest } = query;
      const result = await fetchListNotificationChannels(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 20 },
          formValues: Object.keys(rest).length > 0 ? rest : undefined,
        }),
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteNotificationChannel(Number(ids));
    },
    exportsAction: createPagedExportAction(fetchListNotificationChannels),
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "exports", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
      { type: "index", label: t("common.table.seq"), width: 60 },
      { prop: "name", label: t("pages.notification_channel.name"), minWidth: 140 },
      { prop: "type", label: t("pages.notification_channel.type"), width: 120, slotName: "type" },
      { prop: "smtpHost", label: t("pages.notification_channel.smtpHost"), minWidth: 150 },
      { prop: "smtpPort", label: t("pages.notification_channel.smtpPort"), width: 90 },
      { prop: "smtpFrom", label: t("pages.notification_channel.smtpFrom"), minWidth: 170 },
      {
        prop: "smtpTls",
        label: t("pages.notification_channel.smtpTls"),
        width: 110,
        formatter: (row: any) => tlsLabel(row.smtpTls),
      },
      {
        prop: "webhookUrl",
        label: t("pages.notification_channel.webhookUrl"),
        minWidth: 200,
        // 邮件渠道这一列恒空：显示 "-" 而不是空白，免得读成"配了但看不到"
        formatter: (row: any) => row.webhookUrl || "-",
      },
      { prop: "hasPassword", label: t("pages.notification_channel.hasPassword"), width: 100, slotName: "hasPassword" },
      {
        prop: "hasWebhookSecret",
        label: t("pages.notification_channel.hasWebhookSecret"),
        width: 100,
        slotName: "hasWebhookSecret",
      },
      { prop: "enabled", label: t("pages.notification_channel.enabled"), width: 90, slotName: "enabled" },
      { prop: "remark", label: t("pages.notification_channel.remark"), minWidth: 140 },
      {
        prop: "action",
        label: t("common.table.action"),
        fixed: "right",
        width: 210,
        cellType: "tool",
        buttons: [
          { name: "edit", label: t("common.button.edit"), icon: "lucide:pen-line" },
          {
            name: "test",
            label: t("pages.notification_channel.testSend"),
            icon: "lucide:send",
            // 测试发送只对 EMAIL 有意义：它测的就是这一个 SMTP 账号能否握手发信。
            // WEBHOOK 的"当场试一次"在路由规则页（那里测的是事件 → 渠道 → 台账整条链）。
            visible: (row: any) => row.type === "EMAIL",
          },
          {
            name: "delete",
            label: t("common.button.delete"),
            icon: "lucide:trash-2",
            attrs: { type: "danger" },
          },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: NotificationChannel) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}

function handleOperate(data: { name: string; row: NotificationChannel }) {
  if (data.name !== "test") return;
  testTarget.value = data.row;
  testForm.value.recipient = "";
  testVisible.value = true;
}

// === 测试发送 ===
const testVisible = ref(false);
const testing = ref(false);
const testFormRef = ref();
const testTarget = ref<NotificationChannel>();
const testForm = ref({ recipient: "" });

const testRules = {
  recipient: [
    { required: true, message: t("pages.notification_channel.requiredRecipient"), trigger: "blur" },
    { type: "email" as const, message: t("pages.notification_channel.invalidEmail"), trigger: "blur" },
  ],
};

async function handleTestSend() {
  if (!testTarget.value?.id) return;
  const valid = await testFormRef.value
    ?.validate()
    .then(() => true, () => false);
  if (!valid) return;
  testing.value = true;
  try {
    await sendTestEmail(testTarget.value.id, testForm.value.recipient);
    ElMessage.success(t("pages.notification_channel.testSendSuccess"));
    testVisible.value = false;
  } catch (error: any) {
    console.error("send test email failed:", error);
    ElMessage.error(error?.message || t("pages.notification_channel.testSendFailed"));
  } finally {
    testing.value = false;
  }
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
