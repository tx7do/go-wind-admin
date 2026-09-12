<template>
  <div class="server-monitor-page app-container">
    <template v-if="isLoading">
      <el-card shadow="hover" class="mb-4">
        <div class="loading-wrap"><el-skeleton :rows="3" animated /></div>
      </el-card>
    </template>
    <template v-else>
      <el-card shadow="hover" class="mb-4">
        <template #header>
          <div class="card-header">
            <span class="card-title">{{ $t("pages.server_monitor.database") }}</span>
            <el-tag v-if="db?.pingOk" type="success" size="small">{{ $t("pages.server_monitor.pingOk") }}</el-tag>
            <el-tag v-else type="danger" size="small">{{ $t("pages.server_monitor.pingFail") }}</el-tag>
          </div>
        </template>
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="$t('pages.server_monitor.dbDriver')">{{ db?.driver || "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.maxOpenConnections')">{{ db?.maxOpenConnections ?? "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.openConnections')">{{ db?.openConnections ?? "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.inUseConnections')">{{ db?.inUseConnections ?? "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.idleConnections')">{{ db?.idleConnections ?? "-" }}</el-descriptions-item>
          <el-descriptions-item v-if="db?.pingError" :label="$t('pages.server_monitor.pingError')">
            <span class="error-text">{{ db?.pingError }}</span>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card shadow="hover" class="mb-4">
        <template #header>
          <span class="card-title">{{ $t("pages.server_monitor.goRuntime") }}</span>
        </template>
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="$t('pages.server_monitor.goVersion')">{{ go?.version || "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.numGoroutine')">{{ go?.numGoroutine ?? "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.memAlloc')">{{ fmtBytes(go?.memAllocBytes) }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.memSys')">{{ fmtBytes(go?.memSysBytes) }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.gcCycles')">{{ go?.gcCycles ?? "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.uptime')">{{ fmtUptime(go?.uptimeSeconds) }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card shadow="hover">
        <template #header>
          <span class="card-title">{{ $t("pages.server_monitor.host") }}</span>
        </template>
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="$t('pages.server_monitor.hostname')">{{ host?.hostname || "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.os')">{{ host ? (host.os || "-") + " / " + (host.arch || "-") : "-" }}</el-descriptions-item>
          <el-descriptions-item :label="$t('pages.server_monitor.numCpu')">{{ host?.numCpu ?? "-" }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <div class="disclaimer">{{ $t("pages.server_monitor.disclaimer") }}</div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useServerMonitorInfo } from "@/api/composables";
import type { server_monitorservicev1_ServerMonitorInfo } from "@/api/generated/admin/service/v1";

const { data, isLoading } = useServerMonitorInfo();

const info = computed(
  () => (data.value as server_monitorservicev1_ServerMonitorInfo | undefined) ?? undefined
);
const go = computed(() => info.value?.go);
const db = computed(() => info.value?.database);
const host = computed(() => info.value?.host);

function fmtBytes(bytes?: number | string): string {
  const n = Number(bytes ?? 0);
  if (!Number.isFinite(n)) return "-";
  if (n >= 1024 ** 3) return (n / 1024 ** 3).toFixed(2) + " GB";
  if (n >= 1024 ** 2) return (n / 1024 ** 2).toFixed(2) + " MB";
  if (n >= 1024) return (n / 1024).toFixed(1) + " KB";
  return String(n) + " B";
}

function fmtUptime(seconds?: number | string): string {
  const s = Number(seconds ?? 0);
  if (!Number.isFinite(s)) return "-";
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m ${Math.floor(s % 60)}s`;
}
</script>

<style lang="scss" scoped>
.server-monitor-page {
  // 留白由 app-container 统一施加（§2.6），此处不再自带 padding
  width: 100%;
  min-width: 0;
}
.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.card-title {
  font-weight: 600;
}
.mb-4 {
  margin-bottom: 16px;
}
.loading-wrap {
  padding: 12px 4px;
}
.error-text {
  color: var(--el-color-danger);
}
.disclaimer {
  margin-top: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
