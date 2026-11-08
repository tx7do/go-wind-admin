<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @operate="handleOperate"
    >
      <!-- 指标 -->
      <template #metric="scope: any">
        <ElTag v-if="scope.row.metric" size="small" round effect="plain">
          {{ t(`pages.monitor_alert.metricMap.${scope.row.metric}`) }}
        </ElTag>
        <span v-else>-</span>
      </template>

      <!-- 触发条件：布尔指标单独说人话 -->
      <template #condition="scope: any">
        <span v-if="scope.row.metric === 'DB_PING_FAIL'">
          {{ t("pages.monitor_alert.pingFailCondition") }}
        </span>
        <span v-else>
          {{ t(`pages.monitor_alert.opMap.${scope.row.op}`) }} {{ scope.row.threshold ?? "-" }}
        </span>
      </template>

      <!-- 当前状态：来自最近一轮自动评估（每 5 分钟） -->
      <template #lastFiring="scope: any">
        <ElTag v-if="scope.row.lastFiring" size="small" round type="danger">
          {{ t("pages.monitor_alert.firingNow") }}
        </ElTag>
        <ElTag v-else size="small" round type="success">
          {{ t("pages.monitor_alert.firingNo") }}
        </ElTag>
      </template>

      <!-- 渠道 -->
      <template #channel="scope: any">
        <ElTag v-if="scope.row.channel" size="small" round>
          {{ t(`pages.monitor_alert.channelMap.${scope.row.channel}`) }}
        </ElTag>
        <span v-else>-</span>
      </template>

      <!-- 启用：停用不是删除，是不再评估 -->
      <template #isEnabled="scope: any">
        <ElTag v-if="scope.row.isEnabled" size="small" round type="success">
          {{ t("pages.monitor_alert.enabledOn") }}
        </ElTag>
        <ElTag v-else size="small" round type="info">
          {{ t("pages.monitor_alert.enabledOff") }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 创建/编辑抽屉 -->
    <MonitorAlertDrawer ref="drawerRef" @success="handleSuccess" />

    <!-- 立即评估：与后台每 5 分钟的自动扫描同一内核，结果逐规则展示 -->
    <ElDialog
      v-model="evaluateVisible"
      :title="t('pages.monitor_alert.evaluateResultTitle')"
      width="640px"
      align-center
      :close-on-click-modal="false"
    >
      <div v-if="evaluateOutcomes.length === 0" class="evaluate-empty">
        {{ t("pages.monitor_alert.evaluateEmpty") }}
      </div>
      <ElTable v-else :data="evaluateOutcomes" size="small" border max-height="420">
        <ElTableColumn prop="name" :label="t('pages.monitor_alert.name')" min-width="140" />
        <ElTableColumn :label="t('pages.monitor_alert.evaluateState')" width="90">
          <template #default="{ row }">
            <ElTag v-if="row.firing" size="small" type="danger">
              {{ t("pages.monitor_alert.firingNow") }}
            </ElTag>
            <ElTag v-else size="small" type="success">
              {{ t("pages.monitor_alert.firingNo") }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('pages.monitor_alert.evaluateValue')" width="110">
          <template #default="{ row }">
            {{ row.currentValue != null ? Number(row.currentValue).toFixed(2) : "-" }}
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('pages.monitor_alert.evaluateNotified')" width="90">
          <template #default="{ row }">
            {{ row.notified ? t("pages.monitor_alert.notifiedYes") : "-" }}
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="reason"
          :label="t('pages.monitor_alert.evaluateReason')"
          min-width="180"
          show-overflow-tooltip
        />
      </ElTable>
      <template #footer>
        <ElButton @click="evaluateVisible = false">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="evaluating" @click="handleEvaluate">
          {{ t("pages.monitor_alert.evaluateNow") }}
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
  ElMessage,
  ElMessageBox,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import MonitorAlertDrawer from "./monitor-alert-drawer.vue";
import type { monitor_alertservicev1_MonitorAlertRule as MonitorAlertRule } from "@/api/generated/admin/service/v1";
import {
  fetchListMonitorAlertRules,
  useDeleteMonitorAlertRule,
  useEvaluateMonitorAlerts,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { useI18n } from "@/core/i18n";

const { t } = useI18n();

const pageRef = ref();
const drawerRef = ref();

const { mutateAsync: deleteRule } = useDeleteMonitorAlertRule();
const { mutateAsync: evaluateAlerts } = useEvaluateMonitorAlerts();

const evaluateVisible = ref(false);
const evaluating = ref(false);
const evaluateOutcomes = ref<any[]>([]);

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: t("pages.monitor_alert.name"),
        field: "name",
        attrs: { placeholder: t("common.placeholder.input"), clearable: true },
      },
    ],
  },
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListMonitorAlertRules(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 20 },
          formValues: {
            name: queryParams.name,
          },
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [
      // 自定义工具栏按钮：走 @operate 分发（name !== edit/delete/view/add 即透传）
      {
        name: "evaluate",
        label: t("pages.monitor_alert.evaluateNow"),
        icon: "lucide:play",
      },
    ],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
      {
        prop: "name",
        label: t("pages.monitor_alert.name"),
        minWidth: 150,
      },
      {
        prop: "metric",
        label: t("pages.monitor_alert.metric"),
        width: 150,
        slotName: "metric",
      },
      {
        prop: "condition",
        label: t("pages.monitor_alert.condition"),
        width: 140,
        slotName: "condition",
      },
      {
        prop: "cooldownMinutes",
        label: t("pages.monitor_alert.cooldownMinutes"),
        width: 100,
        formatter: (row: MonitorAlertRule) => `${row.cooldownMinutes ?? 30} min`,
      },
      {
        prop: "channel",
        label: t("pages.monitor_alert.channel"),
        width: 90,
        slotName: "channel",
      },
      {
        prop: "target",
        label: t("pages.monitor_alert.target"),
        minWidth: 200,
        formatter: (row: MonitorAlertRule) => row.target || "-",
      },
      {
        prop: "lastFiring",
        label: t("pages.monitor_alert.lastFiring"),
        width: 90,
        slotName: "lastFiring",
      },
      {
        prop: "lastAlertedAt",
        label: t("pages.monitor_alert.lastAlertedAt"),
        width: 170,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "isEnabled",
        label: t("pages.monitor_alert.isEnabled"),
        width: 80,
        slotName: "isEnabled",
      },
      {
        prop: "action",
        label: t("common.table.action"),
        fixed: "right",
        width: 120,
        cellType: "tool",
        buttons: [
          { name: "edit", label: t("common.button.edit"), icon: "lucide:pen-line" },
          // 删除按钮刻意不叫 "delete"：删掉的规则不再评估，后果必须说清
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

function handleEdit(row: MonitorAlertRule) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}

async function handleOperate(data: { name: string; row?: MonitorAlertRule }) {
  if (data.name === "evaluate") {
    evaluateVisible.value = true;
    evaluateOutcomes.value = [];
    await handleEvaluate();
    return;
  }
  if (data.name !== "remove") return;

  const row = data.row;
  if (!row?.id) return;

  try {
    await ElMessageBox.confirm(
      t("pages.monitor_alert.deleteConfirm"),
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
    await deleteRule({ id: row.id });
    ElMessage.success(t("pages.monitor_alert.deleteSuccess"));
    pageRef.value?.refresh();
  } catch (error: any) {
    // 用户看到的那句翻译不含服务端原因，原始错误必须留在控制台
    console.error("delete monitor alert rule failed", error);
    ElMessage.error(error?.message || t("pages.monitor_alert.deleteFailed"));
  }
}

// === 立即评估 ===
async function handleEvaluate() {
  evaluating.value = true;
  try {
    // body:"*" 自定义路由，请求体扁平：包一层 { data } 会被 protojson 当未知字段丢掉
    const resp = await evaluateAlerts({});
    evaluateOutcomes.value = resp.outcomes ?? [];
  } catch (error: any) {
    console.error("evaluate monitor alerts failed", error);
    ElMessage.error(error?.message || t("pages.monitor_alert.evaluateFailed"));
  } finally {
    evaluating.value = false;
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

.evaluate-empty {
  padding: 24px 0;
  text-align: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
