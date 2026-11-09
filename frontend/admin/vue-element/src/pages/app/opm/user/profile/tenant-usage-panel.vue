<template>
  <div class="tenant-usage-panel">
    <template v-if="isLoading">
      <div class="panel-loading">{{ t("pages.task.tenantUsageLoading") }}</div>
    </template>
    <template v-else-if="!usage || !usage.quotas || usage.quotas.length === 0">
      <div class="panel-empty">{{ t("pages.task.tenantUsageEmpty") }}</div>
    </template>
    <template v-else>
      <p class="plan-line">
        <span class="plan-label">{{ t("pages.task.tenantUsagePlan") }}</span>
        <strong>{{ usage.planName || t("pages.task.tenantUsageNoPlan") }}</strong>
      </p>
      <div v-for="(q, idx) in usage.quotas" :key="idx" class="quota-row">
        <div class="quota-head">
          <span>{{ quotaLabel(q.quotaType) }}</span>
          <span class="quota-numbers">
            {{ currentOf(q.quotaType) }}{{ unitOf(q.quotaType) }} / {{ num(q.quotaValue) }}{{ unitOf(q.quotaType) }}
            <ElTag v-if="pctOf(q.quotaType, q.quotaValue) >= 100" size="small" type="danger">
              {{ t("pages.task.tenantUsageReached") }}
            </ElTag>
          </span>
        </div>
        <ElProgress
          :percentage="pctOf(q.quotaType, q.quotaValue)"
          :status="pctOf(q.quotaType, q.quotaValue) >= 100 ? 'exception' : pctOf(q.quotaType, q.quotaValue) >= 80 ? 'warning' : 'success'"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ElProgress, ElTag } from "element-plus";
import { useMyTenantUsage } from "@/api/composables/my-tenant-usage";
import { useI18n } from "@/core/i18n";

/**
 * 租户套餐用量面板（自取数）：用户数/存储占用 vs 配额上限的进度条。
 * 64-bit 计数经 protojson 是字符串，一律 Number() 转换（文档已知坑）。
 */
const { t } = useI18n();

const { data: usage, isLoading } = useMyTenantUsage();

const num = (v: number | string | undefined): number => Number(v ?? 0);

const currentOf = (quotaType?: string): number => {
  const u = usage.value;
  if (!u) return 0;
  switch (quotaType) {
    case "USER_LIMIT": return num(u.userCount);
    case "STORAGE": return Math.round(num(u.storageUsedBytes) / 1024 / 1024);
    case "API_CALL": return num(u.apiCallCount);
    default: return 0;
  }
};

const limitOf = (quotaValue?: number | string): number => num(quotaValue);

const pctOf = (quotaType?: string, quotaValue?: number | string): number => {
  const limit = limitOf(quotaValue);
  if (limit <= 0) return 0;
  return Math.min(100, Math.round((currentOf(quotaType) / limit) * 100));
};

const unitOf = (quotaType?: string): string => (quotaType === "STORAGE" ? "MB" : "");

const quotaLabel = (quotaType?: string): string => {
  switch (quotaType) {
    case "USER_LIMIT": return t("pages.task.quota_USER_LIMIT");
    case "STORAGE": return t("pages.task.quota_STORAGE");
    case "API_CALL": return t("pages.task.quota_API_CALL");
    case "AI_TOKENS": return t("pages.task.quota_AI_TOKENS");
    default: return quotaType ?? "-";
  }
};
</script>

<style lang="scss" scoped>
.tenant-usage-panel {
  max-width: 560px;
}

.panel-loading,
.panel-empty {
  padding: 32px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}

.plan-line {
  margin: 0 0 20px;
  font-size: 14px;

  .plan-label {
    color: var(--el-text-color-secondary);
    margin-right: 8px;
  }
}

.quota-row {
  margin-bottom: 20px;

  .quota-head {
    display: flex;
    justify-content: space-between;
    margin-bottom: 6px;
    font-size: 13px;

    .quota-numbers {
      color: var(--el-text-color-secondary);
    }
  }
}
</style>
