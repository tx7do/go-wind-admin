<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @operate="handleOperate"
    >
      <!-- 编码：等宽呈现，强调它是被引用的锚 -->
      <template #code="scope: any">
        <ElTag v-if="scope.row.code" size="small" round effect="plain">
          {{ scope.row.code }}
        </ElTag>
        <span v-else>-</span>
      </template>

      <!-- 启用：停用不是删除，发送方引用它会直接报错 -->
      <template #isEnabled="scope: any">
        <ElTag v-if="scope.row.isEnabled" size="small" round type="success">
          {{ t("pages.notification_template.enabledOn") }}
        </ElTag>
        <ElTag v-else size="small" round>
          {{ t("pages.notification_template.enabledOff") }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 创建/编辑抽屉 -->
    <NotificationTemplateDrawer ref="drawerRef" @success="handleSuccess" />

    <!-- 试渲染：变量集 JSON → 服务端渲染，结果就地展示（看到的字节即发送字节） -->
    <ElDialog
      v-model="renderVisible"
      :title="t('pages.notification_template.renderTitle', { name: renderTarget?.name ?? '' })"
      width="560px"
      align-center
      :close-on-click-modal="false"
      @closed="renderTarget = undefined; renderResult = undefined"
    >
      <ElForm label-width="120px">
        <ElFormItem :label="t('pages.notification_template.varsJson')">
          <ElInput
            v-model="varsJson"
            type="textarea"
            :rows="4"
            :placeholder="t('pages.notification_template.varsJsonPlaceholder')"
          />
          <div class="render-field-tip">{{ t("pages.notification_template.varsJsonHint") }}</div>
        </ElFormItem>
      </ElForm>
      <ElDescriptions v-if="renderResult" :column="1" size="small" border class="render-result">
        <ElDescriptionsItem :label="t('pages.notification_template.renderedTitle')">
          {{ renderResult.title }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="t('pages.notification_template.renderedContent')">
          <pre class="render-content">{{ renderResult.content }}</pre>
        </ElDescriptionsItem>
      </ElDescriptions>
      <template #footer>
        <ElButton @click="renderVisible = false">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="rendering" @click="handleRender">
          {{ t("pages.notification_template.renderSubmit") }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import {
  ElButton,
  ElDescriptions,
  ElDescriptionsItem,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElTag,
} from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import NotificationTemplateDrawer from "./notification-template-drawer.vue";
import type { notificationservicev1_NotificationTemplate as NotificationTemplate } from "@/api/generated/admin/service/v1";
import {
  fetchListNotificationTemplates,
  useDeleteNotificationTemplate,
  useRenderNotificationTemplate,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { useI18n } from "@/core/i18n";

const { t } = useI18n();

const pageRef = ref();
const drawerRef = ref();

const { mutateAsync: deleteTemplate } = useDeleteNotificationTemplate();
const { mutateAsync: renderTemplate } = useRenderNotificationTemplate();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: t("pages.notification_template.name"),
        field: "name",
        attrs: { placeholder: t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: t("pages.notification_template.code"),
        field: "code",
        attrs: { placeholder: t("common.placeholder.input"), clearable: true },
      },
    ],
  },
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListNotificationTemplates(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 20 },
          formValues: {
            name: queryParams.name,
            code: queryParams.code,
          },
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
      {
        prop: "name",
        label: t("pages.notification_template.name"),
        minWidth: 150,
      },
      {
        prop: "code",
        label: t("pages.notification_template.code"),
        width: 180,
        slotName: "code",
      },
      {
        prop: "titleTemplate",
        label: t("pages.notification_template.titleTemplate"),
        minWidth: 220,
        formatter: (row: NotificationTemplate) => row.titleTemplate || "-",
      },
      {
        prop: "contentTemplate",
        label: t("pages.notification_template.contentTemplate"),
        minWidth: 280,
        formatter: (row: NotificationTemplate) => row.contentTemplate || "-",
      },
      {
        prop: "isEnabled",
        label: t("pages.notification_template.isEnabled"),
        width: 90,
        slotName: "isEnabled",
      },
      {
        prop: "updatedAt",
        label: t("pages.notification_template.updatedAt"),
        width: 170,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: t("common.table.action"),
        fixed: "right",
        width: 170,
        cellType: "tool",
        buttons: [
          { name: "edit", label: t("common.button.edit"), icon: "lucide:pen-line" },
          { name: "render", label: t("pages.notification_template.render"), icon: "lucide:eye" },
          // 删除按钮刻意不叫 "delete"：正在被引用的模板删掉后发送方直接报错，后果必须说清
          {
            name: "remove",
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

function handleEdit(row: NotificationTemplate) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}

async function handleOperate(data: { name: string; row: NotificationTemplate }) {
  if (data.name === "render") {
    renderTarget.value = data.row;
    varsJson.value = "";
    renderResult.value = undefined;
    renderVisible.value = true;
    return;
  }
  if (data.name !== "remove") return;

  const row = data.row;
  if (!row.id) return;

  try {
    await ElMessageBox.confirm(
      t("pages.notification_template.deleteConfirm"),
      t("common.title.confirm"),
      {
        confirmButtonText: t("common.button.confirm"),
        cancelButtonText: t("common.button.cancel"),
        type: "warning",
        lockScroll: false,
      }
    );
  } catch (error) {
    // 取消是 ElMessageBox 的 reject("cancel")，属正常路径；其余形态（API 误用等）留痕
    if (error !== "cancel" && error !== "close") {
      console.error("delete confirm dialog rejected unexpectedly", error);
    }
    return;
  }

  try {
    await deleteTemplate({ id: row.id });
    ElMessage.success(t("pages.notification_template.deleteSuccess"));
    pageRef.value?.refresh();
  } catch (error: any) {
    // 用户看到的那句翻译不含服务端原因，原始错误必须留在控制台
    console.error("delete notification template failed", error);
    ElMessage.error(error?.message || t("pages.notification_template.deleteFailed"));
  }
}

// === 试渲染 ===
const renderVisible = ref(false);
const rendering = ref(false);
const renderTarget = ref<NotificationTemplate>();
const varsJson = ref("");
const renderResult = ref<{ title: string; content: string }>();

async function handleRender() {
  if (!renderTarget.value?.id) return;

  let variables: Record<string, string> = {};
  const raw = (varsJson.value || "").trim();
  if (raw) {
    try {
      variables = JSON.parse(raw);
    } catch (error: any) {
      console.error("parse template vars json failed", error);
      ElMessage.error(t("pages.notification_template.varsJsonInvalid"));
      return;
    }
  }

  rendering.value = true;
  try {
    // 自定义 RPC（body:"*"）收**扁平**请求体：包一层 { data } 会被 protojson
    // 当未知字段丢掉，接口照样 200、字段全为空
    const resp = await renderTemplate({ id: renderTarget.value.id, variables });
    // 结果就地展示，弹窗保持打开，用户看完手动关闭
    renderResult.value = { title: resp.title ?? "", content: resp.content ?? "" };
  } catch (error: any) {
    console.error("render notification template failed", error);
    ElMessage.error(error?.message || t("pages.notification_template.renderFailed"));
  } finally {
    rendering.value = false;
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

.render-field-tip {
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.render-result {
  margin-top: 12px;
}

.render-content {
  margin: 0;
  white-space: pre-wrap;
  max-height: 240px;
  overflow: auto;
  font-family: inherit;
}
</style>
