<template>
  <div class="app-container h-full flex flex-1 flex-col">
  <ProPage
    ref="pageRef"
    :config="pageConfig"
    @add="handleAdd"
    @edit="handleEdit"
    @operate="handleOperate"
  >
    <!-- 启用状态 -->
    <template #enabled="scope: any">
      <ElTag v-if="scope.row.enabled" type="success" size="small">
        {{ t("pages.ai_agent.enabledOn") }}
      </ElTag>
      <ElTag v-else size="small">{{ t("pages.ai_agent.enabledOff") }}</ElTag>
    </template>
  </ProPage>

  <!-- 创建/编辑抽屉：基本信息 + graph JSON（保存期后端强校验） -->
  <AiAgentDrawer ref="drawerRef" @success="handleSuccess" />

  <!-- 画布编辑器（vue-flow 版，与 react 基准同构） -->
  <AiAgentCanvas ref="canvasRef" @success="handleSuccess" />

  <!-- 试运行抽屉 -->
  <AiAgentRunDrawer ref="runRef" @success="handleSuccess" />

  <!-- 运行台账抽屉 -->
  <AiAgentRunsDrawer ref="runsRef" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import AiAgentDrawer from "./ai-agent-drawer.vue";
import AiAgentCanvas from "./ai-agent-canvas.vue";
import AiAgentRunDrawer from "./ai-agent-run-drawer.vue";
import AiAgentRunsDrawer from "./ai-agent-runs-drawer.vue";
import { useI18n } from "@/core/i18n";
import { PaginationQuery } from "@/core/transport/rest";
import { deleteAiAgentApp, fetchListAiAgentApps } from "@/api/composables";

const { t } = useI18n();

const pageRef = ref();
const drawerRef = ref();
const canvasRef = ref();
const runRef = ref();
const runsRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  exportFilename: "ai-agent-apps",

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...rest } = query;
      const result = await fetchListAiAgentApps(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 20 },
          formValues: Object.keys(rest).length > 0 ? rest : undefined,
        }),
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteAiAgentApp(Number(ids));
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
      { type: "index", label: t("common.table.seq"), width: 60 },
      { prop: "name", label: t("pages.ai_agent.name"), minWidth: 160 },
      { prop: "description", label: t("pages.ai_agent.description"), minWidth: 200 },
      { prop: "enabled", label: t("pages.ai_agent.enabled"), width: 90, slotName: "enabled" },
      { prop: "updatedAt", label: t("pages.ai_agent.updatedAt"), width: 170 },
      {
        prop: "action",
        label: t("common.table.action"),
        fixed: "right",
        width: 220,
        cellType: "tool",
        buttons: [
          { name: "canvas", label: t("pages.ai_agent.openCanvas"), icon: "lucide:network" },
          { name: "edit", label: t("common.button.edit"), icon: "lucide:pen-line" },
          { name: "run", label: t("pages.ai_agent.testRun"), icon: "lucide:play" },
          { name: "runs", label: t("pages.ai_agent.runsTitle"), icon: "lucide:history" },
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

function handleEdit(row: any) {
  drawerRef.value?.open({ create: false, row });
}

function handleOperate(data: { name: string; row: any }) {
  if (data.name === "canvas") canvasRef.value?.open(data.row);
  if (data.name === "run") runRef.value?.open(data.row);
  if (data.name === "runs") runsRef.value?.open(data.row);
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
