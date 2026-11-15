<template>
  <div class="redis-cache-monitor-page">
    <template v-if="isLoading">
      <el-card shadow="hover" class="mb-4">
        <el-empty :description="$t('common.loading')" />
      </el-card>
    </template>
    <template v-else>
      <el-card shadow="hover" class="mb-4">
        <template #header>
          <span class="card-title">{{ $t("pages.redis_cache_monitor.dbSizeCardTitle") }}</span>
        </template>
        <el-descriptions :column="1" border>
          <el-descriptions-item :label="$t('pages.redis_cache_monitor.dbSize')">
            <el-tag :type="dbSize > 0 ? 'warning' : 'info'">{{ String(dbSize) }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <template v-if="sections.length === 0">
        <el-card shadow="hover" class="mb-4">
          <template #header>
            <span class="card-title">{{ $t("pages.redis_cache_monitor.infoCardTitle") }}</span>
          </template>
          <el-empty :description="$t('pages.redis_cache_monitor.noInfo')" />
        </el-card>
      </template>
      <template v-else>
        <el-card
          v-for="(section, idx) in sections"
          :key="`${section.name}-${idx}`"
          shadow="hover"
          class="mb-4"
        >
          <template #header>
            <div class="section-header" @click="toggleSection(`${section.name}-${idx}`)">
              <span class="card-title">{{ section.name }}</span>
              <el-tag>{{ section.entries.length }}</el-tag>
              <span class="toggle-hint">
                {{
                  expandedSections[`${section.name}-${idx}`]
                    ? $t("pages.redis_cache_monitor.collapse")
                    : $t("pages.redis_cache_monitor.expand")
                }}
              </span>
            </div>
          </template>
          <template v-if="expandedSections[`${section.name}-${idx}`]">
            <template v-if="section.entries.length === 0">
              <el-empty :description="$t('pages.redis_cache_monitor.noEntries')" />
            </template>
            <template v-else>
              <el-descriptions :column="1" border>
                <el-descriptions-item
                  v-for="(entry, eIdx) in section.entries"
                  :key="eIdx"
                  :label="entry.key"
                >
                  <span class="entry-value">{{ entry.value }}</span>
                </el-descriptions-item>
              </el-descriptions>
            </template>
          </template>
          <!-- 收起态用轻量提示行，不用 el-empty 插画——empty 语义是“无数据”，
             大片收起时会让整页看起来像加载失败/空表 -->
          <template v-else>
            <div class="collapsed-hint">{{ $t("pages.redis_cache_monitor.collapsed") }}</div>
          </template>
        </el-card>
      </template>

      <el-card shadow="hover">
        <template #header>
          <span class="card-title">{{ $t("pages.redis_cache_monitor.slowlogCardTitle") }}</span>
        </template>
        <el-table
          :data="slowlog"
          border
          stripe
          :empty-text="$t('pages.redis_cache_monitor.noSlowlog')"
        >
          <el-table-column
            prop="id"
            :label="$t('pages.redis_cache_monitor.slowlogId')"
            width="80"
          />
          <el-table-column :label="$t('pages.redis_cache_monitor.createdAt')" width="180">
            <template #default="{ row }">
              {{ formatDateTime(row.createdAt ?? "") }}
            </template>
          </el-table-column>
          <el-table-column
            prop="durationUsec"
            :label="$t('pages.redis_cache_monitor.durationUsec')"
            width="120"
          />
          <el-table-column
            prop="clientAddr"
            :label="$t('pages.redis_cache_monitor.clientAddr')"
            width="200"
          />
          <el-table-column
            prop="clientName"
            :label="$t('pages.redis_cache_monitor.clientName')"
            width="120"
          />
          <el-table-column :label="$t('pages.redis_cache_monitor.args')">
            <template #default="{ row }">
              <span class="args-cell">{{ (row.args ?? []).join(" ") || "-" }}</span>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <div class="disclaimer">{{ $t("pages.redis_cache_monitor.disclaimer") }}</div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { formatDateTime } from "@/utils";
import { $t } from "@go-wind/admin-element";
import { useRedisCacheMonitorInfo } from "@/api/composables";
import type {
  redis_cacheservicev1_RedisCacheMonitorInfo,
  redis_cacheservicev1_SlowLogEntry,
} from "@/api/generated/admin/service/v1";

const { data, isLoading } = useRedisCacheMonitorInfo();

const expandedSections = ref<Record<string, boolean>>({});
function toggleSection(key: string) {
  expandedSections.value = { ...expandedSections.value, [key]: !expandedSections.value[key] };
}

// 通过 computed 派生，使数据返回后视图能响应式更新（data 为 useQuery 的响应式 ref）
const info = computed(
  () => (data.value as redis_cacheservicev1_RedisCacheMonitorInfo | undefined) ?? undefined
);
const sections = computed(
  () =>
    (info.value?.sections ?? []) as {
      name: string;
      entries: { key: string; value: string }[];
    }[]
);
const dbSize = computed(() => info.value?.dbSize ?? 0);
const slowlog = computed(() => (info.value?.slowlog ?? []) as redis_cacheservicev1_SlowLogEntry[]);
</script>

<style lang="scss" scoped>
.redis-cache-monitor-page {
  padding: 16px;
}
.card-title {
  font-weight: 600;
}
.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.toggle-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: auto;
}
.collapsed-hint {
  padding: 4px 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.entry-value {
  word-break: break-all;
}
.args-cell {
  word-break: break-all;
  white-space: pre-wrap;
}
.disclaimer {
  margin-top: 16px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
