<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @operate="handleOperate"
    >
      <!-- 文档数 -->
      <template #docCount="scope: any">
        <ElTag size="small" type="primary">{{ scope.row.docCount ?? 0 }}</ElTag>
      </template>
    </ProPage>

    <AiKnowledgeDrawer ref="drawerRef" @success="handleSuccess" />
    <DocsModal v-model:visible="docsVisible" :base="docsTarget" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElTag } from "element-plus";
import { useI18n } from "@/core/i18n";
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import AiKnowledgeDrawer from "./ai-knowledge-drawer.vue";
import DocsModal from "./docs-modal.vue";
import { PaginationQuery } from "@/core/transport/rest";
import type { aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from "@/api/generated/admin/service/v1";
import { deleteAiKnowledgeBase, fetchListAiKnowledgeBases } from "@/api/composables";

const { t } = useI18n();

const pageRef = ref();
const drawerRef = ref();
const docsVisible = ref(false);
const docsTarget = ref<AiKnowledgeBase>();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  exportFilename: "ai-knowledge-bases",

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...rest } = query;
      const result = await fetchListAiKnowledgeBases(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 20 },
          formValues: Object.keys(rest).length > 0 ? rest : undefined,
        }),
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteAiKnowledgeBase(Number(ids));
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    columns: [
      { type: "index", label: t("common.table.seq"), width: 60 },
      { prop: "name", label: t("pages.ai_knowledge.name"), minWidth: 160 },
      { prop: "description", label: t("pages.ai_knowledge.description"), minWidth: 180 },
      { prop: "embeddingModel", label: t("pages.ai_knowledge.embeddingModel"), minWidth: 180 },
      { prop: "docCount", label: t("pages.ai_knowledge.docCount"), width: 90, slotName: "docCount" },
      {
        prop: "action",
        label: t("pages.ai_knowledge.action"),
        fixed: "right",
        width: 170,
        cellType: "tool",
        buttons: [
          { name: "docs", label: t("pages.ai_knowledge.docs"), icon: "lucide:file-text" },
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

function handleEdit(row: AiKnowledgeBase) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}

// 「文档」按钮走 @operate（非 edit/delete 内置名）
function handleOperate(data: { name: string; row: AiKnowledgeBase }) {
  if (data.name !== "docs") return;
  docsTarget.value = data.row;
  docsVisible.value = true;
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
