<script lang="ts" setup>
import { computed } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { formatDateTime } from '@vben/utils';

import {
  Card,
  Descriptions,
  DescriptionsItem,
  Empty,
  Spin,
  Tag,
} from 'ant-design-vue';

import { useServerMonitorInfo } from '#/api';
import type {
  server_monitorservicev1_ServerMonitorInfo as ServerMonitorInfo,
} from '#/api/generated/admin/service/v1';

/**
 * 服务监控页面（只读）：Go 运行时 / 数据库连接池 / 主机信息。
 * 数据来自 GET /admin/v1/server-monitor（composable 内置 10s 轮询），无任何写操作。
 * 布局对齐 react 基准：三张全宽卡片纵向堆叠，Descriptions 两列。
 */
const { data, isLoading } = useServerMonitorInfo();

const info = computed(() => (data.value ?? undefined) as ServerMonitorInfo | undefined);
const go = computed(() => info.value?.go);
const db = computed(() => info.value?.database);
const host = computed(() => info.value?.host);

function fmtBytes(bytes?: number | string): string {
  const n = Number(bytes ?? 0);
  if (!Number.isFinite(n)) return '-';
  if (n >= 1024 ** 3) return `${(n / 1024 ** 3).toFixed(2)} GB`;
  if (n >= 1024 ** 2) return `${(n / 1024 ** 2).toFixed(2)} MB`;
  if (n >= 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${n} B`;
}

function fmtUptime(seconds?: number | string): string {
  const s = Number(seconds ?? 0);
  if (!Number.isFinite(s)) return '-';
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m ${Math.floor(s % 60)}s`;
}
</script>

<template>
  <Page auto-content-height :title="$t('page.serverMonitor.moduleName')">
    <div v-if="isLoading" class="flex justify-center p-12">
      <Spin size="large" />
    </div>
    <template v-else>
      <!-- 数据库连接池 -->
      <Card size="small" class="mb-4" :title="$t('page.serverMonitor.database')">
        <template #extra>
          <Tag v-if="db?.pingOk === true" color="success">
            {{ $t('page.serverMonitor.pingOk') }}
          </Tag>
          <Tag v-else-if="db?.pingOk === false" color="error">
            {{ $t('page.serverMonitor.pingFail') }}
          </Tag>
        </template>
        <Descriptions v-if="db" :column="2" size="small" bordered>
          <DescriptionsItem :label="$t('page.serverMonitor.dbDriver')">
            {{ db.driver || '-' }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.maxOpenConnections')">
            {{ db.maxOpenConnections ?? '-' }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.openConnections')">
            {{ db.openConnections ?? '-' }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.inUseConnections')">
            {{ db.inUseConnections ?? '-' }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.idleConnections')">
            {{ db.idleConnections ?? '-' }}
          </DescriptionsItem>
          <DescriptionsItem v-if="db.pingError" :label="$t('page.serverMonitor.pingError')">
            <span class="text-destructive">{{ db.pingError }}</span>
          </DescriptionsItem>
        </Descriptions>
        <Empty v-else :description="$t('page.serverMonitor.noData')" />
      </Card>

      <!-- Go 运行时 -->
      <Card size="small" class="mb-4" :title="$t('page.serverMonitor.goRuntime')">
        <Descriptions v-if="go" :column="2" size="small" bordered>
          <DescriptionsItem :label="$t('page.serverMonitor.goVersion')">
            {{ go.version || '-' }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.numGoroutine')">
            {{ go.numGoroutine ?? '-' }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.memAlloc')">
            {{ fmtBytes(go.memAllocBytes) }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.memSys')">
            {{ fmtBytes(go.memSysBytes) }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.gcCycles')">
            {{ go.gcCycles ?? '-' }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.uptime')">
            {{ fmtUptime(go.uptimeSeconds) }}
            <span v-if="go.startedAt">({{ formatDateTime(go.startedAt) }})</span>
          </DescriptionsItem>
        </Descriptions>
        <Empty v-else :description="$t('page.serverMonitor.noData')" />
      </Card>

      <!-- 主机信息 -->
      <Card size="small" class="mb-4" :title="$t('page.serverMonitor.host')">
        <Descriptions v-if="host" :column="2" size="small" bordered>
          <DescriptionsItem :label="$t('page.serverMonitor.hostname')">
            {{ host.hostname || '-' }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.os')">
            {{ `${host.os || '-'} / ${host.arch || '-'}` }}
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.serverMonitor.numCpu')">
            {{ host.numCpu ?? '-' }}
          </DescriptionsItem>
        </Descriptions>
        <Empty v-else :description="$t('page.serverMonitor.noData')" />
      </Card>

      <p class="mt-4 text-sm text-muted-foreground">
        {{ $t('page.serverMonitor.disclaimer') }}
        <span v-if="info?.collectedAt">({{ formatDateTime(info.collectedAt) }})</span>
      </p>
    </template>
  </Page>
</template>
