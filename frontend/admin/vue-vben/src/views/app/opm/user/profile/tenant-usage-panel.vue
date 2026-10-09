<script lang="ts" setup>
import { computed } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { Progress, Table, Tag } from 'ant-design-vue';

import { useMyTenantUsage } from '#/api/composables/my-tenant-usage';
/**
 * 租户套餐用量面板（自取数）：用户数/存储占用 vs 配额上限的进度条。
 * 64-bit 计数经 protojson 是字符串，一律 Number() 转换（文档已知坑）。
 */
const { data: usage, isLoading } = useMyTenantUsage();



const num = (v: number | string | undefined): number => Number(v ?? 0);

const currentOf = (quotaType?: string): number => {
  const u = usage.value;
  if (!u) return 0;
  switch (quotaType) {
    case 'USER_LIMIT': return num(u.userCount);
    case 'STORAGE': return Math.round(num(u.storageUsedBytes) / 1024 / 1024);
    case 'API_CALL': return num(u.apiCallCount);
    default: return 0;
  }
};

const limitOf = (quotaValue?: number | string): number => num(quotaValue);

const pctOf = (quotaType?: string, quotaValue?: number | string): number => {
  const limit = limitOf(quotaValue);
  if (limit <= 0) return 0;
  return Math.min(100, Math.round((currentOf(quotaType) / limit) * 100));
};

const unitOf = (quotaType?: string): string =>
  quotaType === 'STORAGE' ? 'MB' : '';

const quotaLabel = (quotaType?: string): string => {
  switch (quotaType) {
    case 'USER_LIMIT': return $t('page.task.quota_USER_LIMIT');
    case 'STORAGE': return $t('page.task.quota_STORAGE');
    case 'API_CALL': return $t('page.task.quota_API_CALL');
    case 'AI_TOKENS': return $t('page.task.quota_AI_TOKENS');
    default: return quotaType ?? '-';
  }
};

const columns = [
  {
    title: $t('page.task.sysTaskType'),
    dataIndex: 'quotaType',
    key: 'quotaType',
  },
  { title: $t('page.task.tenantUsageLimit'), dataIndex: 'limit', key: 'limit', width: 160 },
  { title: $t('page.task.tenantUsageCurrent'), dataIndex: 'current', key: 'current', width: 160 },
  { title: $t('page.task.tenantUsageProgress'), key: 'progress', width: 200 },
];

const rows = computed(() => {
  const u = usage.value;
  if (!u?.quotas) return [];
  return u.quotas.map((q) => ({
    quotaType: q.quotaType,
    quotaValue: q.quotaValue,
    current: currentOf(q.quotaType),
    limit: limitOf(q.quotaValue),
    pct: pctOf(q.quotaType, q.quotaValue),
    unit: unitOf(q.quotaType),
  }));
});
</script>

<template>
  <Page>
    <div class="tenant-usage-panel">
      <h3 class="section-title">{{ $t('page.task.tenantUsageTitle') }}</h3>
      <div v-if="isLoading" class="panel-loading">
        {{ $t('page.task.tenantUsageLoading') }}
      </div>
      <div v-else-if="!usage?.quotas || usage.quotas.length === 0" class="panel-empty">
        {{ $t('page.task.tenantUsageEmpty') }}
      </div>
      <template v-else>
        <p class="plan-line">
          <span class="plan-label">{{ $t('page.task.tenantUsagePlan') }}</span>
          <strong>{{ usage.planName || $t('page.task.tenantUsageNoPlan') }}</strong>
        </p>
        <Table
          :columns="columns"
          :data-source="rows"
          :loading="isLoading"
          :pagination="false"
          row-key="quotaType"
          size="small"
          bordered
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'quotaType'">
              <Tag color="geekblue">{{ quotaLabel(record.quotaType) }}</Tag>
            </template>
            <template v-else-if="column.key === 'limit'">
              {{ record.limit }}{{ record.unit }}
            </template>
            <template v-else-if="column.key === 'current'">
              {{ record.current }}{{ record.unit }}
              <Tag v-if="record.pct >= 100" color="error">
                {{ $t('page.task.tenantUsageReached') }}
              </Tag>
            </template>
            <template v-else-if="column.key === 'progress'">
              <Progress
                :percent="record.pct"
                :status="record.pct >= 100 ? 'exception' : record.pct >= 80 ? 'active' : 'normal'"
                size="small"
              />
            </template>
          </template>
        </Table>
      </template>
    </div>
  </Page>
</template>

<style lang="scss" scoped>
.tenant-usage-panel {
  max-width: 640px;
}

.section-title {
  margin: 0 0 12px;
  font-size: 15px;
  font-weight: 600;
}

.panel-loading,
.panel-empty {
  padding: 32px 0;
  text-align: center;
  opacity: 0.7;
}

.plan-line {
  margin: 0 0 12px;
  font-size: 14px;

  .plan-label {
    margin-right: 8px;
    opacity: 0.7;
  }
}
</style>
