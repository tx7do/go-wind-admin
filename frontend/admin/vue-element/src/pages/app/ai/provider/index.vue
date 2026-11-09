<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
    >
      <!-- 部署形态 -->
      <template #modelType="scope: any">
        <ElTag v-if="scope.row.modelType === 'CLOUD'" type="primary" size="small">
          {{ t("pages.ai_provider.modelTypeCloud") }}
        </ElTag>
        <ElTag v-else-if="scope.row.modelType === 'LOCAL'" type="success" size="small">
          {{ t("pages.ai_provider.modelTypeLocal") }}
        </ElTag>
        <span v-else>-</span>
      </template>

      <!-- 端点：云端显示 baseUrl，本地显示 host:port -->
      <template #endpoint="scope: any">
        <span v-if="scope.row.modelType === 'CLOUD'">{{ scope.row.baseUrl || "-" }}</span>
        <span v-else-if="scope.row.modelType === 'LOCAL'">
          {{ scope.row.localHost || "localhost" }}:{{ scope.row.localPort ?? 11434 }}
        </span>
        <span v-else>-</span>
      </template>

      <!-- API Key 状态（密钥永不回显，只显示脱敏 hint） -->
      <template #apiKeyHint="scope: any">
        <span v-if="scope.row.apiKeyHint">{{ scope.row.apiKeyHint }}</span>
        <ElTag v-else size="small">{{ t("pages.ai_provider.notSet") }}</ElTag>
      </template>

      <!-- 启用状态 -->
      <template #isEnabled="scope: any">
        <ElTag v-if="scope.row.isEnabled" type="success" size="small">
          {{ t("pages.ai_provider.enabledOn") }}
        </ElTag>
        <ElTag v-else size="small">{{ t("pages.ai_provider.enabledOff") }}</ElTag>
      </template>
    </ProPage>

    <AiProviderDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElTag } from "element-plus";
import { useI18n } from "@/core/i18n";
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import AiProviderDrawer from "./ai-provider-drawer.vue";
import { PaginationQuery } from "@/core/transport/rest";
import type { aiservicev1_AiProvider as AiProvider } from "@/api/generated/admin/service/v1";
import { deleteAiProvider, fetchListAiProviders } from "@/api/composables";

const { t } = useI18n();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  exportFilename: "ai-providers",

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...rest } = query;
      const result = await fetchListAiProviders(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 20 },
          formValues: Object.keys(rest).length > 0 ? rest : undefined,
        }),
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteAiProvider(ids);
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    columns: [
      { type: "index", label: t("common.table.seq"), width: 60 },
      { prop: "name", label: t("pages.ai_provider.name"), minWidth: 140 },
      { prop: "modelType", label: t("pages.ai_provider.modelType"), width: 110, slotName: "modelType" },
      { prop: "modelName", label: t("pages.ai_provider.modelName"), minWidth: 150 },
      { prop: "endpoint", label: t("pages.ai_provider.endpoint"), minWidth: 200, slotName: "endpoint" },
      { prop: "apiKeyHint", label: t("pages.ai_provider.apiKey"), width: 120, slotName: "apiKeyHint" },
      { prop: "timeoutSeconds", label: t("pages.ai_provider.timeoutSeconds"), width: 110 },
      { prop: "isEnabled", label: t("pages.ai_provider.isEnabled"), width: 80, slotName: "isEnabled" },
      { prop: "remark", label: t("pages.ai_provider.remark"), minWidth: 120 },
      {
        prop: "action",
        label: t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", label: t("common.button.edit"), icon: "lucide:pen-line" },
          { name: "delete", label: t("common.button.delete"), icon: "lucide:trash-2", attrs: { type: "danger" } },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: AiProvider) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
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
